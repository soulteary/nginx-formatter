package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/define"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

// captureStdout runs fn with os.Stdout redirected and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	savedOut := updater.Out
	updater.Out = w

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	os.Stdout = saved
	updater.Out = savedOut
	_ = w.Close()
	return <-done
}

// TestQuietSuppressesProgress covers the banner and progress chatter that went
// to stdout on every run, so anything piping stdout got it mixed in with the
// result. Errors are deliberately not suppressed: they travel back as values
// and are printed on stderr.
func TestQuietSuppressesProgress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.conf")
	if err := os.WriteFile(path, []byte("a {\nb;\n}\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Run("loud by default", func(t *testing.T) {
		quiet = false
		out := captureStdout(t, func() {
			applyQuiet()
			if err := runFormat(path, "", 2, " "); err != nil {
				t.Errorf("runFormat: %v", err)
			}
		})
		if !strings.Contains(out, "Successed") {
			t.Errorf("expected progress output, got %q", out)
		}
	})

	t.Run("silent with --quiet", func(t *testing.T) {
		if err := os.WriteFile(path, []byte("a {\nb;\n}\n"), 0600); err != nil {
			t.Fatalf("write: %v", err)
		}
		quiet = true
		defer func() {
			quiet = false
			updater.Out = os.Stdout
		}()

		out := captureStdout(t, func() {
			// Inside the capture: applyQuiet installs io.Discard, which the
			// redirect above would otherwise overwrite.
			applyQuiet()
			if err := runFormat(path, "", 2, " "); err != nil {
				t.Errorf("runFormat: %v", err)
			}
		})
		if out != "" {
			t.Errorf("expected no output with --quiet, got %q", out)
		}

		// The work still happened.
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if !strings.Contains(string(got), "  b;") {
			t.Errorf("--quiet skipped the formatting: %q", got)
		}
	})
}

// TestQuietFlagIsRegistered keeps the flag reachable from every subcommand.
func TestQuietFlagIsRegistered(t *testing.T) {
	root := newRootCmd()
	if root.PersistentFlags().Lookup("quiet") == nil {
		t.Fatal("--quiet is not registered on the root command")
	}
	if root.PersistentFlags().ShorthandLookup("q") == nil {
		t.Error("-q shorthand is not registered")
	}
}

// TestQuietSuppressesPortWarning pins the one place where this feature and the
// port-boundary fix meet. resolvePort's rejection notice is progress output on
// stdout like any other, so --quiet has to cover it too; the fix that corrected
// the bounds and the message landed separately and printed with fmt.Printf.
func TestQuietSuppressesPortWarning(t *testing.T) {
	t.Run("loud by default", func(t *testing.T) {
		quiet = false
		var got int
		out := captureStdout(t, func() {
			applyQuiet()
			got = resolvePort(80)
		})
		if got != define.DEFAULT_PORT {
			t.Errorf("resolvePort(80) = %d, want the default %d", got, define.DEFAULT_PORT)
		}
		if !strings.Contains(out, "Please set the port") {
			t.Errorf("expected the rejection notice, got %q", out)
		}
	})

	t.Run("silent with --quiet", func(t *testing.T) {
		quiet = true
		defer func() {
			quiet = false
			updater.Out = os.Stdout
		}()

		var got int
		out := captureStdout(t, func() {
			applyQuiet()
			got = resolvePort(80)
		})
		if out != "" {
			t.Errorf("expected no output with --quiet, got %q", out)
		}
		// Silencing the notice must not change the decision.
		if got != define.DEFAULT_PORT {
			t.Errorf("resolvePort(80) = %d, want the default %d", got, define.DEFAULT_PORT)
		}
	})
}

// TestQuietCoversEveryProgressPath walks the routes that reach stdout without
// going through the write that TestQuietSuppressesProgress exercises. Each was
// added by a different change and each had kept its own fmt.Printf, so --quiet
// covered the path it was written against and nothing else.
//
// The already-formatted branch matters most: it is the steady state. Run the
// formatter twice in CI and the second run takes it for every file, so a leak
// there means --quiet is silent exactly once and noisy from then on.
func TestQuietCoversEveryProgressPath(t *testing.T) {
	// tree builds one input directory hitting all of them at once: a file that
	// is already formatted, a file carrying a byte order mark, a symbolic link
	// the scan steps over, and a nested output directory it excludes.
	tree := func(t *testing.T) (src, out string) {
		t.Helper()
		src = t.TempDir()
		out = filepath.Join(src, "out")
		for _, f := range []struct{ name, body string }{
			{"done.conf", "server {\n  listen 80;\n}\n"},
			{"bom.conf", "\ufeffserver {\n  listen 81;\n}\n"},
		} {
			if err := os.WriteFile(filepath.Join(src, f.name), []byte(f.body), 0600); err != nil {
				t.Fatalf("write %s: %v", f.name, err)
			}
		}
		if err := os.MkdirAll(filepath.Join(src, "out"), 0750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(out, "nested.conf"), []byte("server {\n  listen 82;\n}\n"), 0600); err != nil {
			t.Fatalf("write nested: %v", err)
		}
		if err := os.Symlink("done.conf", filepath.Join(src, "link.conf")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		return src, out
	}

	run := func(t *testing.T, quietOn bool, fn func()) string {
		t.Helper()
		quiet = quietOn
		defer func() {
			quiet = false
			updater.Out = os.Stdout
		}()
		return captureStdout(t, func() {
			// applyQuiet has to run inside the capture: it installs io.Discard,
			// which the redirect would otherwise overwrite.
			applyQuiet()
			fn()
		})
	}

	t.Run("separate output directory", func(t *testing.T) {
		src, out := tree(t)
		if got := run(t, true, func() {
			if err := runFormat(src, out, 2, " "); err != nil {
				t.Errorf("runFormat: %v", err)
			}
		}); got != "" {
			t.Errorf("--quiet leaked the exclusion and BOM notices:\n%s", got)
		}
	})

	t.Run("in place, twice", func(t *testing.T) {
		src, _ := tree(t)
		for _, pass := range []string{"first", "second"} {
			got := run(t, true, func() {
				if err := runFormat(src, "", 2, " "); err != nil {
					t.Errorf("runFormat: %v", err)
				}
			})
			if got != "" {
				t.Errorf("--quiet leaked on the %s pass:\n%s", pass, got)
			}
		}
	})

	t.Run("single file, already formatted", func(t *testing.T) {
		src, _ := tree(t)
		path := filepath.Join(src, "done.conf")
		if got := run(t, true, func() {
			if err := runFormat(path, "", 2, " "); err != nil {
				t.Errorf("runFormat: %v", err)
			}
		}); got != "" {
			t.Errorf("--quiet leaked the already-formatted line:\n%s", got)
		}
	})

	// Without the flag the same run still narrates, so this cannot be passed by
	// silencing the tool outright.
	t.Run("loud by default", func(t *testing.T) {
		src, _ := tree(t)
		got := run(t, false, func() {
			if err := runFormat(src, "", 2, " "); err != nil {
				t.Errorf("runFormat: %v", err)
			}
		})
		for _, want := range []string{"symbolic link", "UTF-8 BOM", "Successed"} {
			if !strings.Contains(got, want) {
				t.Errorf("default output is missing %q:\n%s", want, got)
			}
		}
	})
}

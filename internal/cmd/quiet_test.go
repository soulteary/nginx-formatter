package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

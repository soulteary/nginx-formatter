package cmd

import (
	"errors"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/define"
)

func TestResolveOutputDefault(t *testing.T) {
	t.Run("src is a file and output empty returns empty", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "nginx.conf")
		if err := os.WriteFile(src, []byte("events {}"), 0600); err != nil {
			t.Fatalf("write conf: %v", err)
		}

		got, err := resolveOutputDefault(src, "")
		if err != nil {
			t.Fatalf("resolveOutputDefault: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty output for file input, got %q", got)
		}
	})

	// A directory with no -o is formatted in place, matching single-file mode.
	// Returning the working directory instead (the old behaviour) left the
	// named directory untouched and scattered a copy of the tree into cwd,
	// overwriting same-named files there.
	t.Run("src is a directory and output empty returns src", func(t *testing.T) {
		dir := t.TempDir()
		got, err := resolveOutputDefault(dir, "")
		if err != nil {
			t.Fatalf("resolveOutputDefault: %v", err)
		}
		if got != dir {
			t.Errorf("expected input dir %q for directory input, got %q", dir, got)
		}
	})

	t.Run("output not empty returns as-is", func(t *testing.T) {
		got, err := resolveOutputDefault(t.TempDir(), "/some/output/dir")
		if err != nil {
			t.Fatalf("resolveOutputDefault: %v", err)
		}
		if got != "/some/output/dir" {
			t.Errorf("expected output returned as-is, got %q", got)
		}
	})
}

func TestResolveIndent(t *testing.T) {
	if got := resolveIndent(0); got != define.DEFAULT_INDENT_SIZE {
		t.Errorf("expected default indent %d, got %d", define.DEFAULT_INDENT_SIZE, got)
	}
	if got := resolveIndent(-5); got != define.DEFAULT_INDENT_SIZE {
		t.Errorf("expected default indent %d, got %d", define.DEFAULT_INDENT_SIZE, got)
	}
	if got := resolveIndent(4); got != 4 {
		t.Errorf("expected indent 4, got %d", got)
	}
}

func TestResolveIndentChar(t *testing.T) {
	cases := map[string]string{
		"":   define.DEFAULT_INDENT_CHAR,
		" ":  " ",
		"\t": "\t",
		// The two escape spellings the README and --help advertise must resolve
		// to the real characters rather than being passed through verbatim:
		// printer.go repeats this value, so "\\s" would be written into the
		// config as literal backslash-s indentation.
		"\\s":   " ",
		"\\t":   "\t",
		"space": " ",
		"tab":   "\t",
		"bogus": define.DEFAULT_INDENT_CHAR,
	}
	for in, want := range cases {
		if got := resolveIndentChar(in); got != want {
			t.Errorf("resolveIndentChar(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolvePort(t *testing.T) {
	rejected := []int{-1, 0, 80, 443, 1024, 65536, 70000}
	for _, p := range rejected {
		if got := resolvePort(p); got != define.DEFAULT_PORT {
			t.Errorf("resolvePort(%d) = %d, want the default %d", p, got, define.DEFAULT_PORT)
		}
	}

	// 1025 and 65535 are the edges of the accepted range. 65535 used to be
	// rejected by a ">= 65535" guard, contradicting the message that promised
	// everything "within 65535".
	accepted := []int{1025, 8123, 65535}
	for _, p := range accepted {
		if got := resolvePort(p); got != p {
			t.Errorf("resolvePort(%d) = %d, want it accepted", p, got)
		}
	}
}

// TestRootCommandStructure ensures the semantic subcommands are mounted and the
// legacy flags are present (as hidden compatibility flags) on the root.
func TestRootCommandStructure(t *testing.T) {
	root := newRootCmd()

	for _, name := range []string{"format", "serve", "version"} {
		sub, _, err := root.Find([]string{name})
		if err != nil || sub == nil || sub.Name() != name {
			t.Fatalf("expected subcommand %q to be registered, err=%v", name, err)
		}
	}

	for _, name := range []string{
		define.APP_ARGV_INPUT, define.APP_ARGV_OUTPUT, define.APP_ARGV_INDENT,
		define.APP_ARGV_CHAR, define.APP_ARGV_WEB, define.APP_ARGV_PORT,
	} {
		f := root.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("expected legacy flag %q on root", name)
		}
		if !f.Hidden {
			t.Errorf("expected legacy flag %q to be hidden", name)
		}
	}
}

// TestFormatSubcommandShortFlags verifies the new canonical short flags parse.
func TestFormatSubcommandShortFlags(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "nginx.conf")
	if err := os.WriteFile(src, []byte("events {\nworker_connections 1024;\n}\n"), 0600); err != nil {
		t.Fatalf("write conf: %v", err)
	}
	out := filepath.Join(dir, "dist")
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatalf("mkdir out: %v", err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"format", "-i", src, "-o", out, "-n", "4", "-c", "space"})
	if err := root.Execute(); err != nil {
		t.Fatalf("format execute: %v", err)
	}

	formatted := filepath.Join(out, "nginx.conf")
	if _, err := os.Stat(formatted); err != nil {
		t.Fatalf("expected formatted output at %q: %v", formatted, err)
	}
}

// TestLegacyInputFileRouting verifies that the old single-dash long flag
// `-input=<file>` still routes to the format logic.
func TestLegacyInputFileRouting(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "nginx.conf")
	original := "events {\nworker_connections 1024;\n}\n"
	if err := os.WriteFile(src, []byte(original), 0600); err != nil {
		t.Fatalf("write conf: %v", err)
	}

	root := newRootCmd()
	root.SetArgs(normalizeLegacyArgs([]string{"-input=" + src}))
	if err := root.Execute(); err != nil {
		t.Fatalf("legacy execute: %v", err)
	}

	// The file should have been overwritten in place.
	got, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected formatted content, got empty file")
	}
}

// TestVersionSubcommand verifies the version command runs without error.
func TestVersionSubcommand(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version execute: %v", err)
	}
}

// TestNormalizeLegacyArgs verifies single-dash legacy long flags are rewritten
// to their pflag-compatible double-dash form, while other args are untouched.
func TestNormalizeLegacyArgs(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"-input=/app", "-web", "-port=8123"}, []string{"--input=/app", "--web", "--port=8123"}},
		{[]string{"-input", "./nginx.conf"}, []string{"--input", "./nginx.conf"}},
		{[]string{"format", "-i", ".", "-n", "4"}, []string{"format", "-i", ".", "-n", "4"}},
		{[]string{"--input=/app"}, []string{"--input=/app"}},
		{[]string{"serve", "-p", "8123"}, []string{"serve", "-p", "8123"}},
	}
	for _, c := range cases {
		got := normalizeLegacyArgs(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("normalizeLegacyArgs(%v) = %v, want %v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("normalizeLegacyArgs(%v) = %v, want %v", c.in, got, c.want)
			}
		}
	}
}

// TestResolveIndentCharEscapeForms is the regression guard for the documented
// "\s" / "\t" spellings. "\s" used to pass validation unchanged and reach
// printer.go's strings.Repeat, writing literal backslash-s pairs into the
// user's config as indentation while the CLI reported "[SPACE]".
func TestResolveIndentCharEscapeForms(t *testing.T) {
	if got := resolveIndentChar(`\s`); got != " " {
		t.Errorf(`resolveIndentChar("\\s") = %q, want a real space`, got)
	}
	if got := resolveIndentChar(`\t`); got != "\t" {
		t.Errorf(`resolveIndentChar("\\t") = %q, want a real tab`, got)
	}
}

// TestFormatPositionalPath covers the silently-discarded positional argument.
// `nginx-formatter format /etc/nginx` used to parse the path, drop it, and
// recursively reformat the *working directory* instead, with exit code 0.
func TestFormatPositionalPath(t *testing.T) {
	t.Run("positional path is used as the input", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "x.conf")
		if err := os.WriteFile(target, []byte("a {\nb;\n}\n"), 0600); err != nil {
			t.Fatalf("write: %v", err)
		}

		cmd := newFormatCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{dir})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if !strings.Contains(string(got), "  b;") {
			t.Errorf("the named path was not formatted: %q", got)
		}
	})

	t.Run("rejects a positional path alongside --input", func(t *testing.T) {
		cmd := newFormatCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"./a", "--input", "./b"})
		if err := cmd.Execute(); err == nil {
			t.Error("expected an error when both forms are given")
		}
	})

	t.Run("rejects more than one positional path", func(t *testing.T) {
		cmd := newFormatCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"./a", "./b"})
		if err := cmd.Execute(); err == nil {
			t.Error("expected an error for two positional paths")
		}
	})
}

// TestPositionalPathWithCheckAndDiff covers the place where the positional
// path and the read-only modes meet. Both arrived independently, and the
// obvious way to combine them is wrong in one specific direction: select the
// mode before resolving the path and --check inspects the *working directory*
// while the argument the user typed is ignored — the same silent-wrong-target
// bug the positional path was added to fix, back again in a mode whose whole
// job is to report accurately.
func TestPositionalPathWithCheckAndDiff(t *testing.T) {
	newTree := func(t *testing.T) (dir, target string) {
		t.Helper()
		dir = t.TempDir()
		target = filepath.Join(dir, "x.conf")
		if err := os.WriteFile(target, []byte("a {\nb;\n}\n"), 0600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return dir, target
	}

	for _, flag := range []string{"--check", "--diff"} {
		t.Run(flag+" reports the named path", func(t *testing.T) {
			dir, target := newTree(t)
			before, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			cmd := newFormatCmd()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{dir, flag})
			if err := cmd.Execute(); !errors.Is(err, updater.ErrNeedsFormatting) {
				t.Fatalf("expected ErrNeedsFormatting for the unformatted file at the named path, got %v", err)
			}

			// Read-only stays read-only, positional path or not.
			after, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if string(after) != string(before) {
				t.Errorf("%s rewrote the file:\nbefore: %q\nafter:  %q", flag, before, after)
			}
		})
	}

	t.Run("--check and --diff cannot be combined", func(t *testing.T) {
		dir, _ := newTree(t)
		cmd := newFormatCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{dir, "--check", "--diff"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected an error")
		}
		if errors.Is(err, updater.ErrNeedsFormatting) {
			t.Errorf("the flag combination was accepted and the run proceeded: %v", err)
		}
	})
}

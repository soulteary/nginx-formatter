package cmd

import (
	"os"
	"path/filepath"
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
	if got := resolvePort(80); got != define.DEFAULT_PORT {
		t.Errorf("expected default port for low value, got %d", got)
	}
	if got := resolvePort(70000); got != define.DEFAULT_PORT {
		t.Errorf("expected default port for high value, got %d", got)
	}
	if got := resolvePort(8123); got != 8123 {
		t.Errorf("expected port 8123, got %d", got)
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

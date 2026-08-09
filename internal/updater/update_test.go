package updater_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

func TestScanFiles(t *testing.T) {
	if _, err := updater.ScanFiles(""); err == nil {
		t.Error("expected error for empty root dir")
	}

	dir := t.TempDir()
	confPath := filepath.Join(dir, "nginx.conf")
	if err := os.WriteFile(confPath, []byte("events {}"), 0600); err != nil {
		t.Fatalf("write conf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("nope"), 0600); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	files, err := updater.ScanFiles(dir)
	if err != nil {
		t.Fatalf("scan files: %v", err)
	}
	if len(files) != 1 || filepath.Clean(files[0]) != "nginx.conf" {
		t.Errorf("unexpected scan result: %v", files)
	}
}

func TestFixReturn(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"number and variable arg", "return 200 $content;", `return 200 "$content";`},
		{"number and bare arg", "return 302 /foo;", `return 302 "/foo";`},
		{"number and quoted arg unchanged", `return 200 "ok";`, `return 200 "ok";`},
		{"plain number unchanged", "return 200;", "return 200;"},
		{"quoted string unchanged", `return "ok";`, `return "ok";`},
		{"bare identifier unchanged", "return BACKEND;", "return BACKEND;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := updater.FixReturn(tc.in); got != tc.want {
				t.Errorf("FixReturn(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUpdateConfInDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// A config that would have broken the old goja/template approach because
	// of backticks and ${var}; the AST formatter must handle it verbatim.
	input := "http {\nlocation / {\nreturn 200 \"${scheme}://`host`\";\n}\n}"
	if err := os.WriteFile(filepath.Join(src, "site.conf"), []byte(input), 0600); err != nil {
		t.Fatalf("write conf: %v", err)
	}

	err := updater.UpdateConfInDir(src, dst, 4, " ", formatter.Formatter)
	if err != nil {
		t.Fatalf("update conf in dir: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dst, "site.conf"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	expected := "http {\n    location / {\n        return 200 \"${scheme}://`host`\";\n    }\n\n}"
	if string(out) != expected {
		t.Errorf("unexpected output.\n got: %q\nwant: %q", string(out), expected)
	}
}

func TestUpdateConfFile(t *testing.T) {
	input := "http {\nlocation / {\nreturn 200 \"ok\";\n}\n}"
	expected := "http {\n    location / {\n        return 200 \"ok\";\n    }\n\n}"

	t.Run("in place overwrite when output empty", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "nginx.conf")
		if err := os.WriteFile(src, []byte(input), 0600); err != nil {
			t.Fatalf("write conf: %v", err)
		}

		if err := updater.UpdateConfFile(src, "", 4, " ", formatter.Formatter); err != nil {
			t.Fatalf("update conf file: %v", err)
		}

		out, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if string(out) != expected {
			t.Errorf("unexpected output.\n got: %q\nwant: %q", string(out), expected)
		}
	})

	t.Run("output is an existing directory", func(t *testing.T) {
		dir := t.TempDir()
		dst := t.TempDir()
		src := filepath.Join(dir, "site.conf")
		if err := os.WriteFile(src, []byte(input), 0600); err != nil {
			t.Fatalf("write conf: %v", err)
		}

		if err := updater.UpdateConfFile(src, dst, 4, " ", formatter.Formatter); err != nil {
			t.Fatalf("update conf file: %v", err)
		}

		out, err := os.ReadFile(filepath.Join(dst, "site.conf"))
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if string(out) != expected {
			t.Errorf("unexpected output.\n got: %q\nwant: %q", string(out), expected)
		}
	})

	t.Run("output is a file path with missing parent dir", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "site.conf")
		if err := os.WriteFile(src, []byte(input), 0600); err != nil {
			t.Fatalf("write conf: %v", err)
		}

		target := filepath.Join(t.TempDir(), "nested", "deeper", "result.conf")
		if err := updater.UpdateConfFile(src, target, 4, " ", formatter.Formatter); err != nil {
			t.Fatalf("update conf file: %v", err)
		}

		out, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if string(out) != expected {
			t.Errorf("unexpected output.\n got: %q\nwant: %q", string(out), expected)
		}
	})

	t.Run("non conf suffix is formatted", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "nginx.txt")
		if err := os.WriteFile(src, []byte(input), 0600); err != nil {
			t.Fatalf("write txt: %v", err)
		}

		if err := updater.UpdateConfFile(src, "", 4, " ", formatter.Formatter); err != nil {
			t.Fatalf("update conf file: %v", err)
		}

		out, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if string(out) != expected {
			t.Errorf("unexpected output.\n got: %q\nwant: %q", string(out), expected)
		}
	})
}

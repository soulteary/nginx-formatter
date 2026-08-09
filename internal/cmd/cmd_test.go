package cmd

import (
	"os"
	"path/filepath"
	"testing"
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

	t.Run("src is a directory and output empty returns cwd", func(t *testing.T) {
		got, err := resolveOutputDefault(t.TempDir(), "")
		if err != nil {
			t.Fatalf("resolveOutputDefault: %v", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if got != cwd {
			t.Errorf("expected cwd %q for directory input, got %q", cwd, got)
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

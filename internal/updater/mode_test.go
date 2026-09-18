package updater_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestCheckModeDoesNotWrite is the guarantee a CI job depends on: --check
// reports and exits non-zero, but leaves the tree exactly as it found it.
func TestCheckModeDoesNotWrite(t *testing.T) {
	for _, mode := range []updater.Mode{updater.ModeCheck, updater.ModeDiff} {
		dir := t.TempDir()
		path := filepath.Join(dir, "x.conf")
		const unformatted = "server {\nlisten 80;\n}\n"
		writeFile(t, path, unformatted)

		err := updater.UpdateConfFileMode(path, "", 2, " ", mode, formatter.Formatter)
		if !errors.Is(err, updater.ErrNeedsFormatting) {
			t.Errorf("mode %v: expected ErrNeedsFormatting, got %v", mode, err)
		}

		got, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read: %v", readErr)
		}
		if string(got) != unformatted {
			t.Errorf("mode %v rewrote the file: %q", mode, got)
		}
	}
}

// TestCheckModeCleanTree reports success when there is nothing to do.
func TestCheckModeCleanTree(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "x.conf"), "server {\n  listen 80;\n}\n")

	if err := updater.UpdateConfInDirMode(dir, dir, 2, " ", updater.ModeCheck, formatter.Formatter); err != nil {
		t.Errorf("a formatted tree should check clean, got %v", err)
	}
}

// TestCheckModeCreatesNoOutputDirectory guards the side effect a read-only
// mode must not have.
func TestCheckModeCreatesNoOutputDirectory(t *testing.T) {
	src := t.TempDir()
	out := filepath.Join(t.TempDir(), "does-not-exist")
	writeFile(t, filepath.Join(src, "x.conf"), "server {\nlisten 80;\n}\n")

	err := updater.UpdateConfInDirMode(src, out, 2, " ", updater.ModeCheck, formatter.Formatter)
	if !errors.Is(err, updater.ErrNeedsFormatting) {
		t.Fatalf("expected ErrNeedsFormatting, got %v", err)
	}
	if _, statErr := os.Stat(out); statErr == nil {
		t.Error("--check created the output directory")
	}
}

// TestCheckModeReportsEveryUnformattedFile makes sure the walk does not stop
// at the first one — CI wants the whole list.
func TestCheckModeReportsEveryUnformattedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.conf"), "server {\nlisten 80;\n}\n")
	writeFile(t, filepath.Join(dir, "b.conf"), "server {\n  listen 81;\n}\n")
	writeFile(t, filepath.Join(dir, "c.conf"), "server {\nlisten 82;\n}\n")

	out := captureStdout(t, func() {
		if err := updater.UpdateConfInDirMode(dir, dir, 2, " ", updater.ModeCheck, formatter.Formatter); !errors.Is(err, updater.ErrNeedsFormatting) {
			t.Errorf("expected ErrNeedsFormatting, got %v", err)
		}
	})
	for _, want := range []string{"a.conf", "c.conf"} {
		if !strings.Contains(out, want) {
			t.Errorf("%s not reported:\n%s", want, out)
		}
	}
	if strings.Contains(out, "b.conf") {
		t.Errorf("the already-formatted b.conf was reported:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			sb.Write(buf[:n])
			if readErr != nil {
				break
			}
		}
		done <- sb.String()
	}()
	fn()
	os.Stdout = saved
	_ = w.Close()
	return <-done
}

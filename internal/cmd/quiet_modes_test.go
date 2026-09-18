package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/updater"
)

// TestQuietNeverSwallowsAResult is the seam between --quiet and the
// machine-readable modes, and the one place where the obvious implementation
// is wrong.
//
// Both features suppress output, so it is tempting to run everything through
// the same writer. But they suppress different things: --quiet removes
// narration, while --check, --diff and stdin exist *to produce* output. Route
// their result through updater.Out and `--check --quiet` prints nothing at all
// and reports only through the exit code -- it silently discards the output
// the user asked for, which is a worse failure than the noise --quiet was
// added to remove.
func TestQuietNeverSwallowsAResult(t *testing.T) {
	tree := func(t *testing.T) (dir, target string) {
		t.Helper()
		dir = t.TempDir()
		target = filepath.Join(dir, "bad.conf")
		if err := os.WriteFile(target, []byte("server {\nlisten 80;\n}\n"), 0600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return dir, target
	}

	// withQuiet runs fn with --quiet in force and returns its stdout.
	withQuiet := func(t *testing.T, fn func()) string {
		t.Helper()
		quiet = true
		defer func() {
			quiet = false
			updater.Out = os.Stdout
		}()
		return captureStdout(t, func() {
			applyQuiet()
			fn()
		})
	}

	t.Run("--check still lists the file", func(t *testing.T) {
		dir, _ := tree(t)
		var err error
		got := withQuiet(t, func() {
			err = runFormatMode(dir, "", 2, " ", updater.ModeCheck)
		})
		if !errors.Is(err, updater.ErrNeedsFormatting) {
			t.Fatalf("expected ErrNeedsFormatting, got %v", err)
		}
		if !strings.Contains(got, "bad.conf") {
			t.Errorf("--quiet swallowed the file list, leaving only an exit code:\n%q", got)
		}
		// The narration is still gone: the list is all there is.
		if strings.Contains(got, "Specify the") || strings.Contains(got, "Successed") {
			t.Errorf("narration leaked alongside the list:\n%s", got)
		}
	})

	t.Run("--diff still emits the patch", func(t *testing.T) {
		dir, _ := tree(t)
		got := withQuiet(t, func() {
			_ = runFormatMode(dir, "", 2, " ", updater.ModeDiff)
		})
		for _, want := range []string{"--- ", "+++ ", "@@", "+  listen 80;"} {
			if !strings.Contains(got, want) {
				t.Errorf("--quiet swallowed part of the patch, missing %q:\n%s", want, got)
			}
		}
	})

	t.Run("write mode is still silent", func(t *testing.T) {
		dir, target := tree(t)
		got := withQuiet(t, func() {
			if err := runFormatMode(dir, "", 2, " ", updater.ModeWrite); err != nil {
				t.Errorf("runFormatMode: %v", err)
			}
		})
		if got != "" {
			t.Errorf("--quiet leaked narration in write mode:\n%s", got)
		}
		// And the work still happened, so this is not passing by doing nothing.
		b, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if !strings.Contains(string(b), "  listen 80;") {
			t.Errorf("--quiet skipped the formatting: %q", b)
		}
	})
}

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

// captureBoth runs fn with both standard streams redirected and returns what
// each received. --check and --diff are only useful if the two can be told
// apart, so a test for them has to look at each stream separately.
func captureBoth(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	read := func(saved **os.File) (*os.File, func() string) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("pipe: %v", err)
		}
		orig := *saved
		*saved = w
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
		return orig, func() string {
			*saved = orig
			_ = w.Close()
			return <-done
		}
	}

	_, finishOut := read(&os.Stdout)
	_, finishErr := read(&os.Stderr)

	// updater.Out is bound to os.Stdout at init and does not follow a later
	// reassignment, so redirecting os.Stdout alone lets every notice written
	// through Out escape to the real terminal. ModeWrite's notices go through
	// it, which is exactly what the write-mode assertion below is checking.
	savedOut := updater.Out
	updater.Out = os.Stdout

	fn()

	updater.Out = savedOut
	return finishOut(), finishErr()
}

// scanNoticeTree builds a tree whose scan produces a skip notice: a symbolic
// link, which is reported and stepped over rather than followed.
func scanNoticeTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.conf"), "server {\nlisten 80;\n}\n")
	if err := os.Symlink("a.conf", filepath.Join(dir, "link.conf")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return dir
}

// TestCheckModeKeepsScanNoticesOffStdout is the whole point of --check: its
// stdout is a list of paths for something else to consume. A skip notice
// printed there reads as one more path, so `--check | xargs` would go looking
// for a file called "Skipping".
//
// The notices are not dropped, only moved — a file the scan stepped over is
// precisely what someone running --check in CI needs to hear about.
func TestCheckModeKeepsScanNoticesOffStdout(t *testing.T) {
	dir := scanNoticeTree(t)

	var err error
	stdout, stderr := captureBoth(t, func() {
		err = updater.UpdateConfInDirMode(dir, dir, 2, " ", updater.ModeCheck, formatter.Formatter)
	})

	if !errors.Is(err, updater.ErrNeedsFormatting) {
		t.Fatalf("expected ErrNeedsFormatting, got %v", err)
	}
	if strings.Contains(stdout, "Skipping") {
		t.Errorf("a skip notice reached the file list on stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "a.conf") {
		t.Errorf("the unformatted file was not listed:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Skipping link.conf: symbolic link") {
		t.Errorf("the skip notice was lost instead of moved to stderr:\n%s", stderr)
	}
}

// TestDiffModeKeepsScanNoticesOffStdout is the same guarantee for the patch:
// a stray line in a unified diff makes it unapplyable.
func TestDiffModeKeepsScanNoticesOffStdout(t *testing.T) {
	dir := scanNoticeTree(t)

	stdout, stderr := captureBoth(t, func() {
		_ = updater.UpdateConfInDirMode(dir, dir, 2, " ", updater.ModeDiff, formatter.Formatter)
	})

	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		switch {
		case line == "",
			strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "),
			strings.HasPrefix(line, "@@"),
			strings.HasPrefix(line, " "), strings.HasPrefix(line, "-"), strings.HasPrefix(line, "+"):
		default:
			t.Errorf("stdout carries a line that is not part of a unified diff: %q\nfull output:\n%s", line, stdout)
		}
	}
	if !strings.Contains(stderr, "symbolic link") {
		t.Errorf("the skip notice was lost instead of moved to stderr:\n%s", stderr)
	}
}

// TestWriteModeKeepsScanNoticesOnStdout guards the other direction. The
// default mode's output is for a person reading along, and the notice has
// always been part of it; moving it would be a silent behaviour change for
// every existing user.
func TestWriteModeKeepsScanNoticesOnStdout(t *testing.T) {
	dir := scanNoticeTree(t)

	stdout, _ := captureBoth(t, func() {
		if err := updater.UpdateConfInDir(dir, dir, 2, " ", formatter.Formatter); err != nil {
			t.Fatalf("UpdateConfInDir: %v", err)
		}
	})

	if !strings.Contains(stdout, "Skipping link.conf: symbolic link") {
		t.Errorf("the skip notice disappeared from the default mode's output:\n%s", stdout)
	}
}

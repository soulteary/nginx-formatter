package updater_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

// TestStaleTempFileDoesNotBlockFormatting covers the fixed temporary name that
// used to be a permanent trap: a run killed between creating the file and
// renaming it left the name taken, and every later run then failed at O_EXCL
// with EEXIST until someone found and deleted the hidden file by hand.
func TestStaleTempFileDoesNotBlockFormatting(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "x.conf"), "a {\nb;\n}\n")
	// Exactly what a killed run would leave behind under a fixed name.
	mustWrite(t, filepath.Join(dir, ".x.conf.tmp"), "")

	if err := updater.UpdateConfInDir(dir, dir, 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("a stale temporary file blocked formatting: %v", err)
	}
	if got := mustRead(t, filepath.Join(dir, "x.conf")); !strings.Contains(got, "  b;") {
		t.Errorf("file was not formatted: %q", got)
	}
}

// TestUpdateConfFileKeepsDanglingSymlink covers an --output symlink whose
// target does not exist yet. filepath.EvalSymlinks fails on a dangling link,
// and treating that failure as "not a symlink" replaced the link with a
// regular file instead of creating the target it names.
func TestUpdateConfFileKeepsDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.conf")
	link := filepath.Join(dir, "out.conf")
	target := filepath.Join(dir, "target.conf")
	mustWrite(t, src, "a {\nb;\n}\n")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := updater.UpdateConfFile(src, link, 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("UpdateConfFile: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the dangling symlink was replaced with a regular file")
	}
	if got := mustRead(t, target); !strings.Contains(got, "  b;") {
		t.Errorf("the link target was not created with the formatted output: %q", got)
	}
}

// TestUpdateConfInDirLeavesNoTempFiles checks the directory writer cleans up
// after itself on both the success and the failure path.
func TestUpdateConfInDirLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "good.conf"), "a {\nb;\n}\n")
	mustWrite(t, filepath.Join(dir, "bad.conf"), "broken {{{\n")

	_ = updater.UpdateConfInDir(dir, dir, 2, " ", formatter.Formatter)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			t.Errorf("temporary file left behind: %s", e.Name())
		}
	}
}

// TestLongFilenameStillFormats covers the regression the atomic write
// introduced: the temporary name is ". + base + .tmp- + randomness", which is
// 16-22 bytes longer than the target, so a legal but long .conf name blew past
// NAME_MAX and the file could not be formatted at all.
func TestLongFilenameStillFormats(t *testing.T) {
	for _, n := range []int{200, 233, 245, 249} {
		name := strings.Repeat("a", n) + ".conf"
		t.Run(name[:12]+"...", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, name)
			mustWrite(t, path, "a {\nb;\n}\n")

			if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
				t.Fatalf("single-file mode: %v", err)
			}
			if got := mustRead(t, path); !strings.Contains(got, "  b;") {
				t.Errorf("single-file mode did not format: %q", got)
			}

			mustWrite(t, path, "a {\nb;\n}\n")
			if err := updater.UpdateConfInDir(dir, dir, 2, " ", formatter.Formatter); err != nil {
				t.Fatalf("directory mode: %v", err)
			}
			if got := mustRead(t, path); !strings.Contains(got, "  b;") {
				t.Errorf("directory mode did not format: %q", got)
			}
		})
	}
}

// TestOutputDirInsideInputIsNotReIngested covers the fractal case: an output
// directory nested in the input tree was walked as input on the next run, so
// each run nested one level deeper (out/, out/out/, ...).
func TestOutputDirInsideInputIsNotReIngested(t *testing.T) {
	src := t.TempDir()
	out := filepath.Join(src, "out")
	mustWrite(t, filepath.Join(src, "x.conf"), "a {\nb;\n}\n")

	for i := range 3 {
		if err := updater.UpdateConfInDir(src, out, 2, " ", formatter.Formatter); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	if _, err := os.Stat(filepath.Join(out, "out")); err == nil {
		t.Error("the output directory re-ingested its own output")
	}
	if got := mustRead(t, filepath.Join(out, "x.conf")); !strings.Contains(got, "  b;") {
		t.Errorf("output was not written: %q", got)
	}
}

// TestUnchangedFileIsNotRewritten keeps the formatter from bumping mtime for
// nothing, which wakes inotify watchers, config reloaders and make.
func TestUnchangedFileIsNotRewritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.conf")
	mustWrite(t, path, "a {\nb;\n}\n")

	if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	first, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// A second pass has nothing to change.
	if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("second pass: %v", err)
	}
	second, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !first.ModTime().Equal(second.ModTime()) {
		t.Errorf("mtime moved on an unchanged file: %v -> %v", first.ModTime(), second.ModTime())
	}
}

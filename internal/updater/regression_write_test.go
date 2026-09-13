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

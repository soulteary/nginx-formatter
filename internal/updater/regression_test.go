package updater_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

// TestScanFilesSkipsSymlinks covers the standard Debian/Ubuntu layout, where
// sites-enabled/x.conf links to sites-available/x.conf. os.Root refuses to
// traverse an absolute link, and that error used to abort the entire walk, so
// a stock /etc/nginx formatted zero files.
func TestScanFilesSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "sites-available"))
	mustMkdir(t, filepath.Join(dir, "sites-enabled"))
	mustMkdir(t, filepath.Join(dir, "conf.d"))

	target := filepath.Join(dir, "sites-available", "site.conf")
	mustWrite(t, target, "server {\nlisten 80;\n}\n")
	mustWrite(t, filepath.Join(dir, "conf.d", "other.conf"), "server {\nlisten 8080;\n}\n")

	// An absolute link, exactly as the nginx package and certbot create them.
	if err := os.Symlink(target, filepath.Join(dir, "sites-enabled", "site.conf")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	files, err := updater.ScanFiles(dir)
	if err != nil {
		t.Fatalf("ScanFiles aborted on a symlink: %v", err)
	}

	got := map[string]bool{}
	for _, f := range files {
		got[filepath.ToSlash(f)] = true
	}
	for _, want := range []string{"sites-available/site.conf", "conf.d/other.conf"} {
		if !got[want] {
			t.Errorf("expected %s to be scanned, got %v", want, files)
		}
	}
	if got["sites-enabled/site.conf"] {
		t.Error("the symlink should be skipped, not formatted a second time")
	}
}

// TestUpdateConfInDirContinuesPastBadFile checks that one unparseable config
// no longer leaves the rest of the tree unprocessed.
func TestUpdateConfInDirContinuesPastBadFile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "1.conf"), "a {\nb;\n}\n")
	mustWrite(t, filepath.Join(dir, "2.conf"), "broken {{{\n")
	mustWrite(t, filepath.Join(dir, "3.conf"), "c {\nd;\n}\n")

	err := updater.UpdateConfInDir(dir, dir, 2, " ", formatter.Formatter)
	if err == nil {
		t.Fatal("expected a non-nil error reporting the failed file")
	}
	if !strings.Contains(err.Error(), "2.conf") {
		t.Errorf("error should name the offending file, got: %v", err)
	}

	// The good files either side of the bad one must still be formatted.
	for _, name := range []string{"1.conf", "3.conf"} {
		got := mustRead(t, filepath.Join(dir, name))
		if !strings.Contains(got, "  ") {
			t.Errorf("%s was not formatted: %q", name, got)
		}
	}
	// The bad file must be left exactly as it was.
	if got := mustRead(t, filepath.Join(dir, "2.conf")); got != "broken {{{\n" {
		t.Errorf("the unparseable file was modified: %q", got)
	}
}

// TestUpdateConfFilePreservesMode guards against the formatter forcing 0600 on
// a config that nginx's workers run as a different user need to read.
func TestUpdateConfFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	mustWrite(t, path, "a {\nb;\n}\n")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("UpdateConfFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0644 {
		t.Errorf("mode changed from 0644 to %o", got)
	}
}

// TestUpdateConfFileKeepsSymlink checks that formatting a symlinked config
// (the sites-enabled pattern, pointed at directly) writes through the link
// instead of replacing it with a regular file.
func TestUpdateConfFileKeepsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.conf")
	link := filepath.Join(dir, "link.conf")
	mustWrite(t, target, "a {\nb;\n}\n")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := updater.UpdateConfFile(link, "", 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("UpdateConfFile: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced with a regular file")
	}
	if got := mustRead(t, target); !strings.Contains(got, "  b;") {
		t.Errorf("the link target was not formatted: %q", got)
	}
}

// TestUpdateConfFileLeavesNoTempFiles checks the atomic write cleans up after
// itself, including on the failure path.
func TestUpdateConfFileLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.conf")
	bad := filepath.Join(dir, "bad.conf")
	mustWrite(t, good, "a {\nb;\n}\n")
	mustWrite(t, bad, "broken {{{\n")

	_ = updater.UpdateConfFile(good, "", 2, " ", formatter.Formatter)
	_ = updater.UpdateConfFile(bad, "", 2, " ", formatter.Formatter)

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

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0750); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

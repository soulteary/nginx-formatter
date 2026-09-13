//go:build unix

package updater_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

// ownerIDs reports the uid and gid of path.
func ownerIDs(t *testing.T, path string) (uid, gid int) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("ownership is not exposed on this platform")
	}
	return int(st.Uid), int(st.Gid)
}

// otherGID is a group the calling process does not run as, used to prove the
// value is carried over rather than merely happening to match.
const otherGID = 1

// TestUpdateConfFilePreservesOwnership guards metadata a rename would
// otherwise drop. A rename installs a NEW inode owned by the calling process,
// so a root:nginx config formatted by root came back root:root and the nginx
// workers could no longer read it.
func TestUpdateConfFilePreservesOwnership(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("needs root to chown a file to another account")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	mustWrite(t, path, "a {\nb;\n}\n")
	if err := os.Chown(path, os.Geteuid(), otherGID); err != nil {
		t.Skipf("chown unavailable: %v", err)
	}

	if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("UpdateConfFile: %v", err)
	}

	if _, gid := ownerIDs(t, path); gid != otherGID {
		t.Errorf("group changed: want %d, got %d", otherGID, gid)
	}
}

// TestUpdateConfInDirPreservesOwnership is the same guarantee for the
// directory walker, which writes through the os.Root-scoped helper.
func TestUpdateConfInDirPreservesOwnership(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("needs root to chown a file to another account")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "site.conf")
	mustWrite(t, path, "a {\nb;\n}\n")
	if err := os.Chown(path, os.Geteuid(), otherGID); err != nil {
		t.Skipf("chown unavailable: %v", err)
	}

	if err := updater.UpdateConfInDir(dir, dir, 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("UpdateConfInDir: %v", err)
	}

	if _, gid := ownerIDs(t, path); gid != otherGID {
		t.Errorf("group changed: want %d, got %d", otherGID, gid)
	}
	if got := mustRead(t, path); !strings.Contains(got, "  b;") {
		t.Errorf("file was not formatted: %q", got)
	}
}

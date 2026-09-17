//go:build unix

package updater

import (
	"os"
	"syscall"
)

// fileOwnership is the uid/gid pair of an existing file.
type fileOwnership struct {
	uid int
	gid int
}

// ownerOf reports the owner of info, or nil when the platform does not expose
// it through os.FileInfo.
func ownerOf(info os.FileInfo) *fileOwnership {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	return &fileOwnership{uid: int(st.Uid), gid: int(st.Gid)}
}

//go:build !unix

package updater

import "os"

// fileOwnership is the uid/gid pair of an existing file.
type fileOwnership struct {
	uid int
	gid int
}

// ownerOf reports the owner of info. Platforms without Unix ownership
// semantics report nothing, and the caller then skips the chown step.
func ownerOf(os.FileInfo) *fileOwnership { return nil }

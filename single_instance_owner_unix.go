//go:build unix

package mygo

import (
	"os"
	"syscall"
)

func ownedBy(fi os.FileInfo, err error) bool {
	if err != nil {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}

// socketOwnedByMe reports whether path is a socket of this user: the
// default socket directory may be one another user can write to.
func socketOwnedByMe(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSocket != 0 && ownedBy(fi, nil)
}

// fileOwnedByMe reports whether path exists and is this user's.
func fileOwnedByMe(path string) bool { return ownedBy(os.Lstat(path)) }

// ownedByMe reports whether the open file f is this user's.
func ownedByMe(f *os.File) bool { return ownedBy(f.Stat()) }

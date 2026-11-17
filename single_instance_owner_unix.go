//go:build unix

package mygo

import (
	"os"
	"syscall"
)

// socketOwnedByMe reports whether path is a socket of this user: the
// default socket directory may be one another user can write to.
func socketOwnedByMe(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil || fi.Mode()&os.ModeSocket == 0 {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}

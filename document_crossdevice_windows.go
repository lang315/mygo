//go:build windows

package mygo

import (
	"errors"
	"syscall"
)

// crossDevice reports whether a rename failed because it crossed drives:
// ERROR_NOT_SAME_DEVICE, which Windows returns instead of EXDEV (Go
// defines that one there too, and tests return it).
func crossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.Errno(17))
}

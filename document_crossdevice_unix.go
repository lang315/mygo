//go:build unix

package mygo

import (
	"errors"
	"syscall"
)

// crossDevice reports whether a rename failed because it crossed volumes.
func crossDevice(err error) bool { return errors.Is(err, syscall.EXDEV) }

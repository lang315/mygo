//go:build windows && !arm64

package wincb

import "syscall"

// New returns the address native code calls fn at, as syscall.NewCallback.
func New(fn any) uintptr { return syscall.NewCallback(fn) }

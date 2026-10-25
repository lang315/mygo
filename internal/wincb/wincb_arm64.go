//go:build windows && arm64

package wincb

import "syscall"

// New returns the address native code calls fn at, as syscall.NewCallback.
// Callbacks are never freed, like syscall.NewCallback's.
func New(fn any) uintptr { return syscall.NewCallback(fn) }

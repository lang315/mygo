//go:build windows && (amd64 || arm64)

package vt

import "github.com/egoist/mygo/internal/wincb"

// newCallback makes the address libghostty-vt calls fn at. purego's would be
// syscall.NewCallback's, whose entry on Windows ARM64 changes the registers
// F8-F15 that C code expects kept: wincb.New's keeps them.
func newCallback(fn any) uintptr { return wincb.New(fn) }

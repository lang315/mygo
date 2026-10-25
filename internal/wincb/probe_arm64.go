//go:build windows && arm64

package wincb

import "syscall"

// ProbeFP calls the C function at entry as native code does, with F8-F15
// set to known values, and returns a bit for each register it changed: 0 for
// an entry that keeps them. For the tests of packages that make callbacks.
func ProbeFP(entry uintptr) uintptr {
	r, _, _ := syscall.SyscallN(fpProbeAddr(), entry)
	return r
}

// ClobberFP sets F8-F15, as the Go code of a callback may anywhere. For the
// tests of packages that make callbacks.
func ClobberFP() { clobberFP() }

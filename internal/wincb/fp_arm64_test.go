//go:build windows && arm64

package wincb

import (
	"syscall"
	"testing"
)

// Native code keeps doubles in F8-F15 across a call, which the callee must
// leave as it found them. The test calls a callback the way native code does
// and sees whether the registers are as they were.
func TestCallbackKeepsTheFloatingPointRegisters(t *testing.T) {
	cb := New(func() uintptr {
		clobberFP() // Go code may change them anywhere
		return 0
	})
	mask, _, _ := syscall.SyscallN(fpProbeAddr(), cb)
	if mask != 0 {
		t.Fatalf("the callback changed F8-F15, bits %#b", mask)
	}
}

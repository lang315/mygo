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

// The arguments on the stack and the result go through the entry: twelve
// arguments, four of them on the stack; the result is in R0 (callbacks of
// MyGo return an HRESULT, an LRESULT or a BOOL, never a double).
func TestCallbackPassesStackArgumentsAndResult(t *testing.T) {
	cb := New(func(a, b, c, d, e, f, g, h, i, j, k, l uintptr) uintptr {
		return a + 2*b + 3*c + 4*d + 5*e + 6*f + 7*g + 8*h + 9*i + 10*j + 11*k + 12*l
	})
	got, _, _ := syscall.SyscallN(cb, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	if want := uintptr(650); got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

// Each entry calls its own callback.
func TestEachEntryCallsItsOwnCallback(t *testing.T) {
	var entries []uintptr
	for i := range 40 {
		entries = append(entries, New(func() uintptr { return uintptr(1000 + i) }))
	}
	for i, e := range entries {
		if got, _, _ := syscall.SyscallN(e); got != uintptr(1000+i) {
			t.Fatalf("entry %d called the callback %d", i, got-1000)
		}
	}
}

func TestOneFunctionGetsOneEntry(t *testing.T) {
	f := func() uintptr { return 7 }
	if a, b := New(f), New(f); a != b {
		t.Fatalf("two entries %#x and %#x for one function", a, b)
	}
}

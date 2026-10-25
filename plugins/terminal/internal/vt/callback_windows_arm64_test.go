package vt

import (
	"testing"

	"github.com/egoist/mygo/internal/wincb"
)

// libghostty-vt, in C, may keep a double in F8-F15 across a call to one of
// our callbacks, as UI Automation's core does: a callback must leave them as
// it found them (see internal/wincb).
func TestCallbacksKeepTheFloatingPointRegisters(t *testing.T) {
	cb := newCallback(func(_, _, _ uintptr) uintptr {
		wincb.ClobberFP() // Go code may change them anywhere
		return 0
	})
	if changed := wincb.ProbeFP(cb); changed != 0 {
		t.Fatalf("a callback of the terminal changed F8-F15, bits %#b", changed)
	}
}

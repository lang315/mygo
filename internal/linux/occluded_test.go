//go:build linux && (amd64 || arm64)

package linux

import "testing"

func TestWindowOut(t *testing.T) {
	for _, c := range []struct {
		state    uint32
		obscured bool
		out      bool
	}{
		{0, false, false},
		{stateFocused | stateMaximized, false, false},
		{stateIconified, false, true},
		{stateWithdrawn, false, true},
		{stateIconified | stateFocused, false, true},
		{0, true, true},
	} {
		if got := windowOut(c.state, c.obscured); got != c.out {
			t.Errorf("windowOut(%#x, %v) = %v, want %v", c.state, c.obscured, got, c.out)
		}
	}
}

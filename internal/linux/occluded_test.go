//go:build linux && (amd64 || arm64)

package linux

import "testing"

func TestWindowOut(t *testing.T) {
	for _, c := range []struct {
		state uint32
		out   bool
	}{
		{0, false},
		{stateFocused | stateMaximized, false},
		{stateIconified, true},
		{stateWithdrawn, true},
		{stateIconified | stateFocused, true},
	} {
		if got := windowOut(c.state); got != c.out {
			t.Errorf("windowOut(%#x) = %v, want %v", c.state, got, c.out)
		}
	}
}

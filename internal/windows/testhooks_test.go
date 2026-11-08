//go:build windows && (amd64 || arm64)

package windows

import (
	"strings"
	"testing"
)

// The diagnosis of a click that did not come back runs only then, in CI,
// so its text is checked here.
func TestDiagnoseClick(t *testing.T) {
	got := diagnoseClick(point{X: 30, Y: 260}, true, 2)
	for _, want := range []string{
		"the button went down 2 times in the loop",
		"the cursor was set to (30, 260): true, and is at (",
		"the input desktop ",
		"this thread's is ",
		"the thread in front is ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q lacks %q", got, want)
		}
	}
}

// A click is sent again only when another app's window took it: not while a
// window of the app is under the point, where a slow app would take two.
func TestClickWentElsewhere(t *testing.T) {
	for _, c := range []struct {
		downs int
		own   bool
		want  bool
	}{
		{0, false, true},  // the runner's terminal over the point, no button reached the app
		{0, true, false},  // the app's window there: its answer is slow
		{1, false, false}, // the button reached the app, a window rose after it
		{1, true, false},
	} {
		if got := clickWentElsewhere(c.downs, c.own); got != c.want {
			t.Errorf("clickWentElsewhere(%d, %v) = %v, want %v", c.downs, c.own, got, c.want)
		}
	}
}

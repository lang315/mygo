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

package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A long Label in a group box of a dialog of fixed width wraps at the
// group's width: its lines stay inside the dialog, none clipped.
func TestLabelWrapsInAFixedWidthDialog(t *testing.T) {
	long := "The quick brown fox jumps over the lazy dog while the careful reader checks that every word of this sentence stays in view."
	for name, shape := range map[string]func(Control) Control{
		"label in the group": func(l Control) Control { return VBox(0).Add(GroupBox("g").Add(l)) },
		"label in a box":     func(l Control) Control { return VBox(0).Add(GroupBox("g").Add(VBox(0).Add(l))) },
	} {
		t.Run(name, func(t *testing.T) {
			bounds := func(s string) (r ui.Rect) {
				t.Helper()
				dt := TestDialog(t, NewDialog(nil, "t", "", 500, 0, shape(Label(s))))
				r, ok := dt.Tester.Tester.Find(s)
				if !ok {
					t.Fatalf("no %q in %q", s, dt.Texts())
				}
				return r
			}
			r, one := bounds(long), bounds("x")
			t.Logf("long %+v, one line %+v", r, one)
			if r.X < 0 || r.X+r.W > 500 {
				t.Fatalf("label at %v overflows the dialog's 500 DIPs", r)
			}
			if r.H < 2*one.H {
				t.Fatalf("label %v is one line (%v): it does not wrap", r, one)
			}
		})
	}
}

package widget

import "testing"

// A long Label in a group box of a dialog of fixed width wraps at the
// group's width: its lines stay inside the dialog, none clipped.
func TestLabelWrapsInAFixedWidthDialog(t *testing.T) {
	long := "The quick brown fox jumps over the lazy dog while the careful reader checks that every word of this sentence stays in view."
	d := NewDialog(nil, "t", "", 500, 0, VBox(0).Add(GroupBox("g").Add(VBox(0).Add(Label(long), Label("x")))))
	dt := TestDialog(t, d)
	r, ok := dt.Tester.Tester.Find(long)
	if !ok {
		t.Fatalf("no label in %q", dt.Texts())
	}
	one, ok := dt.Tester.Tester.Find("x")
	if !ok {
		t.Fatalf("no x in %q", dt.Texts())
	}
	t.Logf("long %+v, one line %+v", r, one)
	if r.X < 0 || r.X+r.W > 500 {
		t.Fatalf("label at %v overflows the dialog's 500 DIPs", r)
	}
	if r.H < 2*one.H {
		t.Fatalf("label %v is one line (%v): it does not wrap", r, one)
	}
}

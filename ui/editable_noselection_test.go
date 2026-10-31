package ui

import "testing"

// EditableText in the rows of a List that chooses none (neither Selected nor
// Selection) builds and shows its text: no row is chosen, so none is edited
// (the list edits the row chosen).
func TestEditableTextInAListWithoutSelection(t *testing.T) {
	vals := []string{"one", "two", "three"}
	tt := NewTester(func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }}
		List(c, s, len(vals), func(i int) {
			Row(c).Padding(4).Children(func() { EditableText(c, &vals[i]) })
		}).Grow(1)
	}, 300, 200)
	tt.Frame()
	r, ok := tt.Find("two")
	if !ok {
		t.Fatalf("the rows do not show their text: %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if _, ok := tt.Find("two"); !ok {
		t.Errorf("the text is gone after a double click: %q", tt.Texts())
	}
}

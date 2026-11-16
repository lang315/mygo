package ui

import "testing"

// Rows that show the fingers on them (Element.Touches): the fingers are
// the engine's, which no state of the row versions, so a row that reads
// them is built in every frame, as one built without RowVersion is.
func TestRetainRowsShowingTheirTouches(t *testing.T) {
	p := newPair(t, 300, 400, hoverRowsScene(func(c *Context, i int) {
		r := Row(c)
		n := len(r.Touches())
		r.Children(func() { Textf(c, "row %d: %d fingers", i, n) })
	}))
	p.idle(3)
	p.both(func(tt *Tester) { tt.TouchDown(1, 100, 90) })
	p.frame("down")
	p.frame("down 2")
	p.both(func(tt *Tester) { tt.TouchDown(2, 100, 95) })
	p.frame("second")
	p.both(func(tt *Tester) { tt.TouchMove(1, 120, 92) })
	p.frame("move")
	p.both(func(tt *Tester) { tt.TouchUp(1, 120, 92); tt.TouchCancel(2) })
	p.frame("up")
	p.frame("up 2")
	p.frame("up 3")
}

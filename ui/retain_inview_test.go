package ui

import "testing"

// InViewMargin of a row's element is true before the element is in view; a
// kept row would hold the answer false until the element shows, so a row
// that asks with a margin is built.
func TestRetainRowsAskingInViewMarginAreBuilt(t *testing.T) {
	p := newPair(t, 300, 200, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			List(c, s, 3, func(i int) {
				Column(c).Children(func() {
					Box(c).Size(100, 400)
					near := Box(c).Size(100, 40)
					Textf(c, "row %d near %v", i, near.InViewMargin(150))
				})
			}).Grow(1)
		}
	})
	p.idle(3)
	p.both(func(tt *Tester) { tt.Scroll(50, 100, 0, 120) })
	p.frame("scrolled")
	p.frame("settled")
	if !p.built.HasText("row 0 near true") {
		t.Fatalf("the twin does not show the element near: %q", p.built.Texts())
	}
}

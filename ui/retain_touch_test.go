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

// LongPressed and PullToRefresh ask again in every build (fingersFrame): the
// commit of a row that did not ask forgets what they asked for. Their rows
// are built in every frame, so that a kept row does not lose its long press
// or its pull to refresh.
func TestRetainLongPressAndRefreshRowsAreBuilt(t *testing.T) {
	refresh := make([]RefreshState, 30)
	for _, tc := range []struct {
		name string
		row  func(c *Context, i int)
	}{
		{"LongPressed", func(c *Context, i int) { Box(c).Size(100, 20).LongPressed() }},
		{"PullToRefresh", func(c *Context, i int) { Scroll(c).Size(100, 20).PullToRefresh(&refresh[i]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 400, 300, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						List(c, s, 30, func(i int) {
							Row(c).Padding(4).Children(func() { tc.row(c, i) })
						}).Grow(1)
					})
				}
			})
			p.idle(4)
			built := p.kept.rt.rowsBuilt
			p.frame("idle")
			if p.kept.rt.rowsBuilt == built {
				t.Errorf("rows that ask %s were kept", tc.name)
			}
		})
	}
}

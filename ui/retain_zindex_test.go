package ui

import "testing"

// The order a row's children paint in is part of what a replay keeps: a
// child whose ZIndex changes paints over its sibling anew, as the twin that
// builds every row shows.
func TestRetainRowsFollowAChildsZIndex(t *testing.T) {
	front := 0
	p := newPair(t, 300, 200, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return uint64(front) }}
		return func(c *Context) {
			List(c, s, 4, func(i int) {
				Row(c).Height(50).Children(func() {
					Box(c).Size(80, 40).Background(RGB(255, 0, 0)).ZIndex(front)
					Box(c).Size(80, 40).Margin(0, 0, 0, -40).Background(RGB(0, 0, 255)).ZIndex(1 - front)
				})
			}).Grow(1)
		}
	})
	p.idle(3)
	front = 1
	p.both(func(tt *Tester) { tt.Frame() })
	p.frame("the z-indexes swapped")
	p.frame("settled")
}

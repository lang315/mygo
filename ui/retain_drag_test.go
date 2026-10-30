package ui

import "testing"

// A row highlights with DragHover while a value dragged from outside the list
// is over it. The row's callback never asks Hovered, so a hovered row was
// kept; the drag path (rt.drag) must keep it from being linked as well.
func TestRetainDragHoverRow(t *testing.T) {
	red := Hex("#ff0000")
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				Box(c).Height(40).Width(300).Drag("v")
				List(c, s, 60, func(i int) {
					r := Row(c)
					if _, ok := DragHover[string](r); ok {
						r.Background(red)
					}
					r.Children(func() { Textf(c, "row %d", i) })
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	p.both(func(tt *Tester) { tt.Press(100, 20) })
	p.frame("press")
	for _, y := range []float32{30, 60, 120, 160} {
		p.both(func(tt *Tester) { tt.Move(100, y) })
		p.frame("drag")
		p.frame("drag+1")
	}
	p.both(func(tt *Tester) { tt.Release(100, 160) })
	p.frame("drop")
	p.frame("drop+1")
}

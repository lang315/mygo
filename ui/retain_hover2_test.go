package ui

import "testing"

// hoverScene: a List with RowVersion and a Selected cursor (hoverRow style),
// whose row callback is fn.
func hoverRowsScene(fn func(c *Context, i int)) func() func(c *Context) {
	return func() func(c *Context) {
		sel := -1
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Selected: &sel}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, 60, func(i int) { fn(c, i) }).Grow(1)
			})
		}
	}
}

// A row that reads Hovered only while Cmd is held (a Cmd+hover link look).
// The record is made without Cmd: no read. Pointer rests on row 3, then Cmd.
func TestRetainModifierGatedHover(t *testing.T) {
	red := Hex("#ff0000")
	p := newPair(t, 300, 400, hoverRowsScene(func(c *Context, i int) {
		r := Row(c)
		if c.Modifiers()&Cmd != 0 && r.Hovered() {
			r.Background(red)
		}
		r.Children(func() { Textf(c, "row %d", i) })
	}))
	p.idle(3)
	p.both(func(tt *Tester) { tt.Move(100, 90) })
	p.frame("hover")
	p.frame("hover 2")
	p.both(func(tt *Tester) { tt.HoldModifiers(Cmd) })
	p.frame("cmd")
	p.frame("cmd 2")
}

// Library hover look (hoverRow) on a kept row: hover, wheel-scroll under a
// resting pointer, move, leave.
func TestRetainHoveredRowsUnderAWheelScroll(t *testing.T) {
	p := newPair(t, 300, 400, hoverRowsScene(func(c *Context, i int) {
		Row(c).Padding(4).Children(func() { Textf(c, "row %d", i) })
	}))
	p.idle(3)
	p.both(func(tt *Tester) { tt.Move(100, 90) })
	p.frame("hover")
	for k := range 6 {
		p.both(func(tt *Tester) { tt.Scroll(100, 90, 0, 13) })
		p.frame("scroll")
		p.frame("scroll+1")
		_ = k
	}
	p.both(func(tt *Tester) { tt.Move(100, 200) })
	p.frame("move")
	p.both(func(tt *Tester) { tt.Move(100, 900) })
	p.frame("leave")
	p.frame("leave 2")
}

// Rows that show their shortcut while Cmd is held (the use Context.Modifiers
// documents), with no hover at all.
func TestRetainRowsShowingModifiersWithNoHover(t *testing.T) {
	p := newPair(t, 300, 400, hoverRowsScene(func(c *Context, i int) {
		Row(c).Children(func() {
			Textf(c, "row %d", i)
			if c.Modifiers()&Cmd != 0 {
				Textf(c, "⌘%d", i)
			}
		})
	}))
	p.idle(3)
	p.both(func(tt *Tester) { tt.HoldModifiers(Cmd) })
	p.frame("cmd")
	p.frame("cmd 2")
}

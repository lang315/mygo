package ui

import (
	"fmt"
	"testing"
)

// A Table of rows with a RowVersion that scrolls, by whole and fractional
// pixels, steps that are not exact in float32, pages and back, at scales 1 and
// 2: a kept Tester and one that builds every row draw alike after every step
// (P4a: a row that moved keeps its layout; P4b adds the assertion that it
// replays its paint shifted, `rowsShiftReplayed`).
func TestRetainMoveScenarios(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		t.Run(fmt.Sprintf("scale %v", scale), func(t *testing.T) {
			type app struct {
				rows []string
				vers []uint64
				list ListState
			}
			p := newPair(t, 400, 600, func() func(c *Context) {
				a := &app{rows: make([]string, 3000), vers: make([]uint64, 3000)}
				for i := range a.rows {
					a.rows[i] = fmt.Sprintf("acc%06d", i)
				}
				cols := []TableColumn{{Title: "Name"}, {Title: "Status", Width: 80}}
				a.list.Key = func(i int) any { return i }
				a.list.RowVersion = func(i int) uint64 { return a.vers[i] }
				return func(c *Context) {
					Table(c, &a.list, cols, len(a.rows), func(r, col int) {
						if col == 0 {
							Text(c, a.rows[r])
						} else {
							Textf(c, "%d", r%7)
						}
					}).Grow(1)
				}
			})
			p.both(func(tt *Tester) { tt.SetScale(scale) })
			for range 10 {
				p.both(func(tt *Tester) { tt.Frame() })
			}
			p.compare("warm")
			p.idle(4)
			p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, 2000) })
			p.compare("scroll far")
			p.idle(4)
			step := func(what string, dy float32) {
				t.Helper()
				p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, dy) })
				p.compare(what)
			}
			for i := range 10 {
				step(fmt.Sprintf("scroll %d", i), float32(1-2*(i%2)))
			}
			if n := p.kept.rt.rowsShifted; n == 0 {
				t.Errorf("a scroll by a pixel shifted the layout of %d rows, want > 0", n)
			}
			for i, dy := range []float32{0.5, 0.5, -0.5, -0.5, 0.25, 0.25, -0.25, -0.25} {
				step(fmt.Sprintf("scroll half %d", i), dy)
			}
			for i, dy := range []float32{0.37, 1.13, -0.37, -1.13} {
				step(fmt.Sprintf("scroll odd %d", i), dy)
			}
			step("scroll page", 500)
			step("scroll page back", -500)
			p.idle(3)
			step("scroll again", 3)
			p.idle(3)
		})
	}
}

// movedThen idles until rows are held, scrolls the list by dy so that its
// rows move (P4a: the layout of a row that moved is kept and placed again),
// and wants rows shifted.
func movedThen(t *testing.T, k *keptList, dy float32, what string) *pair {
	t.Helper()
	p := newPair(t, 300, 400, k.mk)
	p.idle(4)
	// The first move of a row that was built lays it out again and saves its
	// relative boxes (a copy of a row holds absolute ones): move once to prime.
	p.scrolled(1, "first move")
	p.idle(2)
	p.scrolled(dy, what)
	if p.kept.rt.rowsShifted == 0 {
		t.Fatalf("%s: no row shifted", what)
	}
	return p
}

func TestRetainMoveScroll1px(t *testing.T)  { movedThen(t, newKeptList(500), 1, "1 px") }
func TestRetainMoveScrollHalf(t *testing.T) { movedThen(t, newKeptList(500), 0.5, "half a pixel") }
func TestRetainMoveScrollOdd(t *testing.T)  { movedThen(t, newKeptList(500), 1.13, "1.13 px") }
func TestRetainMoveScrollFive(t *testing.T) { movedThen(t, newKeptList(500), 5, "5 px") }

// Rows that moved and then see an input of the layout change: nothing is kept
// (settle refuses; the relative boxes are the ones of the last layout).
func TestRetainMoveThenWidthChange(t *testing.T) {
	k := newKeptList(500)
	p := movedThen(t, k, 1, "1 px")
	k.width = 250
	p.frame("narrower")
	if n := p.kept.rt.rowsShifted + p.kept.rt.rowsLaidKept; n != 0 {
		t.Errorf("narrower: %d rows kept their layout, want 0", n)
	}
	p.idle(2)
	k.width = 0
	p.frame("wider")
	p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, 1) })
	p.compare("scroll after wider")
	p.idle(2)
}

func TestRetainMoveThenTextSize(t *testing.T) {
	k := newKeptList(500)
	p := movedThen(t, k, 1, "1 px")
	k.size = 19
	p.frame("text size")
	if n := p.kept.rt.rowsShifted + p.kept.rt.rowsLaidKept; n != 0 {
		t.Errorf("text size: %d rows kept their layout, want 0", n)
	}
	p.idle(2)
}

func TestRetainMoveDirection(t *testing.T) {
	k := newKeptList(500)
	p := movedThen(t, k, 1, "1 px")
	k.rtl = true
	p.frame("right to left")
	p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, 1) })
	p.compare("scroll")
	p.idle(2)
}

func TestRetainMoveSkippedLayoutsThenScroll(t *testing.T) {
	k := newKeptList(500)
	p := newPair(t, 300, 400, k.mk)
	p.idle(8)
	for i := range 4 {
		p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, float32(1-2*(i%2))) })
		p.compare(fmt.Sprintf("scroll %d", i))
	}
	p.idle(2)
}

func TestRetainMoveHidden(t *testing.T) {
	k := newKeptList(500)
	p := movedThen(t, k, 1, "1 px")
	k.hidden = true
	p.frame("hidden")
	k.hidden = false
	p.frame("shown")
	p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, 1) })
	p.compare("scroll")
	p.idle(2)
}

// A row laid out again after being shifted saves its relative boxes again.
func TestRetainMoveFreshAfterZero(t *testing.T) {
	k := newKeptList(500)
	p := movedThen(t, k, 1, "1 px")
	for i, w := range []float32{250, 0, 270, 0} {
		k.width = w
		p.frame(fmt.Sprintf("width %v", w))
		p.idle(3)
		p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, float32(1+i)) })
		p.compare(fmt.Sprintf("scroll after width %v", w))
	}
	p.idle(2)
}

// Pinned header rows are built late while the rows around them are held and
// moved.
func TestRetainMoveRowsBuiltLate(t *testing.T) {
	p := newPair(t, 300, 300, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Header: func(i int) bool { return i%20 == 0 }}
		return func(c *Context) {
			List(c, s, 400, func(i int) {
				if i%20 == 0 {
					Textf(c, "section %d", i)
					return
				}
				Row(c).Children(func() { Textf(c, "row %d", i); Text(c, "·") })
			}).Grow(1)
		}
	})
	p.idle(4)
	for i := range 40 {
		p.both(func(tt *Tester) { tt.Scroll(150, 150, 0, float32(1+i%3)) })
		p.compare(fmt.Sprintf("scroll %d", i))
	}
	p.idle(2)
}

// A scroll container in a moved row keeps its offset, and an offset that
// changed while the row moved is applied.
func TestRetainMoveInnerScrollContainer(t *testing.T) {
	p := innerScrollPair(t, false)
	p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, 3) })
	p.compare("list scrolled")
	p.both(func(tt *Tester) {
		tt.Press(20, 20)
		tt.Scroll(100, 124, 30, 7) // the inner offset and the list change together
		tt.Release(20, 20)
		tt.Frame()
		tt.Frame()
	})
	p.compare("both changed")
	p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, 2) })
	p.compare("list scrolled again")
	p.idle(3)
}

// Reveal of an element of a row that moved in the frame before.
func TestRetainMoveRevealInAMovedRow(t *testing.T) {
	p := newPair(t, 300, 300, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Padding(40).Children(func() {
				Text(c, "above")
				ScrollHorizontal(c).Width(200).Height(250).Children(func() {
					List(c, s, 30, func(i int) {
						Row(c).Width(900).Children(func() { Textf(c, "near %d", i); Spacer(c); Textf(c, "mid %d", i); Spacer(c); Textf(c, "far %d", i) })
					}).Width(900).Height(250)
				})
			})
		}
	})
	p.idle(4)
	p.both(func(tt *Tester) { tt.Scroll(100, 100, 0, 3) })
	p.compare("moved")
	var find func(n *node) uint64
	find = func(n *node) uint64 {
		if n.kind == kindText && n.text == "mid 2" {
			return n.id
		}
		for c := n.first; c != nil; c = c.next {
			if id := find(c); id != 0 {
				return id
			}
		}
		return 0
	}
	id := find(p.kept.rt.c.root)
	if id == 0 {
		t.Fatal("no text mid 2")
	}
	p.both(func(tt *Tester) { tt.ScrollIntoView(id) })
	p.compare("reveal")
	p.idle(3)
}

// P4a depends on this: a row with a Transition is never held (the animation of a
// transition rewrites boxes after layout, which the relative boxes saved would
// not hold). ExitInARow stays beside it.
func TestRetainTransitionRowsAreNeverHeld(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			List(c, s, 30, func(i int) {
				Row(c).Transition(ElementTransition{Position: true}).Children(func() { Textf(c, "row %d", i) })
			}).Grow(1)
		}
	})
	p.idle(4)
	if n := p.kept.rt.rowsLaidKept; n != 0 {
		t.Errorf("%d rows with a Transition kept their layout", n)
	}
	p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, 1) })
	p.compare("scroll")
}

// A Form is volatile (keepNoLayout): nothing is held in it, shifted or not.
func TestRetainMoveBaselines(t *testing.T) {
	p := newPair(t, 320, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			Form(c, func() {
				Field(c, "Items", func() { auditRows(c, s, 40).Height(200) })
			}).Grow(1)
		}
	})
	p.idle(4)
	p.both(func(tt *Tester) { tt.Scroll(100, 100, 0, 3) })
	p.compare("scroll")
	if n := p.kept.rt.rowsShifted; n != 0 {
		t.Errorf("%d rows shifted in a Form", n)
	}
	p.idle(2)
}

// A row laid out by layoutList saves its relative boxes at the place hook: the
// first move after that layout shifts it, with no move to prime it.
func TestRetainMoveFirstMoveAfterLayout(t *testing.T) {
	k := newKeptList(500)
	p := newPair(t, 300, 400, k.mk)
	p.idle(4)
	k.width = 250
	p.frame("narrower")
	p.idle(3)
	p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, 1) })
	p.compare("first move")
	if p.kept.rt.rowsShifted == 0 {
		t.Fatal("the first move after a layout shifted no row")
	}
	p.idle(2)
}

// Rows shifted in consecutive frames: each frame's commit records the row at
// its new place for the next.
func TestRetainMoveConsecutiveMoves(t *testing.T) {
	k := newKeptList(500)
	p := newPair(t, 300, 400, k.mk)
	p.idle(4)
	for i := range 8 {
		p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, float32(1+i%2)) })
		p.compare(fmt.Sprintf("move %d", i))
		if i > 0 && p.kept.rt.rowsShifted == 0 {
			t.Fatalf("move %d shifted no row", i)
		}
	}
}

package ui

import (
	"fmt"
	"testing"
	"time"
)

// The audit of what walks into the rows of a list after the build: each test
// puts a List with a RowVersion in the element a walker belongs to, idles
// until rows are held, changes the thing, and compares with rows built.

// auditRows is a List of n rows of two texts and a RowVersion.
func auditRows(c *Context, s *ListState, n int) Element {
	return List(c, s, n, func(i int) { Row(c).Children(func() { Textf(c, "row %d", i); Text(c, "·") }) })
}

func auditState() *ListState {
	return &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
}

// auditPair idles a pair of the view until rows are held, and wants them.
func auditPair(t *testing.T, mk func() func(c *Context), w, h int) *pair {
	t.Helper()
	p := newPair(t, w, h, mk)
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	if p.kept.rt.rowsLaidKept == 0 {
		t.Fatalf("no row held")
	}
	return p
}

// dragdrop.go: a list with Reorder keeps no rows, and a drag from outside over
// the list leaves the rows as built.
func TestRetainLayoutReorderListHoldsNothing(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		s.Reorder = func(rows []int, to int) {}
		return func(c *Context) { auditRows(c, s, 30).Grow(1) }
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	if n := p.kept.rt.rowsLaidKept; n != 0 {
		t.Errorf("%d rows of a Reorder list kept their layout", n)
	}
	b, _ := p.kept.Find("row 3")
	p.both(func(tt *Tester) { drag(tt, b.X+2, b.Y+b.H/2, b.X+2, b.Y+4*b.H) })
	p.compare("dragged")
	p.idle(2)
}

func TestRetainLayoutDragOverTheList(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		var off Point
		return func(c *Context) {
			Column(c).Grow(1).Children(func() {
				Box(c).Width(40).Height(30).Background(RGB(200, 0, 0)).Draggable(&off, DraggableOptions{})
				auditRows(c, s, 30).Grow(1)
			})
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	p.both(func(tt *Tester) { tt.Press(20, 15) })
	for i, y := range []float32{40, 120, 200, 260, 90} {
		p.both(func(tt *Tester) { tt.Move(100, y) })
		p.compare(fmt.Sprintf("drag %d", i))
	}
	p.both(func(tt *Tester) { tt.Release(100, 90) })
	p.compare("dropped")
	p.idle(3)
}

// effect.go: a list inside an Effect.
func TestRetainLayoutListInAnEffect(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		fx := &layerSpy{ids: map[uint64]bool{}}
		return func(c *Context) {
			Column(c).Grow(1).Effect(fx).Children(func() { auditRows(c, s, 30).Grow(1) })
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	p.both(func(tt *Tester) { tt.Scroll(100, 100, 0, 30) })
	p.compare("scroll")
	p.idle(3)
}

// form.go: a list as the content of a Form field.
func TestRetainLayoutListInAForm(t *testing.T) {
	p := newPair(t, 320, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			Form(c, func() {
				Field(c, "Name", func() { Text(c, "x") })
				Field(c, "Items", func() { auditRows(c, s, 20).Height(200) })
			}).Grow(1)
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	p.both(func(tt *Tester) { tt.h.w = 260 })
	p.frame("narrower")
	p.idle(3)
}

// layout.go: a column that is as tall as its content up to a limit, around a
// list.
func TestRetainLayoutListInAFitContainer(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			Column(c).MaxHeight(250).Children(func() { auditRows(c, s, 4) })
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	p.both(func(tt *Tester) { tt.h.w = 220 })
	p.frame("narrower")
	p.idle(3)
}

// toolbar.go: a list as an item of a toolbar.
func TestRetainLayoutListInAToolbar(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			Column(c).Grow(1).Children(func() {
				Toolbar(c, func() { Text(c, "a"); auditRows(c, s, 3).Width(120) })
				Text(c, "below")
			})
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	p.both(func(tt *Tester) { tt.h.w = 150 })
	p.frame("narrower")
	p.idle(3)
}

// transition.go: a child of a row leaves with an Exit when the row's version
// changes; a sibling of the list leaves with an Exit.
func TestRetainLayoutExitInARow(t *testing.T) {
	var vers [30]uint64
	var show [30]bool
	for i := range show {
		show[i] = true
	}
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		s.RowVersion = func(i int) uint64 { return vers[i] }
		return func(c *Context) {
			List(c, s, 30, func(i int) {
				Row(c).Children(func() {
					Textf(c, "row %d", i)
					if show[i] {
						Text(c, "tag").Key("tag").Transition(ElementTransition{Duration: 100 * time.Millisecond, Exit: &Motion{Collapse: true, Opacity: 0}})
					}
				})
			}).Grow(1)
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	show[3], vers[3] = false, 1
	p.frame("tag removed")
	for i := range 4 {
		p.both(func(tt *Tester) { tt.Advance(40 * time.Millisecond) })
		p.compare(fmt.Sprintf("exit %d", i))
	}
	p.idle(3)
}

func TestRetainLayoutExitBesideTheList(t *testing.T) {
	var banner = true
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			Column(c).Grow(1).Children(func() {
				if banner {
					Text(c, "banner").Key("banner").Transition(ElementTransition{Duration: 100 * time.Millisecond, Exit: &Motion{Collapse: true}})
				}
				auditRows(c, s, 30).Grow(1)
			})
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	banner = false
	p.frame("banner removed")
	for i := range 4 {
		p.both(func(tt *Tester) { tt.Advance(40 * time.Millisecond) })
		p.compare(fmt.Sprintf("exit %d", i))
	}
	p.idle(3)
}

// list.go: PageDown and PageUp, which read the rows' places.
func TestRetainLayoutPageKeys(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		sel := 0
		s.Selected = &sel
		return func(c *Context) { auditRows(c, s, 200).Grow(1) }
	})
	p.both(func(tt *Tester) { tt.ClickAt(100, 20) })
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	for i, k := range []Key{KeyPageDown, KeyPageDown, KeyPageUp, KeyEnd, KeyHome} {
		p.both(func(tt *Tester) { tt.Key(0, k) })
		p.compare(fmt.Sprintf("key %d", i))
		p.idle(2)
	}
}

// textselection.go: selectable text in the rows, selected across held rows.
func TestRetainLayoutSelectableText(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			Column(c).Grow(1).Selectable().Children(func() {
				List(c, s, 30, func(i int) { Textf(c, "selectable row %d", i) }).Grow(1)
			})
		}
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	p.both(func(tt *Tester) { drag(tt, 20, 30, 150, 120) })
	p.compare("selected")
	p.idle(3)
}

// access.go: the access tree is the same.
func TestRetainLayoutAccessTree(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) { auditRows(c, s, 30).Grow(1) }
	})
	p.idle(4)
	t.Logf("rows keeping their layout when idle: %d", p.kept.rt.rowsLaidKept)
	a, b := fmt.Sprintf("%+v", p.kept.rt.accessTree()), fmt.Sprintf("%+v", p.built.rt.accessTree())
	if a != b {
		t.Errorf("the access trees differ:\nkept  %.300s\nbuilt %.300s", a, b)
	}
}

// pose.go: an ancestor with a pose is TestRetainCommitParentPose.

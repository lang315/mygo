package ui

import "testing"

// built returns how many rows the twin that keeps rows built and kept in the
// next frame, drawn in both twins.
func builtInNextFrame(p *pair, what string) (built, kept int) {
	b, k := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame(what)
	return p.kept.rt.rowsBuilt - b, p.kept.rt.rowsKept - k
}

// Rows that read the layout of their own elements are built while it settles
// and kept once it does: the answers they recorded are the ones a build would
// get.
func TestRetainRowsReadingTheirOwnLayoutSettleAndAreKept(t *testing.T) {
	for _, tc := range []struct {
		name string
		show func(c *Context, r Element)
	}{
		{"Bounds", func(c *Context, r Element) { Textf(c, "w=%v", r.Bounds().W) }},
		{"LayoutBounds", func(c *Context, r Element) { Textf(c, "h=%v", r.LayoutBounds().H) }},
		{"LocalToWindow", func(c *Context, r Element) { x, y := r.LocalToWindow(1, 2); Textf(c, "%v,%v", x, y) }},
		{"WindowToLocal", func(c *Context, r Element) { x, _, ok := r.WindowToLocal(50, 50); Textf(c, "%v %v", x, ok) }},
		{"InView", func(c *Context, r Element) { Textf(c, "%v", r.InView()) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						List(c, s, 30, func(i int) {
							r := Row(c).Padding(4)
							r.Children(func() { tc.show(c, r) })
						}).Grow(1)
					})
				}
			})
			p.idle(6)
			if b, k := builtInNextFrame(p, "settled"); b != 0 || k == 0 {
				t.Errorf("settled: %d rows built, %d kept; want none built", b, k)
			}
		})
	}
}

// The header's layout changes: rows that read the layout of an element outside
// them are built (the element may not be built in this pass, and its state
// outlives it), and show what a build shows.
func TestRetainRowsReadingAnotherElementsLayoutAreBuilt(t *testing.T) {
	h := float32(40)
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				head := Row(c).Key("head").Height(h).Children(func() { Textf(c, "header") })
				List(c, s, 30, func(i int) {
					Row(c).Padding(4).Children(func() { Textf(c, "row %d h=%v", i, head.Bounds().H) })
				}).Grow(1)
			})
		}
	})
	p.idle(6)
	if b, _ := builtInNextFrame(p, "settled"); b == 0 {
		t.Errorf("rows that read an element outside them were kept")
	}
	h = 60
	p.frame("taller")
	p.frame("taller 2")
}

// The element outside the row stops being built while its state lives on: with
// KeepAlive for good, without it until the frame prunes it. A row kept would show
// the height it had; a row built finds no element and shows what a fresh one does.
func TestRetainRowsReadingAnElementThatIsNoLongerBuilt(t *testing.T) {
	for _, keepAlive := range []bool{false, true} {
		name := "without KeepAlive"
		if keepAlive {
			name = "with KeepAlive"
		}
		t.Run(name, func(t *testing.T) {
			show := true
			p := newPair(t, 300, 400, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						var head Element
						// A slot of a fixed height: the list does not move as the header goes.
						Box(c).Height(40).Children(func() {
							if show {
								c.GlobalKey("head")
								head = Row(c).Height(40).Children(func() { Textf(c, "header") })
							} else if keepAlive {
								c.KeepAlive("head")
							}
						})
						List(c, s, 30, func(i int) {
							Row(c).Padding(4).Children(func() { Textf(c, "row %d h=%v", i, head.Bounds().H) })
						}).Grow(1)
					})
				}
			})
			p.idle(5)
			show = false
			for i := range 4 {
				p.frame("gone " + string(rune('0'+i)))
			}
			show = true
			p.frame("back")
			p.frame("back 2")
		})
	}
}

// The header is built after the list in one frame and before it in the next,
// with the same global key: the row reads the state the frame before left.
func TestRetainRowsReadingAnElementThatMovesAfterTheList(t *testing.T) {
	after := false
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				var head Element
				if !after {
					c.GlobalKey("head")
					head = Row(c).Height(40).Children(func() { Textf(c, "header") })
				}
				List(c, s, 30, func(i int) {
					Row(c).Padding(4).Children(func() { Textf(c, "row %d h=%v", i, head.Bounds().H) })
				}).Grow(1)
				if after {
					c.GlobalKey("head")
					Row(c).Height(40).Children(func() { Textf(c, "header") })
				}
			})
		}
	})
	p.idle(5)
	after = true
	p.frame("moved")
	p.frame("moved 2")
	p.idle(3)
}

// What a record says is compared with the states now: an element whose state is
// gone answers as one the frame before lacked, and a state taken by another
// element (the same id, other places) answers with its own.
func TestSameLayoutsComparesWithTheStatesNow(t *testing.T) {
	var id uint64
	p := newPair(t, 300, 400, func() func(c *Context) {
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				id = Row(c).Key("head").Height(40).Children(func() { Textf(c, "header") }).ID()
			})
		}
	})
	p.idle(4)
	rt := p.kept.rt
	read := []layoutRead{{id: id, kind: lkBounds, answer: layoutOf(rt, rt.states[id], lkBounds, 0, 0)}}
	if read[0].answer.d == 0 {
		t.Fatal("the header has no height: the test does not test")
	}
	if !rt.sameLayouts(read) {
		t.Errorf("the recorded answer is not the one now")
	}
	st := rt.states[id]
	delete(rt.states, id) // its state is gone
	if rt.sameLayouts(read) {
		t.Errorf("an element whose state is gone answered as before")
	}
	rt.states[id] = &state{} // the id taken by a state that was never laid out
	if rt.sameLayouts(read) {
		t.Errorf("a state of another element answered as before")
	}
	rt.states[id] = st
	if !rt.sameLayouts(read) {
		t.Errorf("the state is back and the answer is not")
	}
}

// layoutsOfItsOwn: a list whose rows hold n boxes and add up their widths, read
// as the layout of the frame before.
func layoutsOfItsOwn(n int) func() func(c *Context) {
	return func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, 30, func(i int) {
					Row(c).Padding(4).Children(func() {
						w := float32(0)
						for k := range n {
							w += Box(c).Key(k).Size(10, 10).Bounds().W
						}
						Textf(c, "row %d: %v", i, w)
					})
				}).Grow(1)
			})
		}
	}
}

// A row that asks the layout of more elements than the list records is built
// in every frame, as before.
func TestRetainRowsReadingTooManyLayoutsAreBuilt(t *testing.T) {
	p := newPair(t, 300, 400, layoutsOfItsOwn(maxLayoutReads+1))
	p.idle(6)
	if b, _ := builtInNextFrame(p, "settled"); b == 0 {
		t.Errorf("a row that asks %d layouts was kept; want built in every frame", maxLayoutReads+1)
	}
}

// A row that asks eight layouts is recorded and kept.
func TestRetainRowsReadingTheMostLayoutsAreKept(t *testing.T) {
	p := newPair(t, 300, 400, layoutsOfItsOwn(maxLayoutReads))
	p.idle(6)
	if b, k := builtInNextFrame(p, "settled"); b != 0 || k == 0 {
		t.Errorf("settled: %d rows built, %d kept; want none built", b, k)
	}
}

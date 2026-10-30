package ui

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
)

// rowsScene: a header above a List with RowVersion and a Selected cursor;
// head builds the header and returns what the rows see, row builds row i.
func rowsScene(head func(c *Context) any, row func(c *Context, i int, h any)) func() func(c *Context) {
	return func() func(c *Context) {
		sel := -1
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Selected: &sel}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				h := head(c)
				List(c, s, 30, func(i int) { row(c, i, h) }).Grow(1)
			})
		}
	}
}

// A row that asks Hovered, PointerPosition or Pressed of an element built
// outside it (the header, saved by the app) depends on that element's state,
// which the row's own states do not say: the row is built.
func TestRetainRowAsksOfAnElementOutsideIt(t *testing.T) {
	red := Hex("#ff0000")
	for _, tc := range []struct {
		name string
		ask  func(h Element) bool
		do   func(tt *Tester)
	}{
		{"Hovered", func(h Element) bool { return h.Hovered() }, func(tt *Tester) { tt.Move(100, 20) }},
		{"PointerPosition", func(h Element) bool { _, _, over := h.PointerPosition(); return over }, func(tt *Tester) { tt.Move(100, 20) }},
		{"Pressed", func(h Element) bool { return h.Pressed() }, func(tt *Tester) { tt.Move(100, 20); tt.Press(100, 20) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, rowsScene(func(c *Context) any {
				return Row(c).Height(40).Children(func() { Textf(c, "header") })
			}, func(c *Context, i int, h any) {
				r := Row(c)
				if tc.ask(h.(Element)) {
					r.Background(red)
				}
				r.Children(func() { Textf(c, "row %d", i) })
			}))
			p.idle(3)
			p.both(tc.do)
			p.frame("asked")
			p.frame("asked 2")
			p.both(func(tt *Tester) { tt.Release(100, 20); tt.Move(100, 390) })
			p.frame("back on the list")
			p.frame("back on the list 2")
		})
	}
}

// The parent asks Modifiers and the rows use the answer it passes down: the
// parent is no row, so the rows never asked, and RowVersion has to say it
// (the doc of Modifiers, like that of RowVersion). The rows are kept.
func TestRetainRowsUsingAValueTheParentReadIsKept(t *testing.T) {
	p := newPair(t, 300, 400, rowsScene(func(c *Context) any {
		Row(c).Height(40).Children(func() { Textf(c, "header") })
		return c.Modifiers()&Cmd != 0
	}, func(c *Context, i int, h any) {
		Row(c).Children(func() { Textf(c, "row %d", i) })
	}))
	p.idle(3)
	p.both(func(tt *Tester) { tt.HoldModifiers(Cmd) })
	p.frame("cmd")
	p.frame("cmd 2")
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept")
	}
}

// A random walk of the pointer, wheel, modifiers, presses and version bumps
// over rows that ask Hovered, PointerPosition, Modifiers, or Modifiers-gated
// Hovered, or nothing; the kept twin must match the one that builds every row.
func TestRetainRandomWalkOfReaders(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) { walkOfReaders(t, seed) })
	}
}

func walkOfReaders(t *testing.T, seed int64) {
	red, blue := Hex("#ff0000"), Hex("#0000ff")
	vers := make([]uint64, 60)
	p := newPair(t, 300, 400, func() func(c *Context) {
		sel := -1
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return vers[i] }, Selected: &sel}
		if seed%2 == 0 {
			s.Selected = nil
		}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				head := Row(c).Height(30).Children(func() { Textf(c, "head") })
				List(c, s, len(vers), func(i int) {
					r := Row(c).Padding(4)
					switch i % 5 {
					case 0:
						if r.Hovered() {
							r.Background(red)
						}
					case 1:
						if x, _, over := r.PointerPosition(); over && x > 150 {
							r.Background(blue)
						}
					case 2:
						if c.Modifiers()&Cmd != 0 {
							r.Background(red)
						}
					case 3:
						if c.Modifiers()&Alt != 0 && r.Hovered() {
							r.Background(blue)
						}
					case 4:
						if head.Hovered() || head.Pressed() {
							r.Background(blue) // asks of an element outside the row
						}
					}
					r.Children(func() {
						Textf(c, "row %d v%d", i, vers[i])
						if i%5 == 2 && c.Modifiers()&Shift != 0 {
							Textf(c, "shift")
						}
					})
				}).Grow(1)
			})
		}
	})
	rng := rand.New(rand.NewSource(seed))
	sys := newSystemTheme(uint64(seed))
	mods := []Modifiers{0, Cmd, Alt, Shift, Cmd | Alt}
	p.idle(2)
	for step := range 120 {
		x, y := float32(rng.Intn(320)-10), float32(rng.Intn(420)-10)
		var what string
		switch k := rng.Intn(10); {
		case k < 4:
			what = fmt.Sprintf("move %v,%v", x, y)
			p.both(func(tt *Tester) { tt.Move(x, y) })
		case k < 5:
			dy := float32(rng.Intn(80) - 40)
			what = fmt.Sprintf("scroll %v at %v,%v", dy, x, y)
			p.both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
		case k < 7:
			m := mods[rng.Intn(len(mods))]
			what = fmt.Sprintf("mods %v", m)
			p.both(func(tt *Tester) { tt.HoldModifiers(m) })
		case k < 8:
			i := rng.Intn(len(vers))
			vers[i]++
			what = fmt.Sprintf("bump %d", i)
		case k < 9:
			what = fmt.Sprintf("click %v,%v", x, y)
			p.both(func(tt *Tester) { tt.Press(x, y); tt.Release(x, y) })
		default:
			what = "blur/focus"
			p.both(func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
		}
		what += sys.step(p.both)
		p.frame(fmt.Sprintf("%d %s", step, what))
		p.frame(fmt.Sprintf("%d %s +1", step, what))
	}
	if p.kept.rt.rowsKept == 0 {
		t.Fatalf("no row kept: the walk tests nothing")
	}
}

// Rows that ask of the header (an element outside them) about its focus, its
// clicks and its drag: built, or the header would lose the flags asking sets
// (flagClickable, flagDraggable...) and the twin's hits would differ at once.
func TestRetainRowsAskingOfTheHeaderAnyState(t *testing.T) {
	for _, tc := range []struct {
		name string
		ask  func(h Element) bool
	}{
		{"Focused", func(h Element) bool { return h.Focused() }},
		{"FocusWithin", func(h Element) bool { return h.FocusWithin() }},
		{"Clicked", func(h Element) bool { return h.Clicked() }},
		{"Clicks", func(h Element) bool { return h.Clicks() > 0 }},
		{"DoubleClicked", func(h Element) bool { return h.DoubleClicked() }},
		{"RightClicked", func(h Element) bool { return h.RightClicked() }},
		{"Dragged", func(h Element) bool { _, _, ok := h.Dragged(); return ok }},
		{"DragVelocity", func(h Element) bool { vx, vy := h.DragVelocity(); return vx != 0 || vy != 0 }},
		{"DragEnded", func(h Element) bool { _, _, ok := h.DragEnded(); return ok }},
		{"Dragging", func(h Element) bool { return h.Dragging() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						h := Row(c).Height(40).Focusable().Drag("v").Children(func() { Textf(c, "header") })
						List(c, s, 60, func(i int) {
							r := Row(c)
							if tc.ask(h) {
								r.Background(Hex("#ff0000"))
							}
							r.Children(func() { Textf(c, "row %d", i) })
						}).Grow(1)
					})
				}
			})
			p.idle(3)
			p.both(func(tt *Tester) { tt.Press(100, 20) })
			p.frame("press")
			p.both(func(tt *Tester) { tt.Move(100, 120) })
			p.frame("drag")
			p.frame("drag 2")
			p.both(func(tt *Tester) { tt.Release(100, 120) })
			p.frame("release")
			p.both(func(tt *Tester) { tt.ClickAt(100, 20); tt.ClickAt(100, 20) })
			p.frame("clicks")
			p.both(func(tt *Tester) { tt.RightClickAt(100, 20) })
			p.frame("right click")
			p.frame("right click 2")
		})
	}
}

// Rows that ask only of themselves are kept even with the pointer on the
// header: asking of the header is the only thing that makes a row outside.
func TestRetainRowsAskingOnlyOfThemselvesAreKept(t *testing.T) {
	red := Hex("#ff0000")
	p := newPair(t, 300, 400, rowsScene(func(c *Context) any {
		return Row(c).Height(40).Children(func() { Textf(c, "header") })
	}, func(c *Context, i int, _ any) {
		r := Row(c).Padding(4)
		if r.Hovered() || r.Pressed() {
			r.Background(red)
		}
		r.Children(func() {
			x := Textf(c, "row %d", i)
			if _, _, over := x.PointerPosition(); over {
				Textf(c, "over")
			}
		})
	}))
	p.both(func(tt *Tester) { tt.Move(100, 20) })
	p.idle(3)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
		t.Errorf("idle: %d rows built, %d kept; want none built", b, k)
	}
}

// A List inside a row: an inner row that asks of an element of the outer row
// is outside it and built; one that asks of itself, or an outer row that asks of
// an inner element, leaves the inner rows kept.
func TestRetainNestedRowAsksOfTheOuterRow(t *testing.T) {
	for _, mode := range []string{"inner asks of itself", "inner asks of the outer row", "outer asks of an inner element"} {
		t.Run(mode, func(t *testing.T) {
			p := newPair(t, 300, 400, func() func(c *Context) {
				outer := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				inner := make([]*ListState, 20)
				for i := range inner {
					inner[i] = &ListState{Key: func(j int) any { return j }, RowVersion: func(int) uint64 { return 0 }}
				}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						List(c, outer, 20, func(i int) {
							Column(c).Children(func() {
								o := Textf(c, "outer %d", i)
								var last Element
								List(c, inner[i], 2, func(j int) {
									r := Row(c)
									last = r
									switch mode {
									case "inner asks of itself":
										if r.Hovered() {
											r.Background(Hex("#ff0000"))
										}
									case "inner asks of the outer row":
										if o.Hovered() {
											r.Background(Hex("#ff0000"))
										}
									}
									r.Children(func() { Textf(c, "inner %d.%d", i, j) })
								}).Height(50)
								if mode == "outer asks of an inner element" && last.Hovered() {
									o.Background(Hex("#0000ff"))
								}
							})
						}).Grow(1)
					})
				}
			})
			p.idle(3)
			built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
			p.frame("idle")
			b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept
			t.Logf("idle: %d rows built, %d kept", b, k)
			p.both(func(tt *Tester) { tt.Move(100, 30) })
			p.frame("move")
			p.frame("move 2")
			p.both(func(tt *Tester) { tt.Move(100, 70) })
			p.frame("move again")
			p.frame("move again 2")
			// A row holding a List is never kept: the rows kept are the inner ones, and
			// an inner row that asks of an element of the outer row is outside it.
			if inner := mode != "inner asks of the outer row"; inner && k == 0 || !inner && k != 0 {
				t.Errorf("idle: %d rows built, %d kept", b, k)
			}

		})
	}
}

// More ways to ask of the header: files dragged over it, a press outside it,
// its animations. Some set a flag on the header (flagDropTarget...) that a kept
// row would never set again; the pictures do not show an event of one frame
// (TestRetainRowsSeeTheEventsOfTheHeader counts them).
func TestRetainRowsAskingOfTheHeaderMore(t *testing.T) {
	for _, tc := range []struct {
		name string
		ask  func(h Element) bool
	}{
		{"FileDragOver", func(h Element) bool { return h.FileDragOver() }},
		{"DroppedFiles", func(h Element) bool { return len(h.DroppedFiles()) > 0 }},
		{"PressedOutside", func(h Element) bool { return h.PressedOutside() }},
		{"ClickModifiers", func(h Element) bool { return h.ClickModifiers() != 0 }},
		{"DragOffset", func(h Element) bool { return h.DragOffset() != Point{} }},
		{"Settled", func(h Element) bool { return !h.Settled("k") }},
		{"Arrived", func(h Element) bool { return h.Arrived("k") }},
		{"Velocity", func(h Element) bool { return h.Velocity("k") != 0 }},
		{"Transitioning", func(h Element) bool { return h.Transitioning() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, rowsScene(func(c *Context) any {
				return Row(c).Height(40).Children(func() { Textf(c, "header") })
			}, func(c *Context, i int, h any) {
				r := Row(c)
				if tc.ask(h.(Element)) {
					r.Background(Hex("#ff0000"))
				}
				r.Children(func() { Textf(c, "row %d", i) })
			}))
			p.idle(3)
			p.both(func(tt *Tester) { tt.ClickAt(100, 20) })
			p.frame("click")
			p.frame("click 2")
		})
	}
}

// A row that shows the layout of the frame before: the header's Bounds, which
// the app changes, and its own element's, which settle as the frames go. A
// row kept shows what it read when it was built; these rows are built.
func TestRetainRowsReadingTheLayout(t *testing.T) {
	for _, tc := range []struct {
		name string
		show func(c *Context, h, r Element)
	}{
		{"header Bounds", func(c *Context, h, r Element) { Textf(c, "h=%v", h.Bounds().H) }},
		{"own Bounds", func(c *Context, h, r Element) { Textf(c, "w=%v", r.Bounds().W) }},
		{"header LayoutBounds", func(c *Context, h, r Element) { Textf(c, "h=%v", h.LayoutBounds().H) }},
		{"own LocalToWindow", func(c *Context, h, r Element) { x, y := r.LocalToWindow(0, 0); Textf(c, "at %v,%v", x, y) }},
		{"header WindowToLocal", func(c *Context, h, r Element) { x, _, ok := h.WindowToLocal(50, 10); Textf(c, "%v %v", x, ok) }},
		{"own InView", func(c *Context, h, r Element) { Textf(c, "%v", r.InView()) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hh := float32(40)
			p := newPair(t, 300, 400, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						h := Row(c).Height(hh).Children(func() { Textf(c, "header") })
						List(c, s, 60, func(i int) {
							r := Row(c).Padding(float32(i%3) * 4)
							r.Children(func() { tc.show(c, h, r) })
						}).Grow(1)
					})
				}
			})
			p.idle(3)
			hh = 60
			p.frame("taller")
			p.frame("taller 2")
			p.frame("taller 3")
		})
	}
}

// Rows that only ask of themselves stay kept with these queries around: the
// ones that cannot be wrong for an element inside the row are not marked.
func TestRetainRowsAskingInsideStayKept(t *testing.T) {
	p := newPair(t, 300, 400, rowsScene(func(c *Context) any {
		return Row(c).Height(40).Children(func() { Textf(c, "header") })
	}, func(c *Context, i int, _ any) {
		r := Row(c).Padding(4)
		if r.Clicked() || r.Changed() || r.FileDragOver() || r.Settled("k") && r.Velocity("k") != 0 || r.ClickModifiers() != 0 {
			r.Background(Hex("#ff0000"))
		}
		r.Children(func() { Textf(c, "row %d", i) })
	}))
	p.idle(3)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
		t.Errorf("idle: %d rows built, %d kept; want none built", b, k)
	}
}

// seen counts the rows whose callback saw an event, by the twin that kept rows
// and the one that built them all.
var seenKept, seenBuilt atomic.Int64

func countSeen(c *Context) {
	if c.rt.noRetain {
		seenBuilt.Add(1)
	} else {
		seenKept.Add(1)
	}
}

// The events of one frame reach the rows that ask of the header (a TextInput):
// a pass can clear them, so the pictures do not tell, but the number of rows
// that saw them does. The twin that keeps rows must see as many as the other.
func TestRetainRowsSeeTheEventsOfTheHeader(t *testing.T) {
	for _, tc := range []struct {
		name string
		ask  func(h Element) bool
		do   func(tt *Tester)
	}{
		{"Changed", func(h Element) bool { return h.Changed() }, func(tt *Tester) { tt.Type("x") }},
		{"Submitted", func(h Element) bool { return h.Submitted() }, func(tt *Tester) { tt.Key(0, KeyEnter) }},
		{"Shortcut", func(h Element) bool { return h.Shortcut(Cmd, KeyK) }, func(tt *Tester) { tt.Key(Cmd, KeyK) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, func() func(c *Context) {
				var text string
				sel := -1
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Selected: &sel}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						h := TextInput(c, &text)
						List(c, s, 30, func(i int) {
							r := Row(c).Padding(4)
							if tc.ask(h) {
								r.Background(Hex("#ff0000"))
								countSeen(c)
							}
							r.Children(func() { Textf(c, "row %d", i) })
						}).Grow(1)
					})
				}
			})
			p.idle(2)
			p.both(func(tt *Tester) { tt.ClickAt(50, 15) })
			p.frame("focus")
			p.idle(2)
			seenKept.Store(0)
			seenBuilt.Store(0)
			p.both(tc.do)
			p.compare("do")
			p.frame("after")
			if seenKept.Load() != seenBuilt.Load() {
				t.Errorf("rows that saw the event: %d with kept rows, %d with every row built", seenKept.Load(), seenBuilt.Load())
			}
			if seenBuilt.Load() == 0 {
				t.Errorf("no row saw the event: the test does not test")
			}
		})
	}
}

// A row asks PressedOutside of its own element while the press is on the header:
// nothing of the row is acting, and the row still has to see it.
func TestRetainRowAsksPressedOutsideOfItself(t *testing.T) {
	p := newPair(t, 300, 400, rowsScene(func(c *Context) any {
		return Row(c).Height(40).Children(func() { Textf(c, "header") })
	}, func(c *Context, i int, _ any) {
		r := Row(c).Padding(4)
		if r.PressedOutside() {
			r.Background(Hex("#ff0000"))
			countSeen(c)
		}
		r.Children(func() { Textf(c, "row %d", i) })
	}))
	p.idle(3)
	seenKept.Store(0)
	seenBuilt.Store(0)
	p.both(func(tt *Tester) { tt.Press(100, 20) })
	p.compare("press on the header")
	p.frame("after")
	if seenKept.Load() != seenBuilt.Load() || seenBuilt.Load() == 0 {
		t.Errorf("rows that saw the press: %d with kept rows, %d with every row built", seenKept.Load(), seenBuilt.Load())
	}
}

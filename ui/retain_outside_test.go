package ui

import (
	"fmt"
	"math/rand"
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
				Row(c).Height(30).Children(func() { Textf(c, "head") })
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
		p.frame(fmt.Sprintf("%d %s", step, what))
		p.frame(fmt.Sprintf("%d %s +1", step, what))
	}
	if p.kept.rt.rowsKept == 0 {
		t.Fatalf("no row kept: the walk tests nothing")
	}
}

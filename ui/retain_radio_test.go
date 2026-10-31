package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

type radioSize int8

type radioFloat float64

type radioPair struct{ a, b int }

// radioApp is the data of one twin: the choice of each kind of T, and one per
// row for the groups inside a row.
type radioApp struct {
	i    int
	s    string
	e    radioSize
	f    float64
	st   radioPair
	rows []int
}

type radioScene struct{ apps []*radioApp }

func (r *radioScene) set(f func(a *radioApp)) {
	for _, a := range r.apps {
		f(a)
	}
}

// mk makes a list of n rows of one radio button each, whose type is kind;
// group puts one RadioGroup around the list ("around"), or one in each row
// ("rows"), or none.
func (r *radioScene) mk(kind, group string, n int) func() func(c *Context) {
	return func() func(c *Context) {
		a := &radioApp{rows: make([]int, n)}
		r.apps = append(r.apps, a)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		radio := func(c *Context, i int, v int) {
			label := fmt.Sprint("Option ", v)
			switch kind {
			case "int":
				Radio(c, &a.i, v, label)
			case "string":
				Radio(c, &a.s, fmt.Sprint(v), label)
			case "enum":
				Radio(c, &a.e, radioSize(v), label)
			case "float":
				Radio(c, &a.f, float64(v), label)
			case "struct":
				Radio(c, &a.st, radioPair{v, 1}, label)
			case "row":
				Radio(c, &a.rows[i], v, label)
			}
		}
		list := func(c *Context) Element {
			return List(c, s, n, func(i int) {
				Row(c).Padding(4).Gap(8).Children(func() {
					Textf(c, "row %d", i)
					if group == "rows" {
						RadioGroup(c, func() {
							for v := range 3 {
								radio(c, i, v)
							}
						}).Label(fmt.Sprint("Group ", i))
					} else {
						radio(c, i, i)
					}
				})
			}).Grow(1)
		}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				if group == "around" {
					RadioGroup(c, func() { list(c) }).Fill().Label("Options")
				} else {
					list(c)
				}
			})
		}
	}
}

// sameAccess fails when what assistive technology sees differs between the
// twins, the checked state of a radio button among it.
func (p *pair) sameAccess(what string) {
	p.t.Helper()
	view := func(tt *Tester) (out []string) {
		for _, n := range tt.Nodes() {
			out = append(out, fmt.Sprint(n.Role, n.Label, n.Value, n.Checked, n.Mixed, n.Selected, n.Disabled))
		}
		return out
	}
	a, b := view(p.kept), view(p.built)
	if len(a) != len(b) {
		p.t.Fatalf("after %s: the tree has %d nodes kept, %d built", what, len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			p.t.Fatalf("after %s: node %d is %q kept, %q built", what, i, a[i], b[i])
		}
	}
}

func (p *pair) keptRows(f func()) (kept int) {
	before := p.kept.rt.rowsKept
	f()
	return p.kept.rt.rowsKept - before
}

// The choice of a radio button moves by a click and by the arrows, across
// rows of a group around the list, some off screen: the twin that keeps the
// rows shows, to the eye and to assistive technology, what the twin that builds
// them all does, and keeps the rows of the buttons it does not touch.
func TestRetainRadioRowsFollowTheChoice(t *testing.T) {
	for _, kind := range []string{"int", "string", "enum"} {
		t.Run(kind, func(t *testing.T) {
			const n = 40
			var r radioScene
			p := newPair(t, 400, 300, r.mk(kind, "around", n))
			p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
			p.idle(4)
			if k := p.keptRows(func() { p.frame("idle") }); k == 0 {
				t.Fatalf("idle: no row kept; the rows of radio buttons are kept at rest")
			}
			step := func(what string, f func(tt *Tester)) {
				t.Helper()
				p.both(f)
				p.frame(what)
				p.frame(what + " +1")
				p.sameAccess(what)
			}
			step("click", func(tt *Tester) {
				if err := tt.Click("Option 3"); err != nil {
					t.Fatal(err)
				}
			})
			for i := range 3 {
				step(fmt.Sprint("down ", i), func(tt *Tester) { tt.Key(0, KeyDown) })
			}
			step("up", func(tt *Tester) { tt.Key(0, KeyUp) })
			step("end", func(tt *Tester) { tt.Key(0, KeyEnd) })
			// The last row is the choice and the first is off screen: the arrow
			// wraps to it.
			step("down wraps", func(tt *Tester) { tt.Key(0, KeyDown) })
			step("up wraps", func(tt *Tester) { tt.Key(0, KeyUp) })
			step("home", func(tt *Tester) { tt.Key(0, KeyHome) })
			if k := p.keptRows(func() { p.idle(2) }); k == 0 {
				t.Errorf("at rest after the keys: no row kept")
			}
		})
	}
}

// The app moves the choice without a word (no RowVersion): the row that was
// chosen and the one that is, kept, show it.
func TestRetainRadioRowsFollowTheApp(t *testing.T) {
	const n = 30
	var r radioScene
	p := newPair(t, 400, 300, r.mk("int", "around", n))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	for _, v := range []int{2, 5, 2, 0} {
		r.set(func(a *radioApp) { a.i = v })
		p.frame(fmt.Sprint("the app chose ", v))
		p.frame(fmt.Sprint("the app chose ", v, " +1"))
		p.sameAccess(fmt.Sprint("the app chose ", v))
	}
	r.set(func(a *radioApp) { a.i = 7; a.i = 0 }) // changed and changed back
	built := p.kept.rt.rowsBuilt
	p.frame("changed back")
	if got := p.kept.rt.rowsBuilt - built; got > 2 {
		t.Errorf("a choice changed back before the frame built %d rows", got)
	}
}

// A group in each row, its buttons sharing one choice of the row.
func TestRetainRadioGroupInsideARow(t *testing.T) {
	const n = 30
	var r radioScene
	p := newPair(t, 500, 300, r.mk("row", "rows", n))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	if k := p.keptRows(func() { p.frame("idle") }); k == 0 {
		t.Fatalf("idle: no row kept")
	}
	step := func(what string, f func(tt *Tester)) {
		t.Helper()
		p.both(f)
		p.frame(what)
		p.frame(what + " +1")
		p.sameAccess(what)
	}
	step("click", func(tt *Tester) {
		if err := tt.Click("Option 2"); err != nil {
			t.Fatal(err)
		}
	})
	step("up", func(tt *Tester) { tt.Key(0, KeyUp) })
	step("down", func(tt *Tester) { tt.Key(0, KeyDown) })
	r.set(func(a *radioApp) { a.rows[1] = 2 })
	p.frame("the app chose in a row")
	p.sameAccess("the app chose in a row")
}

// Other types than integers, strings and bools are compared by the row's
// callback in every frame, not recorded: their rows are built.
func TestRetainRadioOfOtherTypesIsBuilt(t *testing.T) {
	for _, kind := range []string{"float", "struct"} {
		t.Run(kind, func(t *testing.T) {
			var r radioScene
			p := newPair(t, 400, 300, r.mk(kind, "around", 30))
			p.idle(4)
			if k := p.keptRows(func() { p.frame("idle") }); k != 0 {
				t.Errorf("%d rows kept; rows of radio buttons of a %s are built in every frame", k, kind)
			}
			r.set(func(a *radioApp) { a.f, a.st = 3, radioPair{3, 1} })
			p.frame("the app chose")
			p.frame("the app chose +1")
		})
	}
}

// Radio over a bool and a 16-bit enum in kept rows: the app flips them, and a
// byte next to the enum changes, which is not the choice.
func TestRetainRadioBoolAndUint16(t *testing.T) {
	type u16 uint16
	type app struct {
		b bool
		e u16
		n int8 // next to the enum
	}
	var apps []*app
	p := newPair(t, 400, 300, func() func(c *Context) {
		a := &app{}
		apps = append(apps, a)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			List(c, s, 20, func(i int) {
				Row(c).Padding(4).Children(func() {
					Radio(c, &a.b, i%2 == 0, fmt.Sprint("B ", i))
					Radio(c, &a.e, u16(i+300), fmt.Sprint("E ", i))
				})
			}).Fill()
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	if k := p.keptRows(func() { p.frame("idle") }); k == 0 {
		t.Fatal("no row kept")
	}
	// 302+256 differs from 302 in the high byte only.
	for _, v := range []u16{302, 305, 302 + 256} {
		for _, a := range apps {
			a.b = !a.b
			a.e = v
		}
		p.frame(fmt.Sprint("app ", v))
		p.frame(fmt.Sprint("app ", v, " +1"))
		p.sameAccess(fmt.Sprint("app ", v))
	}
	for _, a := range apps {
		a.n = 9
	}
	if k := p.keptRows(func() { p.frame("neighbor") }); k == 0 {
		t.Error("a change of the byte next to the enum rebuilt every row")
	}
	p.both(func(tt *Tester) { tt.Click("B 3") })
	p.frame("click")
	p.sameAccess("click")
}

// The dot of a radio button shows: a checked one has a dot in its middle that
// its ring does not, differing from an unchecked one and from the window; the
// border of an unchecked one changes under the pointer. (No picture to compare
// with: it would differ between fonts and systems.)
func TestRadioDotShows(t *testing.T) {
	sel := 1
	var row *node
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Gap(8).Children(func() {
			row = coreRadio(c, &sel, 1, "Checked")
			coreRadio(c, &sel, 2, "Unchecked")
		})
	}, 200, 120)
	tt.Frame()
	tt.Frame()
	dot := func(r *node) Rect { return r.first.Bounds() }
	pixel := func(x, y float32) [4]uint8 {
		c := tt.Image().RGBAAt(int(x), int(y))
		return [4]uint8{c.R, c.G, c.B, c.A}
	}
	on := dot(row)
	cx, cy := on.X+on.W/2, on.Y+on.H/2
	centre, ring, window := pixel(cx, cy), pixel(on.X+2, cy), pixel(1, 1)
	if centre == ring {
		t.Errorf("a checked radio button has no dot in its middle: %v at the centre and in the ring", centre)
	}
	if ring == window {
		t.Errorf("a checked radio button looks like the window: ring %v, window %v", ring, window)
	}
	off := dot(row.next)
	ocy := off.Y + off.H/2
	if pixel(off.X+2, ocy) == ring {
		t.Errorf("an unchecked radio button is filled as a checked one is: %v", ring)
	}
	border := func() [4]uint8 { return pixel(off.X, ocy) }
	resting := border()
	tt.Move(off.X+off.W/2, ocy)
	tt.Frame()
	tt.Frame()
	if hovered := border(); hovered == resting {
		t.Errorf("the border of an unchecked radio button does not change under the pointer: %v", resting)
	}
}

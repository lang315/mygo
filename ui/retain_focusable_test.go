package ui

import (
	"bytes"
	"fmt"
	"image"
	"math/rand"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// focusRows: a list of rows that hold a Button, a Link, a Toggle or a Checkbox
// (by index), whose state each twin keeps for itself and which RowVersion
// versions, as an app must: the checked state, the count of presses, and a
// number the test bumps.
type focusRows struct {
	on     [2][]bool
	clicks [2][]int
	bump   [2][]uint64
	// note is shown in the row and not in its version: a change of it is the app
	// not saying so.
	note [2][]int
	twin int
	// versionOn: RowVersion holds the checked state; false, the app does not say
	// (the control versions itself).
	versionOn bool
}

func newFocusRows(n int) *focusRows {
	r := &focusRows{versionOn: true}
	for k := range 2 {
		r.on[k], r.clicks[k], r.bump[k], r.note[k] = make([]bool, n), make([]int, n), make([]uint64, n), make([]int, n)
	}
	return r
}

func (r *focusRows) scene(n int) func() func(c *Context) {
	return func() func(c *Context) {
		me := r.twin
		r.twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 {
			v := r.bump[me][i]<<20 + uint64(r.clicks[me][i])<<1
			if r.versionOn && r.on[me][i] {
				v |= 1
			}
			return v
		}}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d clicks %d note %d", i, r.clicks[me][i], r.note[me][i])
						switch i % 4 {
						case 0:
							if Button(c, "press").Clicked() {
								r.clicks[me][i]++
							}
						case 1:
							Link(c, "link", "https://example.com")
						case 2:
							Toggle(c, &r.on[me][i], "toggle")
						default:
							Checkbox(c, &r.on[me][i], "check")
						}
					})
				}).Grow(1)
			})
		}
	}
}

// sameAccess compares the accessibility trees of the twins: role, name, value
// and states of every node.
func sameAccess(kept, built *Tester) string {
	a, b := kept.h.access.Nodes, built.h.access.Nodes
	if len(a) != len(b) {
		return fmt.Sprintf("accessibility trees of %d and %d nodes", len(a), len(b))
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.ID != y.ID || x.Role != y.Role || x.Label != y.Label || x.Value != y.Value || x.States != y.States || x.PosInSet != y.PosInSet {
			return fmt.Sprintf("accessibility node %d: kept %+v, built %+v", i, x, y)
		}
	}
	return ""
}

// compareFocus compares what a frame left in the twins besides the pictures:
// the order of the elements that take the focus, the focused element and the
// accessibility tree.
func (p *pair) compareFocus(what string) {
	p.t.Helper()
	p.compare(what)
	if !slices.Equal(p.kept.rt.focusOrder, p.built.rt.focusOrder) {
		p.t.Fatalf("after %s: focus order differs: kept %v, built %v", what, p.kept.rt.focusOrder, p.built.rt.focusOrder)
	}
	if p.kept.rt.focused != p.built.rt.focused {
		p.t.Fatalf("after %s: focused %d and %d", what, p.kept.rt.focused, p.built.rt.focused)
	}
	if d := sameAccess(p.kept, p.built); d != "" {
		p.t.Fatalf("after %s: %s", what, d)
	}
}

// A random walk of the keys that move the focus and act on it, the pointer, the
// wheel, the window's focus and the app's own changes (versioned) over rows with
// controls that take the focus: the twin that keeps the rows must draw, order the
// focus and describe to assistive technology what the one that builds them all does.
func TestRetainFocusableRowsRandomWalk(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) { walkFocusable(t, seed) })
	}
}

func walkFocusable(t *testing.T, seed int64) {
	const n = 60
	r := newFocusRows(n)
	r.versionOn = seed%2 == 1 // on even seeds the app does not version the checked state
	p := newPair(t, 500, 400, r.scene(n))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(3)
	rng := rand.New(rand.NewSource(seed))
	for step := range 80 {
		x, y := float32(rng.Intn(520)-10), float32(rng.Intn(420)-10)
		var what string
		switch k := rng.Intn(15); k {
		case 0, 1:
			what = "tab"
			p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
		case 2:
			what = "shift tab"
			p.both(func(tt *Tester) { tt.Key(Shift, KeyTab) })
		case 3, 4:
			what = "space"
			p.both(func(tt *Tester) { tt.Key(0, KeySpace) })
		case 5:
			what = "enter"
			p.both(func(tt *Tester) { tt.Key(0, KeyEnter) })
		case 6:
			what = fmt.Sprintf("click %v,%v", x, y)
			p.both(func(tt *Tester) { tt.ClickAt(x, y) })
		case 7, 8:
			what = fmt.Sprintf("move %v,%v", x, y)
			p.both(func(tt *Tester) { tt.Move(x, y) })
		case 9:
			what = "arrow"
			key := []Key{KeyDown, KeyUp, KeyLeft, KeyRight}[rng.Intn(4)]
			p.both(func(tt *Tester) { tt.Key(0, key) })
		case 10:
			dy := float32(rng.Intn(120) - 60)
			what = fmt.Sprintf("scroll %v", dy)
			p.both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
		case 11:
			what = "blur and focus"
			p.both(func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
		case 12:
			i := rng.Intn(n)
			what = fmt.Sprintf("bump %d", i)
			for k := range 2 {
				r.bump[k][i]++
			}
		case 13:
			i := rng.Intn(n)
			what = fmt.Sprintf("app changes %d", i)
			for k := range 2 {
				r.on[k][i] = !r.on[k][i]
			}
		default:
			what = "app checks all" // a "select all" outside the list, no input
			for k := range 2 {
				for i := range r.on[k] {
					r.on[k][i] = true
				}
			}
		}
		p.compareFocus(fmt.Sprintf("%d %s", step, what))
		p.compareFocus(fmt.Sprintf("%d %s +1", step, what))
	}
	if p.kept.rt.rowsKept == 0 {
		t.Fatalf("no row kept: the walk tests nothing")
	}
}

// The focus ring of a row that is kept: it shows when the focus comes by the
// keyboard (Tab) and by the mouse, and goes with the focus, in the picture and
// not in the row's key.
func TestRetainFocusableRowsDrawTheFocusRing(t *testing.T) {
	const n = 60
	r := newFocusRows(n)
	p := newPair(t, 500, 400, r.scene(n))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept")
	}
	before := p.kept.Image()
	p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
	p.compareFocus("tab")
	if bytesEqualImages(before, p.kept.Image()) {
		t.Errorf("Tab drew no focus ring on a kept row")
	}
	p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
	p.compareFocus("tab again")
	p.both(func(tt *Tester) { tt.ClickAt(40, 40) }) // the mouse: a row button
	p.compareFocus("click")
	p.frame("click 2")
	p.both(func(tt *Tester) { tt.SetFocused(false) })
	p.compareFocus("window blurred")
	p.both(func(tt *Tester) { tt.SetFocused(true) })
	p.compareFocus("window focused")
}

func bytesEqualImages(a, b *image.RGBA) bool { return bytes.Equal(a.Pix, b.Pix) }

// A change of the app's data that does not come through input and that
// RowVersion does not say is the contract broken: the row that shows it may stay
// as it was, and only that row; the others are what a build shows. Said by the
// app (a bump), it shows at once. (A control's own value, the *bool of a check
// box, is noticed by the control: that row is built either way.)
func TestRetainFocusableRowsAnUnversionedChangeStaysInItsRow(t *testing.T) {
	const n = 60
	r := newFocusRows(n)
	p := newPair(t, 500, 400, r.scene(n))
	p.idle(5)
	for k := range 2 {
		r.note[k][4] = 5 // no bump: RowVersion is not told
	}
	p.both(func(tt *Tester) { tt.Frame() })
	a, b := p.kept.Texts(), p.built.Texts()
	if len(a) != len(b) {
		t.Fatalf("%d and %d texts", len(a), len(b))
	}
	var differ []string
	for i := range a {
		if a[i] != b[i] {
			differ = append(differ, fmt.Sprintf("%q / %q", a[i], b[i]))
		}
	}
	if len(differ) != 1 || slices.IndexFunc(a, func(s string) bool { return s == "row 4 clicks 0 note 0" }) < 0 {
		t.Errorf("an unversioned change of row 4 left these texts different: %v; want row 4's only", differ)
	}
	for k := range 2 {
		r.bump[k][4]++ // the app says so
	}
	p.both(func(tt *Tester) { tt.Frame() })
	p.compare("versioned")
}

func (p *pair) sameTexts(what string) {
	p.t.Helper()
	a, b := p.kept.Texts(), p.built.Texts()
	if !slices.Equal(a, b) {
		var d []string
		for i := range min(len(a), len(b)) {
			if a[i] != b[i] {
				d = append(d, fmt.Sprintf("%q/%q", a[i], b[i]))
			}
		}
		p.t.Fatalf("after %s: texts differ (kept/built): %v (len %d/%d)", what, d, len(a), len(b))
	}
}

func (p *pair) all(what string) {
	p.t.Helper()
	p.sameTexts(what)
	p.compareFocus(what)
}

// A row reads whether its own button has the focus, and whether it shows, as it
// builds: nothing records that read but the row being acting when the focus is
// on it or just left it.
func TestRetainRowsReadingTheFocusOfTheirOwnButton(t *testing.T) {
	const n = 30
	p := newPair(t, 400, 300, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						b := Button(c, "b")
						Textf(c, "row %d focused %v visible %v", i, b.Focused(), b.FocusVisible())
					})
				}).Grow(1)
			})
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(3)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept")
	}
	for k := range 4 {
		p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
		p.all(fmt.Sprintf("tab %d", k))
		p.frame(fmt.Sprintf("tab %d +1", k))
		p.sameTexts(fmt.Sprintf("tab %d +1", k))
	}
	p.both(func(tt *Tester) { tt.ClickAt(20, 15) })
	p.all("click")
	p.both(func(tt *Tester) { tt.Key(Shift, KeyTab) })
	p.all("shift tab")
	p.both(func(tt *Tester) { tt.SetFocused(false) })
	p.all("blur")
	p.both(func(tt *Tester) { tt.SetFocused(true) })
	p.all("focus")
}

// The app changes the *bool of check boxes without input and without saying so in
// RowVersion: one row, all rows (a "select all" outside the list), and a change
// that is changed back before the frame (the value is what it was, the rows
// are kept).
func TestRetainRowsOfCheckBoxesFollowTheirValue(t *testing.T) {
	const n = 30
	var on [2][]bool
	on[0], on[1] = make([]bool, n), make([]bool, n)
	twin := 0
	p := newPair(t, 400, 300, func() func(c *Context) {
		me := twin
		twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						if i%2 == 0 {
							Checkbox(c, &on[me][i], fmt.Sprint("item ", i))
						} else {
							Toggle(c, &on[me][i], fmt.Sprint("item ", i))
						}
					})
				}).Grow(1)
			})
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept")
	}
	set := func(f func(k int)) {
		for k := range 2 {
			f(k)
		}
	}
	set(func(k int) { on[k][3] = true })
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("one row")
	set(func(k int) {
		for i := range on[k] {
			on[k][i] = true
		}
	})
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("all rows")
	built := p.kept.rt.rowsBuilt
	set(func(k int) { on[k][5] = !on[k][5]; on[k][5] = !on[k][5] }) // changed and changed back
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("changed back")
	if p.kept.rt.rowsBuilt != built {
		t.Errorf("a value changed back before the frame built %d rows", p.kept.rt.rowsBuilt-built)
	}
	set(func(k int) {
		for i := range on[k] {
			on[k][i] = false
		}
	})
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("none")
}

// A CheckboxGroup around a List counts the boxes its rows build: a row linked adds
// none, so the group's click would miss them.
func TestRetainRowsInACheckboxGroupAreBuilt(t *testing.T) {
	const n = 8
	var on [2][]bool
	on[0], on[1] = make([]bool, n), make([]bool, n)
	twin := 0
	p := newPair(t, 400, 400, func() func(c *Context) {
		me := twin
		twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 {
			if on[me][i] {
				return 2
			}
			return 1
		}}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				CheckboxGroup(c, "all", func() {
					List(c, s, n, func(i int) {
						Checkbox(c, &on[me][i], fmt.Sprint("item ", i))
					}).Height(250)
				})
			})
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(3)
	p.both(func(tt *Tester) {
		if err := tt.Click("all"); err != nil {
			t.Fatal(err)
		}
	})
	p.both(func(tt *Tester) { tt.Frame(); tt.Frame() })
	if !slices.Equal(on[0], on[1]) {
		t.Errorf("the group's click checked kept %v, built %v", on[0], on[1])
	}
	p.all("group")
}

// Focus asked in a row's callback is an action of the build: the row that asks
// is built, so the focus goes where the app says. The app says by RowVersion
// when the row that asks changes.
func TestRetainRowsThatAskForTheFocus(t *testing.T) {
	const n = 20
	focusRow := -1
	var vers [2]uint64
	twin := 0
	p := newPair(t, 400, 300, func() func(c *Context) {
		me := twin
		twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 {
			if i == focusRow {
				return vers[me] + 1
			}
			return 0
		}}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Children(func() {
						b := Button(c, fmt.Sprint("button ", i))
						if i == focusRow {
							b.Focus()
						}
					})
				}).Grow(1)
			})
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	for _, row := range []int{3, 7, 3} {
		focusRow = row
		for k := range vers {
			vers[k]++
		}
		p.both(func(tt *Tester) { tt.Frame() })
		p.compareFocus(fmt.Sprint("focus row ", row))
		p.frame(fmt.Sprint("focus row ", row, " +1"))
		// The app keeps asking in every frame (the version does not move): the
		// row is built, asks again, and the focus the user moved goes back.
		p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
		p.compareFocus(fmt.Sprint("tab from row ", row))
		p.frame(fmt.Sprint("tab from row ", row, " +1"))
	}
}

// Two places ask for the focus in the same build, an element before the list and
// a row that keeps asking: the one built last wins. A row that is kept does not
// ask, so Focus makes the row built.
func TestRetainRowsThatKeepAskingForTheFocusWinOverEarlierOnes(t *testing.T) {
	grab := false
	p := newPair(t, 400, 300, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				out := Button(c, "outside")
				if grab {
					out.Focus()
				}
				List(c, s, 10, func(i int) {
					Row(c).Padding(4).Children(func() {
						b := Button(c, fmt.Sprint("button ", i))
						if i == 3 {
							b.Focus()
						}
					})
				}).Grow(1)
			})
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	grab = true
	p.both(func(tt *Tester) { tt.Frame() })
	p.compareFocus("outside asks too")
	p.frame("outside asks too +1")
}

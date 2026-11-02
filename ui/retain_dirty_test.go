package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// dirtyScene is a list of rows with what a user acts on (a button with a
// tooltip, a text input, a context menu, a place to drop files), beside a
// label and a progress bar the app changes in every frame: the rows are kept
// while nothing touches them, and built again after each thing a user does in
// them (rowKeep.dirty: a write to a state field, or a change of the focus, the
// hover, the press, a menu, a tooltip, a drag).
type dirtyScene struct {
	text    [2][]string
	clicks  [2][]int
	dropped [2][]int
	label   [2]int
	twin    int
}

const dirtyRows = 30

func (d *dirtyScene) mk() func() func(c *Context) {
	return func() func(c *Context) {
		me := d.twin
		d.twin++
		d.text[me] = make([]string, dirtyRows)
		d.clicks[me] = make([]int, dirtyRows)
		d.dropped[me] = make([]int, dirtyRows)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 {
			return uint64(d.clicks[me][i])<<8 | uint64(d.dropped[me][i])
		}}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				Textf(c, "label %d", d.label[me]).Tooltip("the label")
				Progress(c, float64(d.label[me]%10)/10)
				List(c, s, dirtyRows, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d clicks %d dropped %d", i, d.clicks[me][i], d.dropped[me][i])
						switch i % 4 {
						case 3:
							// A scroll area in the row, and a hover the row reads.
							Scroll(c).Height(30).Width(120).Children(func() {
								for j := range 6 {
									Textf(c, "line %d", j)
								}
							})
							if Text(c, "hot").Hovered() {
								Text(c, "!")
							}
						case 0:
							if Button(c, "press").Clicked() {
								d.clicks[me][i]++
							}
						case 1:
							TextInput(c, &d.text[me][i])
						default:
							Text(c, "menu").ContextMenu(func(m *Menu) {
								m.Item("one")
								m.Item("two")
							})
							Text(c, "drop here").DroppedFiles()
						}
					})
				}).Grow(1)
			})
		}
	}
}

// dirtyRowsKept draws frames until a frame keeps rows, and fails when none
// does within the limit: a row left dirty for good would never be kept again.
func (p *pair) rowsKeptAgain(what string) {
	p.t.Helper()
	for i := range 8 {
		if p.keptRows(func() { p.frame(fmt.Sprintf("%s: rest %d", what, i)) }) > 0 {
			return
		}
	}
	p.t.Fatalf("%s: no row is kept again", what)
}

// dirtyRun is a scene at rest, rows kept, and what the scenarios act on: its
// own pair for each scenario, since the focus a scenario leaves in a row keeps
// that row built.
type dirtyRun struct {
	t        *testing.T
	d        *dirtyScene
	p        *pair
	bx, by   float32 // a row's button
	ix, iy   float32 // a row's input
	mx, my   float32 // a row's menu
	dx, dy   float32 // a row's drop place
	sx, sy   float32 // a row's scroll area
	hx, hy   float32 // a row that reads the hover
	btnLabel string
}

func newDirtyRun(t *testing.T) *dirtyRun {
	t.Helper()
	r := &dirtyRun{t: t, d: &dirtyScene{}}
	r.p = newPair(t, 500, 400, r.d.mk())
	r.p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	r.p.idle(3)
	r.p.rowsKeptAgain("start")
	// The scenarios act on rows that are kept at rest: every row in view.
	if n := r.built(func() { r.p.frame("rest") }); n != 0 {
		t.Fatalf("a frame at rest built %d rows: the scenarios would act on rows that are never kept", n)
	}
	at := func(s string) (float32, float32) {
		b, ok := r.p.kept.Find(s)
		if !ok {
			t.Fatalf("%q is not in view", s)
		}
		return b.X + b.W/2, b.Y + b.H/2
	}
	r.bx, r.by = at("press")
	r.sx, r.sy = at("line 1")
	r.hx, r.hy = at("hot")
	r.mx, r.my = at("menu")
	r.dx, r.dy = at("drop here")
	// The first input in view has no text to find: it is below its row's text.
	b, _ := r.p.kept.Find("row 1 clicks 0 dropped 0")
	r.ix, r.iy = b.X+b.W+40, b.Y+b.H/2
	return r
}

func (r *dirtyRun) step(what string, f func(tt *Tester)) {
	r.t.Helper()
	r.p.both(f)
	r.p.compareFocus(what)
	r.p.compareFocus(what + " +1")
}

// tick changes the label and the progress beside the list, as an app does in
// every frame.
func (r *dirtyRun) tick() {
	for k := range 2 {
		r.d.label[k]++
	}
}

func (r *dirtyRun) rest(what string) { r.t.Helper(); r.p.rowsKeptAgain(what) }

// built is how many rows the next frame builds in the kept twin.
func (r *dirtyRun) built(f func()) int {
	before := r.p.kept.rt.rowsBuilt
	f()
	return r.p.kept.rt.rowsBuilt - before
}

// What a user does in a kept row builds it again (the twin that builds every
// row draws as the one that keeps them), and the rows are kept again once the
// thing is over. okRow panics (checkDirty) when a row it finds clean is pending.
func TestRetainRowDirtyScenarios(t *testing.T) {
	t.Run("focus into a row, then Tab out of the table", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("click the button", func(tt *Tester) { tt.ClickAt(r.bx, r.by) })
		for i := range 8 {
			r.tick()
			r.step(fmt.Sprint("tab ", i), func(tt *Tester) { tt.Key(0, KeyTab) })
		}
		r.step("blur", func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
		r.rest("after the tabs")
	})
	t.Run("hover a row that reads the hover, then leave the window", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("move onto the row", func(tt *Tester) { tt.Move(r.hx, r.hy) })
		r.tick()
		r.step("move off the window", func(tt *Tester) { tt.Move(-5, -5) })
		r.rest("after the pointer left")
	})
	t.Run("hover a row, then leave the window", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("move onto the row", func(tt *Tester) { tt.Move(r.bx, r.by) })
		r.tick()
		r.step("move off the window", func(tt *Tester) { tt.Move(-5, -5) })
		r.rest("after the pointer left")
	})
	t.Run("press in a row, drag out, release", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("press", func(tt *Tester) { tt.Press(r.bx, r.by) })
		for _, off := range []float32{20, 80, 200} {
			r.tick()
			r.step(fmt.Sprint("drag ", off), func(tt *Tester) { tt.Move(r.bx+off, r.by+off) })
		}
		r.step("release", func(tt *Tester) { tt.Release(r.bx+200, r.by+200) })
		r.rest("after the release")
	})
	t.Run("a menu from a row, closed with Esc", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("right click", func(tt *Tester) { tt.RightClickAt(r.mx, r.my) })
		r.tick()
		r.step("esc", func(tt *Tester) { tt.Key(0, KeyEscape) })
		r.rest("after the menu")
	})
	t.Run("a tooltip shown, then hidden by the clock", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("hover the button", func(tt *Tester) { tt.Move(r.bx, r.by) })
		r.step("wait for the tooltip", func(tt *Tester) { tt.Advance(1500 * time.Millisecond) })
		r.tick()
		r.step("leave", func(tt *Tester) { tt.Move(-5, -5) })
		r.step("wait", func(tt *Tester) { tt.Advance(1500 * time.Millisecond) })
		r.rest("after the tooltip")
	})
	t.Run("a key presses a row's button", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("tab to the button", func(tt *Tester) { tt.Key(0, KeyTab) })
		r.step("space", func(tt *Tester) { tt.Key(0, KeySpace) })
		if r.d.clicks[0][0] == 0 {
			t.Fatalf("the key pressed nothing: %v", r.d.clicks[0])
		}
		r.rest("after the key")
	})
	t.Run("typing and a composition", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("click the input", func(tt *Tester) { tt.ClickAt(r.ix, r.iy) })
		r.step("type", func(tt *Tester) { tt.Type("ab") })
		r.tick()
		r.step("compose", func(tt *Tester) { tt.Compose("ni", 2) })
		r.step("commit", func(tt *Tester) { tt.Compose("", 0); tt.Type("x") })
		if r.d.text[0][1] == "" {
			t.Fatalf("nothing typed: %q", r.d.text[0])
		}
		r.step("click away", func(tt *Tester) { tt.ClickAt(250, 390) })
		r.rest("after typing")
	})
	t.Run("files dropped on a row without a drag over first", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("hover the place", func(tt *Tester) { tt.Move(r.dx, r.dy) })
		r.rest("hovered")
		r.step("drop", func(tt *Tester) {
			tt.rt.event(platform.SurfaceEvent{Kind: platform.FileDrop, X: float64(r.dx), Y: float64(r.dy), Files: []string{"/tmp/a.txt"}})
			tt.settle()
		})
		r.rest("after the drop")
	})
	t.Run("files dropped on a row", func(t *testing.T) {
		r := newDirtyRun(t)
		// The pointer is over the place before the drop: the drop is the only change.
		r.step("hover the place", func(tt *Tester) { tt.Move(r.dx, r.dy) })
		r.rest("hovered")
		r.step("drop", func(tt *Tester) { tt.DropFiles(r.dx, r.dy, "/tmp/a.txt") })
		r.tick()
		r.rest("after the drop")
	})
	t.Run("assistive technology presses a row's button", func(t *testing.T) {
		r := newDirtyRun(t)
		found := false
		for _, n := range r.p.kept.Nodes() {
			if n.Label == "press" {
				found = true
				id := n.ID
				r.step("press", func(tt *Tester) {
					tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: id, Action: platform.AccessPress})
				})
				break
			}
		}
		if !found || r.d.clicks[0][0] == 0 {
			t.Fatalf("the action pressed nothing: found %v, clicks %v", found, r.d.clicks[0])
		}
		r.rest("after the action")
	})
	t.Run("the wheel scrolls the list", func(t *testing.T) {
		r := newDirtyRun(t)
		r.step("hover the area", func(tt *Tester) { tt.Move(r.sx, r.sy) })
		r.rest("hovered")
		for i := range 5 {
			r.tick()
			r.step(fmt.Sprint("wheel ", i), func(tt *Tester) { tt.Wheel(r.sx, r.sy, 0, -40) })
		}
		r.rest("after the wheel")
	})
	t.Run("the app changes a row's data and the label beside the list", func(t *testing.T) {
		r := newDirtyRun(t)
		for k := range 2 {
			r.d.clicks[k][3]++
		}
		r.tick()
		r.step("data", func(tt *Tester) { tt.Frame() })
		r.rest("after the data")
	})
}

package ui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

// pair runs one view in two Testers, one keeping rows and one building every
// row (noRetain), and compares what layout and commit left after each frame.
type pair struct {
	t           *testing.T
	kept, built *Tester
	steps       []string
}

// newPair makes the two Testers of w×h, each from its own mk(): state such
// as a ListState lives in mk's closure, data the test changes outside it.
func newPair(t *testing.T, w, h int, mk func() func(c *Context)) *pair {
	t.Helper()
	p := &pair{t: t, kept: NewTester(mk(), w, h), built: NewTester(mk(), w, h)}
	p.built.rt.noRetain = true
	return p
}

func (p *pair) both(f func(tt *Tester)) { f(p.kept); f(p.built) }

// compare fails on the first difference between the two.
func (p *pair) compare(what string) {
	p.t.Helper()
	p.steps = append(p.steps, what)
	d := sameFrames(p.kept, p.built, true)
	if d == "" {
		d = sameStates(p.kept, p.built)
	}
	if d == "" {
		d = sameNodes(p.kept.rt.c.root, p.built.rt.c.root, "root")
	}
	if d != "" {
		s := p.steps
		if len(s) > 6 {
			s = s[len(s)-6:]
		}
		p.t.Fatalf("after %s: %s", strings.Join(s, ", "), d)
	}
}

// frame draws a frame in both and compares them.
func (p *pair) frame(what string) {
	p.t.Helper()
	p.both(func(tt *Tester) { tt.Frame() })
	p.compare(what)
}

// idle draws n frames with nothing changed.
func (p *pair) idle(n int) {
	p.t.Helper()
	for i := range n {
		p.frame(fmt.Sprintf("idle %d", i))
	}
}

// A Table of rows with a RowVersion, put through what users do to it: a kept
// Tester and one that builds every row draw alike.
func TestRetainLayoutScenarios(t *testing.T) {
	type app struct {
		rows    []string
		vers    []uint64
		list    ListState
		dark    bool
		checked Selection[int]
	}
	var apps []*app // the kept one's, then the built one's
	p := newPair(t, 400, 600, func() func(c *Context) {
		a := &app{rows: make([]string, 3000), vers: make([]uint64, 3000)}
		for i := range a.rows {
			a.rows[i] = fmt.Sprintf("acc%06d", i)
		}
		apps = append(apps, a)
		cols := []TableColumn{{Title: "Name"}, {Title: "Status", Width: 80, Sortable: true}}
		a.list.Checked = &a.checked
		a.list.Key = func(i int) any { return i }
		a.list.RowVersion = func(i int) uint64 { return a.vers[i] }
		return func(c *Context) {
			if a.dark {
				c.SetTheme(DarkTheme())
			}
			Table(c, &a.list, cols, len(a.rows), func(r, col int) {
				if col == 0 {
					Text(c, a.rows[r])
				} else {
					Textf(c, "%d", r%7)
				}
			}).Grow(1)
		}
	})
	each := func(f func(a *app)) {
		for _, a := range apps {
			f(a)
		}
	}
	for range 10 {
		p.both(func(tt *Tester) { tt.Frame() })
	}
	p.compare("warm")
	p.idle(5)
	if p.kept.rt.rowsLaidKept == 0 || p.kept.rt.rowsCommitReplayed == 0 {
		t.Fatalf("an idle frame kept the layout of %d rows and replayed the commit of %d, want both > 0",
			p.kept.rt.rowsLaidKept, p.kept.rt.rowsCommitReplayed)
	}
	for i := range 10 {
		y := float32(120 + 40*(i%2))
		p.both(func(tt *Tester) { tt.Move(150, y) })
		p.compare(fmt.Sprintf("hover %d", i))
	}
	p.both(func(tt *Tester) { tt.Move(-10, -10) })
	p.compare("pointer away")
	for i := range 4 {
		each(func(a *app) { a.rows[5] += "x"; a.vers[5]++ })
		p.frame(fmt.Sprintf("update %d", i))
	}
	p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, 2000) })
	p.compare("scroll far")
	for i := range 10 {
		dy := float32(1 - 2*(i%2))
		p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, dy) })
		p.compare(fmt.Sprintf("scroll %v", dy))
	}
	for i, dy := range []float32{0.5, 0.5, -0.5, -0.5} {
		p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, dy) })
		p.compare(fmt.Sprintf("scroll half %d", i))
	}
	p.idle(3)
	p.both(func(tt *Tester) { tt.h.w, tt.h.h = 330, 600 })
	p.frame("narrower")
	p.idle(3)
	each(func(a *app) { a.dark = true })
	p.frame("theme")
	p.idle(3)
	each(func(a *app) { a.checked.set(3, true) })
	p.frame("check")
	p.idle(3)
	p.both(func(tt *Tester) { tt.h.w, tt.h.h = 400, 500 })
	p.frame("shorter")
	p.idle(1)
}

// keptList is a List of n rows with a RowVersion, in a column the test can
// change around it.
type keptList struct {
	n      int
	vers   []uint64
	width  float32 // 0: fill
	size   float32 // text size of the column, 0 for the theme's
	color  Color
	rtl    bool
	hidden bool
	button []bool // rows with a Button, which a copy cannot keep
}

func (k *keptList) mk() func(c *Context) {
	s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return k.vers[i] }}
	return func(c *Context) {
		col := Column(c)
		if k.width > 0 {
			col.Width(k.width)
		}
		if k.size > 0 {
			col.FontSize(k.size)
		}
		if k.color != (Color{}) {
			col.TextColor(k.color)
		}
		col.Hidden(k.hidden).Children(func() {
			List(c, s, k.n, func(i int) {
				Row(c).Children(func() {
					Textf(c, "row %d", i)
					Text(c, "·")
					if k.button != nil && k.button[i] {
						Button(c, "x")
					}
				})
			}).Grow(1)
		})
		if k.rtl {
			// After Children, on purpose: setDir then walks rows already linked
			// (the spike's hole 5). Set before, P2's dir check in keep refuses
			// the link, and the rdir guard of settle is never reached.
			col.Direction(RightToLeft)
		}
	}
}

func newKeptList(n int) *keptList { return &keptList{n: n, vers: make([]uint64, n)} }

func heldThenFlip(t *testing.T, k *keptList, flip func(), what string) *pair {
	t.Helper()
	p := newPair(t, 300, 400, k.mk)
	p.idle(4)
	if p.kept.rt.rowsLaidKept == 0 {
		t.Fatalf("no row held before %s", what)
	}
	flip()
	p.frame(what)
	if n := p.kept.rt.rowsLaidKept; n != 0 {
		t.Errorf("%s: %d rows kept their layout, want 0", what, n)
	}
	p.idle(2)
	return p
}

func TestRetainLayoutWidthChange(t *testing.T) {
	k := newKeptList(50)
	heldThenFlip(t, k, func() { k.width = 250 }, "narrower")
}

// FontSize is in textLayoutSet and not in textPaintSet (paintstyle.go), so it
// flips layRows alone, not inheritsPaint: it kills the "skip ignores layRows"
// mutation by itself.
func TestRetainLayoutInheritedTextSize(t *testing.T) { // layRows
	k := newKeptList(50)
	heldThenFlip(t, k, func() { k.size = 19 }, "text size of an ancestor")
}

func TestRetainLayoutInheritedColor(t *testing.T) { // inheritsPaint
	k := newKeptList(50)
	heldThenFlip(t, k, func() { k.color = RGB(200, 0, 0) }, "color of an ancestor")
}

func TestRetainLayoutDirectionAfterTheList(t *testing.T) { // rdir: setDir walks the linked rows
	k := newKeptList(50)
	heldThenFlip(t, k, func() { k.rtl = true }, "right to left")
}

func TestRetainLayoutListHiddenForAFrame(t *testing.T) { // at+1, setAt
	k := newKeptList(50)
	p := newPair(t, 300, 400, k.mk)
	p.idle(4)
	k.hidden = true
	p.frame("hidden")
	k.hidden = false
	p.frame("shown")
	if n := p.kept.rt.rowsLaidKept; n != 0 {
		t.Errorf("rows kept the layout of a frame they were not laid out in: %d", n)
	}
	p.idle(3)
}

func TestRetainLayoutSkippedLayoutsThenChange(t *testing.T) { // rewind
	k := newKeptList(50)
	p := newPair(t, 300, 400, k.mk)
	p.idle(8) // several layouts skipped: the states' lay.layout must not fall behind
	k.size = 17
	p.frame("text size after skipped layouts")
	p.idle(2)
}

// A frame whose rows are not kept (the paint of an ancestor changed) still
// compares them with the last layout, though layouts were skipped since: the
// layout work is that of rows built.
func TestRetainLayoutSkippedLayoutsStillCompare(t *testing.T) { // rewind in settle
	k := newKeptList(50)
	p := newPair(t, 300, 400, k.mk)
	p.idle(8)
	k.color = RGB(200, 0, 0)
	kept, built := p.kept.rt.computed, p.built.rt.computed
	p.frame("color of an ancestor")
	if dk, db := p.kept.rt.computed-kept, p.built.rt.computed-built; dk > db {
		t.Errorf("the kept rows laid out %d boxes, the rows built %d", dk, db)
	}
}

func TestRetainLayoutMovedRows(t *testing.T) { // keptPlace, moved path
	k := newKeptList(500)
	p := newPair(t, 300, 400, k.mk)
	p.idle(4)
	for i := range 10 {
		dy := float32(1)
		if i%2 == 1 {
			dy = -1
		}
		p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, dy) })
		p.frame(fmt.Sprintf("scroll %v", dy))
	}
}

// A Header pins the header of the section on top of the rows, built late
// (buildLate) while the rows around it are held.
func TestRetainLayoutRowsBuiltLate(t *testing.T) {
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
	for i := range 12 {
		p.both(func(tt *Tester) { tt.Scroll(150, 150, 0, 7) })
		p.frame(fmt.Sprintf("scroll %d", i))
		p.idle(2)
	}
}

// A reveal of an element in a held row: descendants of a held row hold window
// coordinates, reveal sums relative ones (Review Focus 1).
func TestRetainLayoutRevealInAHeldRow(t *testing.T) {
	p := newPair(t, 300, 300, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			// Away from the window's corner: the boxes of a row that was placed
			// hold window coordinates, and reveal reads relative ones.
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
	if p.kept.rt.rowsLaidKept == 0 {
		t.Fatalf("no row held")
	}
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
	if id == 0 || id != find(p.built.rt.c.root) {
		t.Fatalf("no text mid 2, or its id differs: %d", id)
	}
	p.both(func(tt *Tester) { tt.ScrollIntoView(id) })
	p.compare("reveal")
	var scrolled func(n *node) bool
	scrolled = func(n *node) bool {
		if n.flags&flagScrollX != 0 && n.st.scrollX > 0 {
			return true
		}
		for c := n.first; c != nil; c = c.next {
			if scrolled(c) {
				return true
			}
		}
		return false
	}
	if !scrolled(p.built.rt.c.root) {
		t.Fatalf("the reveal did not scroll the container: the test shows nothing")
	}
	p.idle(3)
}

// The states of a row out of view for two frames are recycled: a state must
// never point at a record that is not its row's.
func TestRetainLayoutStateRecycled(t *testing.T) {
	k := newKeptList(300)
	p := newPair(t, 300, 400, k.mk)
	check := func(what string) {
		t.Helper()
		for id, st := range p.kept.rt.states {
			if rec := st.krec; rec != nil && (rec.root == nil || len(rec.states) == 0 || rec.states[0] != st) {
				t.Fatalf("%s: state %d points at a record that is not its row's", what, id)
			}
		}
	}
	p.idle(3)
	check("held")
	for i, dy := range []float32{2000, 0, 0, -2000, 0, 0, 5, -5} {
		p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, dy) })
		p.compare(fmt.Sprintf("scroll %d", i))
		check(fmt.Sprintf("scroll %d", i))
	}
	// A row built instead of linked (the pointer is on it).
	for i, y := range []float32{60, 100, -10} {
		p.both(func(tt *Tester) { tt.Move(100, y) })
		p.compare(fmt.Sprintf("hover %d", i))
		check(fmt.Sprintf("hover %d", i))
	}
	for i := range 3 {
		k.vers[10]++ // a row copied again
		p.frame(fmt.Sprintf("version %d", i))
		check(fmt.Sprintf("version %d", i))
	}
	// A row that cannot be kept any more (it holds a button) releases its
	// record while its state lives.
	p.idle(3)
	k.button = make([]bool, k.n)
	k.button[12] = true
	k.vers[12]++
	p.frame("button")
	check("button")
	p.idle(2)
	check("idle")
}

// Two rows of one key share their states: neither may be held on the other's.
func TestRetainLayoutDuplicateKeys(t *testing.T) {
	// An app only logs the duplicate; a Tester panics.
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i / 2 }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			c.rt.strict = false
			List(c, s, 40, func(i int) { Row(c).Children(func() { Textf(c, "row %d", i) }) }).Grow(1)
		}
	})
	p.idle(4)
}

// Fitting a column builds every row to measure it, late (buildLate), among
// rows linked held.
func TestRetainLayoutFitColumn(t *testing.T) {
	p := newPair(t, 400, 300, func() func(c *Context) {
		var list ListState
		list.Key = func(i int) any { return i }
		list.RowVersion = func(int) uint64 { return 0 }
		cols := []TableColumn{{Title: "Name"}, {Title: "Status", Width: 80}}
		return func(c *Context) {
			Table(c, &list, cols, 100, func(r, col int) {
				if col == 0 {
					Textf(c, "account number %d", r*r)
				} else {
					Textf(c, "%d", r%7)
				}
			}).Grow(1)
		}
	})
	p.idle(4)
	for range 2 {
		p.both(func(tt *Tester) {
			var table *node
			var walk func(e *node)
			walk = func(e *node) {
				for ; e != nil && table == nil; e = e.next {
					if e.widget == "Table" {
						table = e
					}
					walk(e.first)
				}
			}
			walk(tt.rt.c.root)
			h := table.first.first // the header of the first column
			x, y := h.x+h.w-2, h.y+h.h/2
			tt.ClickAt(x, y)
			tt.ClickAt(x, y)
		})
		p.compare("fit")
		p.idle(3)
	}
}

// What settle and keptPlace read must hold for a row to keep its layout: a
// record that was committed in a layout before the last, or whose state is
// another record's, is laid out.
func TestRetainLayoutRecordTampered(t *testing.T) {
	for _, tc := range []struct {
		name string
		do   func(rec, other *rowKeep)
	}{
		{"layout before the last", func(rec, _ *rowKeep) { rec.lay.at-- }},
		{"state of another record", func(rec, other *rowKeep) { rec.root.st.krec = other }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := newKeptList(50)
			p := newPair(t, 300, 400, k.mk)
			p.idle(4)
			var recs []*rowKeep
			for _, st := range p.kept.rt.states {
				if st.krec != nil {
					recs = append(recs, st.krec)
				}
			}
			if len(recs) < 2 {
				t.Fatalf("%d rows held", len(recs))
			}
			before := p.kept.rt.rowsLaidKept
			for _, rec := range recs {
				tc.do(rec, recs[0])
			}
			// The first record is its own other, and stays held.
			p.frame(tc.name)
			if n := p.kept.rt.rowsLaidKept; n > 1 {
				t.Errorf("%s: %d rows kept their layout of %d, want 1 at most", tc.name, n, before)
			}
			p.idle(3)
		})
	}
}

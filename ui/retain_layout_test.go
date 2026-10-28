package ui

import (
	"fmt"
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

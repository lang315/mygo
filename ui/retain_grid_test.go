package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// gridApp is the data of one twin of a grid: the items' texts and versions, how
// the grid is laid out, and its state.
type gridApp struct {
	names  []string
	vers   []uint64
	minW   float32
	height float32
	sel    int
	chosen Selection[int]
	checks Selection[int]
	st     GridState
}

type gridScene struct {
	apps    []*gridApp
	version bool // RowVersion is set
	heads   bool // every 13th item heads a group
	reorder bool
	menu    func(k int, items []int) // the context menu of the items
	n       int
}

func (g *gridScene) set(f func(a *gridApp)) {
	for _, a := range g.apps {
		f(a)
	}
}

func (g *gridScene) mk() func() func(c *Context) {
	return func() func(c *Context) {
		a := &gridApp{names: make([]string, g.n), vers: make([]uint64, g.n), minW: 120, height: 60, sel: -1}
		for i := range a.names {
			a.names[i] = fmt.Sprint("Item ", i)
		}
		g.apps = append(g.apps, a)
		a.st.Selected, a.st.Selection, a.st.Checked = &a.sel, &a.chosen, &a.checks
		a.st.Label = func(i int) string { return a.names[i] }
		a.st.Key = func(i int) any {
			if g.heads && i%13 == 0 {
				return fmt.Sprint("header ", i) // not of the Selection's type
			}
			return i
		}
		if g.heads {
			a.st.Header = func(i int) bool { return i%13 == 0 }
		}
		if g.reorder {
			a.st.Reorder = func([]int, int) {}
		}
		if g.menu != nil {
			k := len(g.apps) - 1
			a.st.ItemMenu = func(m *Menu, items []int) { g.menu(k, items) }
		}
		if g.version {
			a.st.RowVersion = func(i int) uint64 { return a.vers[i] }
		}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				GridView(c, &a.st, g.n, a.minW, a.height, func(i int) {
					Column(c).Children(func() {
						if a.st.Header != nil && a.st.Header(i) {
							Textf(c, "Header %d", i)
							return
						}
						Text(c, a.names[i])
					})
				}).Grow(1)
			})
		}
	}
}

// A grid whose items are versioned keeps the rows at rest, and builds those whose
// items the app changed (and told, by their version), whatever else moves.
func TestRetainGridRowsFollowTheApp(t *testing.T) {
	g := &gridScene{version: true, n: 300}
	p := newPair(t, 500, 400, g.mk())
	p.idle(4)
	if k := keptGridRows(p, func() { p.frame("idle") }); k == 0 {
		t.Fatal("idle: no row of the grid kept")
	}
	g.set(func(a *gridApp) { a.names[3], a.vers[3] = "Renamed", a.vers[3]+1 })
	p.frame("renamed")
	p.frame("renamed +1")
	if !p.kept.HasText("Renamed") {
		t.Error("a versioned change is not shown by the kept rows")
	}
	g.set(func(a *gridApp) { a.names[3], a.vers[3] = "Item 3", a.vers[3]+1 })
	p.frame("back")
	// Changed without a word: the kept line shows what it was built with, as a
	// list's row does; it follows once the version moves.
	g.set(func(a *gridApp) { a.names[4] = "Silent" })
	p.both(func(tt *Tester) { tt.Frame() })
	if p.kept.HasText("Silent") {
		t.Error("a change nobody versioned reached a kept line")
	}
	if !p.built.HasText("Silent") {
		t.Error("the twin that builds every line does not show the change")
	}
	g.set(func(a *gridApp) { a.vers[4]++ })
	p.frame("versioned")
	if !p.kept.HasText("Silent") {
		t.Error("a line is not built again once its version moved")
	}
	p.compare("versioned")
}

// The choice, the checks, the keys, the focus and the pointer move the lines they
// touch and leave the others.
func TestRetainGridRowsFollowTheInput(t *testing.T) {
	for _, heads := range []bool{false, true} {
		t.Run(fmt.Sprint("headers=", heads), func(t *testing.T) {
			g := &gridScene{version: true, n: 300, heads: heads}
			p := newPair(t, 500, 400, g.mk())
			p.idle(4)
			if k := keptGridRows(p, func() { p.frame("idle") }); k == 0 {
				t.Fatal("idle: no row of the grid kept")
			}
			step := func(what string, f func(tt *Tester)) {
				t.Helper()
				p.both(f)
				p.frame(what)
				p.frame(what + " +1")
			}
			step("click", func(tt *Tester) {
				if err := tt.Click("Item 5"); err != nil {
					t.Fatal(err)
				}
			})
			for _, k := range []Key{KeyRight, KeyRight, KeyDown, KeyLeft, KeyUp, KeyEnd, KeyHome} {
				step(fmt.Sprint("key ", k), func(tt *Tester) { tt.Key(0, k) })
			}
			step("shift down", func(tt *Tester) { tt.Key(Shift, KeyDown) })
			step("cmd click", func(tt *Tester) { _ = tt.ClickWith(Cmd, "Item 9") })
			g.set(func(a *gridApp) { a.checks.Add(7); a.vers[7]++ })
			p.frame("checked by the app")
			step("hover", func(tt *Tester) { tt.Move(60, 60) })
			step("hover away", func(tt *Tester) { tt.Move(-20, -20) })
			step("blur", func(tt *Tester) { tt.SetFocused(false) })
			step("focus", func(tt *Tester) { tt.SetFocused(true) })
			if k := keptGridRows(p, func() { p.idle(2) }); k == 0 {
				t.Error("at rest after the input: no row kept")
			}
		})
	}
}

// The lines depend on the layout: the width of the window and the size of the
// items change which items a line holds (the line is not the same line), and the
// theme how they look. Kept lines never show the layout they were built in.
func TestRetainGridLinesFollowTheLayout(t *testing.T) {
	for _, heads := range []bool{false, true} {
		t.Run(fmt.Sprint("headers=", heads), func(t *testing.T) {
			g := &gridScene{version: true, n: 300, heads: heads}
			p := newPair(t, 500, 400, g.mk())
			p.idle(4)
			if k := keptGridRows(p, func() { p.frame("idle") }); k == 0 {
				t.Fatal("idle: no row of the grid kept")
			}
			p.both(func(tt *Tester) { tt.SetSize(330, 400) })
			p.frame("narrower")
			p.frame("narrower +1")
			p.both(func(tt *Tester) { tt.SetSize(700, 400) })
			p.frame("wider")
			p.frame("wider +1")
			g.set(func(a *gridApp) { a.minW = 200 })
			p.frame("wider items")
			p.frame("wider items +1")
			g.set(func(a *gridApp) { a.height = 90 })
			p.frame("taller items")
			p.frame("taller items +1")
			p.both(func(tt *Tester) { tt.SetDark(true) })
			p.frame("dark")
			p.frame("dark +1")
			p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, 300) })
			p.frame("scrolled")
			p.both(func(tt *Tester) { tt.SetSize(420, 400) })
			p.frame("narrower, scrolled")
			if k := keptGridRows(p, func() { p.idle(2) }); k == 0 {
				t.Error("at rest after the layout moved: no row kept")
			}
		})
	}
}

// A grid that reorders reads the cells it built to place a drop: its lines are
// built in every frame, whatever RowVersion says.
func TestRetainGridIgnoresRowVersionWhileItReorders(t *testing.T) {
	g := &gridScene{version: true, n: 300, reorder: true}
	p := newPair(t, 500, 400, g.mk())
	p.idle(4)
	if k := keptGridRows(p, func() { p.frame("idle") }); k != 0 {
		t.Errorf("%d lines kept in a grid that reorders", k)
	}
}

// keptGridRows returns how many rows the kept twin kept while f ran.
func keptGridRows(p *pair, f func()) int {
	before := p.kept.rt.rowsKept
	f()
	return p.kept.rt.rowsKept - before
}

// Every choice of the items of a line is a version of its own: the versions are
// not folded into one byte, where a line of nine items has more choices than
// versions.
func TestGridLineVersionsTellEveryChoiceOfALineApart(t *testing.T) {
	g := &gridScene{version: true, n: 60}
	p := newPair(t, 1540, 400, g.mk())
	p.idle(2)
	a := g.apps[0]
	cols := a.st.cols
	if cols < 12 {
		t.Fatalf("%d columns, want 12", cols)
	}
	seen := map[uint64]int{}
	for m := range 1 << cols {
		a.chosen.Clear()
		for i := range cols {
			if m&(1<<i) != 0 {
				a.chosen.Add(cols + i)
			}
		}
		v := a.st.lineVersion(1, g.n, a.minW, a.height)
		if o, ok := seen[v]; ok {
			t.Fatalf("choices %012b and %012b of a line give one version", o, m)
		}
		seen[v] = m
	}
}

// The item the keys are on is the grid's active descendant for assistive
// technology while its line is kept, as when it is built.
func TestRetainGridKeepsTheActiveDescendantOfAKeptLine(t *testing.T) {
	g := &gridScene{version: true, n: 300}
	p := newPair(t, 500, 400, g.mk())
	g.set(func(a *gridApp) { a.st.AccentChoice = true }) // as a ListView's
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(2)
	p.both(func(tt *Tester) { _ = tt.Click("Item 5") })
	p.frame("click")
	p.both(func(tt *Tester) { tt.Key(0, KeyRight) })
	for i := range 3 {
		p.frame(fmt.Sprint("right ", i))
		if a, b := p.kept.h.access.Focus, p.built.h.access.Focus; a != b || a == 0 {
			t.Fatalf("frame %d after the key: focus kept %d, built %d", i, a, b)
		}
	}
}

// Every cell says how many items the grid has: an item added builds the lines.
func TestRetainGridCellsSayHowManyItemsThereAreAfterAnAppend(t *testing.T) {
	g := &gridScene{version: true, n: 300}
	p := newPair(t, 500, 400, g.mk())
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(3)
	g.n = 301 // an item added at the end, as AddRow does
	g.set(func(a *gridApp) { a.names = append(a.names, "Item 300"); a.vers = append(a.vers, 0) })
	p.frame("added")
	p.frame("added +1")
	kept, built := p.kept.h.access.Nodes, p.built.h.access.Nodes
	if len(kept) != len(built) {
		t.Fatalf("the tree has %d nodes kept, %d built", len(kept), len(built))
	}
	for i, n := range kept {
		if n.SetSize != built[i].SetSize {
			t.Fatalf("node %d %q: set size kept %d, built %d", i, n.Label, n.SetSize, built[i].SetSize)
		}
	}
}

// What the grid is set to draw with moves the lines: the panel under the check
// boxes, the accent of the choice, and whether items can be checked at all.
func TestRetainGridLinesFollowWhatTheGridDrawsWith(t *testing.T) {
	g := &gridScene{version: true, n: 300}
	p := newPair(t, 500, 400, g.mk())
	p.idle(3)
	p.both(func(tt *Tester) { _ = tt.Click("Item 5") })
	g.set(func(a *gridApp) { a.checks.Add(5); a.vers[5]++ })
	p.idle(3)
	for _, c := range []struct {
		name string
		f    func(a *gridApp)
	}{
		{"accent choice", func(a *gridApp) { a.st.AccentChoice = true }},
		{"plain checks", func(a *gridApp) { a.st.PlainChecks = true }},
		{"no checks", func(a *gridApp) { a.st.Checked = nil }},
		{"checks again", func(a *gridApp) { a.st.Checked = &a.checks }},
	} {
		g.set(c.f)
		p.frame(c.name)
		p.frame(c.name + " +1")
	}
	if k := keptGridRows(p, func() { p.idle(2) }); k == 0 {
		t.Error("at rest after the changes: no row kept")
	}
}

// The menu of a kept line sees the grid as it is: an item added, and the items
// chosen.
func TestRetainGridMenuOfAKeptLineSeesTheItemsNow(t *testing.T) {
	var seen [2][]int
	g := &gridScene{version: true, n: 300, menu: func(k int, items []int) { seen[k] = append([]int(nil), items...) }}
	p := newPair(t, 500, 400, g.mk())
	p.idle(3)
	g.set(func(a *gridApp) { a.chosen.Add(5); a.chosen.Add(299) })
	p.frame("chosen")
	p.idle(2)
	g.n = 301
	g.set(func(a *gridApp) {
		a.names = append(a.names, "Item 300")
		a.vers = append(a.vers, 0)
		a.chosen.Add(300)
	})
	p.frame("added")
	p.idle(2)
	for _, tt := range []*Tester{p.kept, p.built} {
		if err := tt.RightClick("Item 5"); err != nil {
			t.Fatal(err)
		}
	}
	p.frame("menu")
	if len(seen[1]) != 3 {
		t.Fatalf("the menu of a built line gets items %v, want 5, 299 and 300", seen[1])
	}
	if fmt.Sprint(seen[0]) != fmt.Sprint(seen[1]) {
		t.Errorf("the menu of a kept line gets items %v, of a built one %v", seen[0], seen[1])
	}
}

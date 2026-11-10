package ui

import (
	"runtime"
	"testing"
)

// hBarTop returns where the horizontal bar's track of the list s starts
// (hTrack.Y), and whether the list shows one: its content is wider than it.
func hBarTop(tt *Tester, s *ListState) (float32, bool) {
	st := listStateOf(s)
	g := scrollBars(Rect{st.x, st.y, st.w, st.h}, st.barInset, float32(st.contentW), float32(st.contentH),
		float32(st.scrollX), float32(st.scrollY), st.flags, ScrollbarStyle{}.Resolve(tt.rt.c.theme), st.rtl)
	return g.hTrack.Y, g.horizontal
}

// hTable is a table of n rows whose columns, 900 DIPs, are wider than its
// 400 DIPs.
func hTable(n int, rtl bool) (*Tester, *ListState) {
	s := &ListState{Selection: &Selection[int]{}}
	cols := []TableColumn{{Title: "A", Width: 300, Fixed: true}, {Title: "B", Width: 300, Fixed: true}, {Title: "C", Width: 300, Fixed: true}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, s, cols, n, func(row, col int) { coreTextf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	if rtl {
		tt.SetDirection(RightToLeft)
	}
	tt.Frame()
	return tt, s
}

func wantLastAboveBar(t *testing.T, tt *Tester, s *ListState, n int, what string) {
	t.Helper()
	tt.Frame()
	tt.Frame()
	top, ok := hBarTop(tt, s)
	if !ok {
		t.Fatalf("%s: no horizontal bar", what)
	}
	r, ok := rowBox(tt, s, n-1)
	if !ok {
		t.Fatalf("%s: the last row is not built", what)
	}
	if r.Y+r.H > top {
		t.Errorf("%s: the last row ends at %v, under the bar's track at %v", what, r.Y+r.H, top)
	}
}

func TestTableScrolledToEndKeepsLastRowAboveHorizontalBar(t *testing.T) {
	const n = 100000
	tt, s := hTable(n, false)
	s.ScrollToEnd()
	wantLastAboveBar(t, tt, s, n, "ScrollToEnd")
}

func TestTableCtrlEndKeepsLastRowAboveHorizontalBar(t *testing.T) {
	const n = 100000
	tt, s := hTable(n, false)
	tt.Click("r0c0")
	tt.Key(0, KeyEnd)
	wantLastAboveBar(t, tt, s, n, "End")
	if runtime.GOOS == "darwin" {
		return // Ctrl+End moves the cursor only elsewhere (list.go keys)
	}
	tt, s = hTable(n, false)
	tt.Click("r0c0")
	tt.Key(Ctrl, KeyEnd)
	wantLastAboveBar(t, tt, s, n, "Ctrl+End")
}

func TestTableScrollIntoViewLastRowKeepsItAboveHorizontalBar(t *testing.T) {
	const n = 1000
	tt, s := hTable(n, false)
	s.ScrollIntoView(n - 1)
	wantLastAboveBar(t, tt, s, n, "ScrollIntoView")
}

func TestTableFollowEndKeepsLastRowAboveHorizontalBar(t *testing.T) {
	const n = 1000
	tt, s := hTable(n, false)
	s.FollowEnd = true
	s.ScrollToEnd()
	wantLastAboveBar(t, tt, s, n, "FollowEnd")
}

func TestTableRTLScrolledToEndKeepsLastRowAboveHorizontalBar(t *testing.T) {
	const n = 1000
	tt, s := hTable(n, true)
	s.ScrollToEnd()
	wantLastAboveBar(t, tt, s, n, "RTL ScrollToEnd")
}

// A List that scrolls sideways, with rows wider than it, is the same.
func TestListScrollXScrolledToEndKeepsLastRowAboveHorizontalBar(t *testing.T) {
	const n = 1000
	s := &ListState{}
	tt := coreNewTester(func(c *context) {
		l := coreList(c, s, n, func(i int) { coreBox(c).Width(900).Height(30) }).Grow(1)
		l.flags |= flagScrollX
		l.rowMinW = 900
	}, 400, 300)
	tt.Frame()
	s.ScrollToEnd()
	wantLastAboveBar(t, tt, s, n, "List ScrollX")
}

// A table that fits sideways reserves nothing: its last row ends at the
// bottom, as before.
func TestTableWithoutHorizontalScrollKeepsItsLastRowAtTheBottom(t *testing.T) {
	const n = 1000
	s := &ListState{}
	cols := []TableColumn{{Title: "A", Width: 100, Fixed: true}, {Title: "B", Width: 100, Fixed: true}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, s, cols, n, func(row, col int) { coreTextf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	s.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if r, ok := rowBox(tt, s, n-1); !ok || r.Y+r.H != 300 {
		t.Errorf("the last row is at %v (built %v); want it to end at 300", r, ok)
	}
}

// The room is kept whether the bar shows or not: the pointer coming over
// moves no row.
func TestTableHoverDoesNotMoveRowsOfAHorizontalBar(t *testing.T) {
	const n = 1000
	tt, s := hTable(n, false)
	s.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	before, _ := rowBox(tt, s, n-1)
	tt.Move(200, 150)
	tt.Frame()
	tt.Frame()
	if after, _ := rowBox(tt, s, n-1); after != before {
		t.Errorf("hover moved the last row from %v to %v", before, after)
	}
}

package ui

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A column the app pins (Frozen) at run time pins in the rows a Table keeps
// (RowVersion), as in its header: the rows read which columns are pinned and
// where (frozenN, starts, frozenStyle), so it is part of what they were built
// with (rowSignature). Compared with a twin that builds every row, as the
// frames go, a sideways scroll included.
func TestRetainTableFrozenToggle(t *testing.T) {
	cols := make([]TableColumn, 20)
	for j := range cols {
		cols[j] = TableColumn{Title: "H" + strconv.Itoa(j), Width: 100}
	}
	p := newPair(t, 500, 300, func() func(c *Context) {
		s := &ListState{RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Table(c, s, cols, 10, func(row, col int) {
				Textf(c, "r%dc%d", row, col)
			}).Grow(1)
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	same := func(what string) {
		t.Helper()
		p.compare(what)
		if a, b := cellLabels(p.kept), cellLabels(p.built); len(a) != len(b) {
			t.Fatalf("%s: %d cells kept, %d built", what, len(a), len(b))
		}
	}
	for range 3 {
		p.both(func(tt *Tester) { tt.Frame() })
	}
	same("built")
	cols[0].Frozen = true
	for f := range 2 {
		p.both(func(tt *Tester) { tt.Frame() })
		same("pinned, frame " + strconv.Itoa(f))
	}
	p.both(func(tt *Tester) { tt.Scroll(250, 150, 600, 0) })
	for f := range 2 {
		p.both(func(tt *Tester) { tt.Frame() })
		same("pinned and scrolled, frame " + strconv.Itoa(f))
	}
	cols[0].Frozen = false
	for f := range 2 {
		p.both(func(tt *Tester) { tt.Frame() })
		same("unpinned, frame " + strconv.Itoa(f))
	}
}

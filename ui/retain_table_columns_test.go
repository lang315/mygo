package ui

import (
	"bytes"
	"strconv"
	"testing"
)

// A Table whose rows are kept (RowVersion) builds, with VirtualColumns, the
// columns its scroll and width bring into view: which cells a row builds
// follows them (window), which keep does not compare, so such rows are not
// kept (rowNoKeep). Its frames show what the same table built in every frame
// shows.
func TestRetainVirtualColumnsFollowTheWindow(t *testing.T) {
	scene := func(keep bool) (*Tester, *ListState) {
		s := &ListState{VirtualColumns: true, RowVersion: func(int) uint64 { return 1 }}
		tt := coreNewTester(wideTable(s, 60, 100, 0, nil), 500, 300)
		tt.rt.noRetain = !keep
		for range 3 {
			tt.Frame()
		}
		return tt, s
	}
	a, _ := scene(true)
	b, _ := scene(false)
	same := func(what string) {
		t.Helper()
		if !bytes.Equal(a.Image().Pix, b.Image().Pix) {
			t.Fatalf("%s: the rows kept show another picture than the rows built", what)
		}
	}
	same("built")
	// Wider, the window takes in columns no row built.
	for i, w := range []int{900, 1400} {
		a.SetSize(w, 300)
		b.SetSize(w, 300)
		for f := range 3 {
			a.Frame()
			b.Frame()
			same("widened to " + strconv.Itoa(w) + ", frame " + strconv.Itoa(f))
		}
		if col := "r3c" + strconv.Itoa(8+5*i); !a.HasText(col) {
			t.Errorf("widened to %d: %s, in view, was not built", w, col)
		}
	}
	// Scrolled sideways, the window moves.
	for step := range 3 {
		a.Scroll(250, 150, 1000, 0)
		b.Scroll(250, 150, 1000, 0)
		for f := range 3 {
			a.Frame()
			b.Frame()
			same("scrolled, step " + strconv.Itoa(step) + ", frame " + strconv.Itoa(f))
		}
	}
}

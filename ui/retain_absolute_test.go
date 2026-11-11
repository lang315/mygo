package ui

import "testing"

// A row kept across frames (RowVersion) is not built again, so its elements
// keep what the build told their parents (absoluteChild): an absolute child
// in a kept row stays where its insets place it, idle and scrolled.
func TestRetainAbsoluteChildInKeptRows(t *testing.T) {
	var s ListState
	s.Key = func(i int) any { return i }
	s.RowVersion = func(i int) uint64 { return 1 }
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 100, func(i int) {
			coreRow(c).Height(30).Children(func() {
				coreText(c, "r")
				coreBox(c).Absolute().Left(30).Top(5).Size(10, 10).Background(RGB(1, 2, 3))
			})
		}).Grow(1)
	}, 200, 300)
	check := func(when string) {
		t.Helper()
		n := 0
		walkRow(tt.rt.c.root, func(e *node) {
			if e.flags&flagAbsolute != 0 && e.parent != nil && !e.parent.listRow && e.w == 10 {
				n++
				if e.x-e.parent.x != 30 || e.y-e.parent.y != 5 {
					t.Errorf("%s: absolute child at %v,%v in parent at %v,%v", when, e.x, e.y, e.parent.x, e.parent.y)
				}
			}
		})
		if n == 0 {
			t.Fatalf("%s: no absolute child found", when)
		}
	}
	for range 4 {
		tt.Frame()
	}
	check("warm")
	kept := tt.rt.rowsKept
	tt.Frame()
	check("idle")
	if tt.rt.rowsKept == kept {
		t.Fatal("no row kept")
	}
	tt.Scroll(100, 100, 0, 37)
	tt.Frame()
	check("scrolled")
	tt.Frame()
	check("idle after scroll")
}

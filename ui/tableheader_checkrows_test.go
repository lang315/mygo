package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A table with headers hands the header's check-all to CheckRows too: one call
// for all the lines, which leaves the headers out itself, not a call of Key for
// every line.
func TestTableCheckAllWithHeadersGoesThroughCheckRows(t *testing.T) {
	const n = 12
	isHeader := func(i int) bool { return i%4 == 0 }
	var checked Selection[int]
	var calls [][3]int
	s := ListState{Checked: &checked, Header: isHeader, Key: func(i int) any {
		if isHeader(i) {
			return struct{ header int }{i}
		}
		return i
	}}
	s.CheckRows = func(first, last int, on bool) bool {
		calls = append(calls, [3]int{first, last, map[bool]int{false: 0, true: 1}[on]})
		changed := false
		for i := first; i <= last; i++ {
			if !isHeader(i) {
				if on {
					checked.Add(i)
				} else {
					checked.Remove(i)
				}
				changed = true
			}
		}
		return changed
	}
	cols := []TableColumn{{Title: "Name"}}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, &s, cols, n, func(row, col int) { coreTextf(c, "Row %d", row).SingleLine() }).Grow(1)
		})
	}, 500, 600)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	box := accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all")
	tt.ClickAt(float32(box.Bounds.X+box.Bounds.W/2), float32(box.Bounds.Y+box.Bounds.H/2))
	if len(calls) != 1 || calls[0] != [3]int{0, n - 1, 1} {
		t.Fatalf("CheckRows was called %v, want once for lines 0 to %d, on", calls, n-1)
	}
	if checked.Len() != 9 || checked.Has(0) || checked.Has(4) || checked.Has(8) {
		t.Fatalf("check all checked %v: want the nine rows and no header", checked.keys)
	}
}

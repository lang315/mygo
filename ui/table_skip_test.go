package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// The header's check-all checks the rows that can be chosen only, and its
// state counts them: a row ListState.Skip names, as a Loading placeholder,
// is not checked by it nor counted.
func TestTableHeaderCheckAllLeavesSkippedRows(t *testing.T) {
	names := []string{"a.txt", "Loading…", "c.txt"}
	cols := []TableColumn{{Title: "Name"}}
	var checked Selection[string]
	s := ListState{Checked: &checked, Key: func(i int) any { return names[i] }, Skip: func(i int) bool { return names[i] == "Loading…" }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, &s, cols, len(names), func(row, col int) { coreText(c, names[row]).SingleLine() }).Grow(1)
		})
	}, 500, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	header := func() platform.AccessNode { return accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all") }
	click := func(n platform.AccessNode) {
		tt.ClickAt(float32(n.Bounds.X+n.Bounds.W/2), float32(n.Bounds.Y+n.Bounds.H/2))
	}
	click(header())
	if checked.Has("Loading…") || checked.Len() != 2 {
		t.Fatalf("check all checked %v: want a.txt and c.txt, not the skipped row", checked.keys)
	}
	// Both rows that can be checked are: the header shows all, not some.
	if h := header(); h.States&platform.AccessChecked == 0 || h.States&platform.AccessMixed != 0 {
		t.Fatalf("the header with every row that can be checked checked: %+v", h)
	}
	click(header())
	if checked.Len() != 0 {
		t.Fatalf("unchecking all left %v", checked.keys)
	}
}

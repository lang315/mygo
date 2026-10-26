package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A header row of a table is a row of its set like the others: assistive
// technology reads "Group 3, 4 of 8", and the rows after it by their place in
// the lines, headers counted, as the icon views of a list view do.
func TestTableHeaderRowsCountInThePositionAndSizeOfTheSet(t *testing.T) {
	label := func(i int) string {
		if i%4 == 0 {
			return fmt.Sprintf("Group %d", i)
		}
		return fmt.Sprintf("Line %d", i)
	}
	s := &ListState{Selection: &Selection[int]{}}
	s.Header = func(i int) bool { return i%4 == 0 }
	s.Label = label
	s.Key = func(i int) any {
		if s.Header(i) {
			return fmt.Sprint("header ", i)
		}
		return i
	}
	cols := []TableColumn{{Title: "Name"}}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, s, cols, 8, func(row, col int) { coreText(c, label(row)).SingleLine() }).Grow(1)
		})
	}, 448, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, w := range []struct {
		label string
		pos   int
	}{{"Group 0", 1}, {"Line 1", 2}, {"Line 3", 4}, {"Group 4", 5}, {"Line 7", 8}} {
		n := accessNode(t, tt.h.access, platform.RoleRow, w.label)
		if n.PosInSet != w.pos || n.SetSize != 8 {
			t.Errorf("%s is %d of %d, want %d of 8", w.label, n.PosInSet, n.SetSize, w.pos)
		}
		if n.States&platform.AccessChecked != 0 {
			t.Errorf("%s is chosen", w.label)
		}
	}
}

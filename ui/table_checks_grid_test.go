package ui

import (
	"sort"
	"testing"
)

// With GridLines, each cell of a row ends in a line, the check box cell
// first in a row as the others: a table with a check box column and an index
// column had none between them.
func TestTableGridLinesEndTheCheckBoxCell(t *testing.T) {
	var checked Selection[int]
	s := ListState{Checked: &checked, GridLines: true}
	cols := []TableColumn{{Title: "#", Width: 50}, {Title: "Name", Width: 80}}
	th := LightTheme()
	tt := coreNewTester(func(c *context) {
		c.SetTheme(th)
		coreTable(c, &s, cols, 3, func(row, col int) { coreBox(c).Size(4, 4) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	// The cells of the first row: the row below the header's.
	var cells []TestNode
	for _, n := range tt.Nodes() {
		if n.Role == RoleCell {
			cells = append(cells, n)
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Bounds.Y != cells[j].Bounds.Y {
			return cells[i].Bounds.Y < cells[j].Bounds.Y
		}
		return cells[i].Bounds.X < cells[j].Bounds.X
	})
	var row []TestNode
	for _, n := range cells {
		if n.Bounds.Y > cells[0].Bounds.Y {
			if len(row) > 0 && n.Bounds.Y != row[0].Bounds.Y {
				break
			}
			row = append(row, n)
		}
	}
	if len(row) != 3 {
		t.Fatalf("the first row has %d cells, want a check box, # and Name", len(row))
	}
	if head := cells[0]; head.Bounds.X != row[0].Bounds.X || head.Bounds.W != row[0].Bounds.W {
		t.Errorf("the line moved the row's check box cell to %v from the header's %v", row[0].Bounds, head.Bounds)
	}
	img := tt.Image()
	want := th.Border
	for i, n := range row {
		x, y := int(n.Bounds.X+n.Bounds.W)-1, int(n.Bounds.Y)+2 // the last column of the cell, near its top
		c := img.RGBAAt(x, y)
		if c.R != want.R || c.G != want.G || c.B != want.B {
			t.Errorf("cell %d (%v) does not end in the border color: pixel (%d, %d) is %v, want %v", i, n.Bounds, x, y, c, want)
		}
	}
}

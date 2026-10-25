package ui

import "testing"

// A header's key need not be of the type of the Selection's keys: a table
// does not ask the Selection about a header, in Copy, Cmd+A, a context menu,
// Space or anywhere else it lists the rows chosen.
func TestTableHeaderKeyOfAnotherTypeIsNotAskedOfTheSelection(t *testing.T) {
	type headerKey struct{ row int }
	chosen := &Selection[int]{}
	var copied [][]int
	s := &ListState{Selection: chosen}
	s.Header = func(i int) bool { return i%4 == 0 }
	s.Key = func(i int) any {
		if s.Header(i) {
			return headerKey{i}
		}
		return i
	}
	s.Copy = func(rows []int) string {
		copied = append(copied, append([]int(nil), rows...))
		return "copied"
	}
	cols := []TableColumn{{Title: "Name"}}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, s, cols, 12, func(row, col int) { coreTextf(c, "Row %d", row) }).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	tt.Click("Row 2")
	tt.Command("copy")
	if tt.Clipboard() != "copied" || len(copied) != 1 || len(copied[0]) != 1 || copied[0][0] != 2 {
		t.Fatalf("Copy of Row 2 gave %q for %v", tt.Clipboard(), copied)
	}
	tt.Key(Cmd, KeyA)
	tt.Command("copy")
	if len(copied) != 2 || len(copied[1]) != 9 {
		t.Fatalf("Copy of all gave %v, want the nine rows and no header", copied)
	}
}

package ui

import (
	"slices"
	"testing"
)

// currentTable shows a table of n rows and columns A, B, C that keeps a
// current column (CurrentCell) and chooses whole rows; cell r1c2 is row 1's
// in column C.
func currentTable(t *testing.T, n int) (*Tester, *ListState, *Selection[int]) {
	t.Helper()
	s := &ListState{CurrentCell: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, n, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	return tt, s, sel
}

// A click sets the current column and chooses the row; Left and Right move
// the column and leave the rows alone, Shift does not extend.
func TestCurrentCellClickAndKeys(t *testing.T) {
	tt, s, sel := currentTable(t, 6)
	tt.Click("r1c1")
	if s.Cells.Cursor != "B" || !slices.Equal(chosenRows(sel), []int{1}) {
		t.Fatalf("click r1c1: column %q rows %v, want B and [1]", s.Cells.Cursor, chosenRows(sel))
	}
	tt.ClickWith(Shift, "r3c1")
	if s.Cells.Cursor != "B" || !slices.Equal(chosenRows(sel), []int{1, 2, 3}) {
		t.Fatalf("Shift-click: column %q rows %v, want the column kept (B) and rows [1 2 3]", s.Cells.Cursor, chosenRows(sel))
	}
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	if s.Cells.Cursor != "C" || !slices.Equal(chosenRows(sel), []int{1, 2, 3}) {
		t.Fatalf("Right twice: column %q rows %v, want C and the rows kept", s.Cells.Cursor, chosenRows(sel))
	}
	tt.Key(Shift, KeyLeft)
	if s.Cells.Cursor != "B" || s.Cells.Anchor != s.Cells.Cursor {
		t.Fatalf("Shift+Left: %+v, want the cursor moved to B with no block", s.Cells)
	}
}

// The rows look as they do without CurrentCell: every cell of a chosen row
// is chosen, not only the current column's.
func TestCurrentCellRowsStayWhole(t *testing.T) {
	tt, s, _ := currentTable(t, 4)
	tt.Click("r2c0")
	for col := range 3 {
		if !s.CellSelected(2, col) {
			t.Errorf("CellSelected(2, %d) = false, want true: the whole row is chosen", col)
		}
	}
}

// CurrentCell and CellSelect are two ways to track columns: not both.
func TestCurrentCellWithCellSelectPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("CurrentCell with CellSelect did not panic")
		}
	}()
	s := &ListState{CurrentCell: true, CellSelect: true, Selection: &Selection[int]{}}
	tt := NewTester(func(c *Context) {
		Table(c, s, []TableColumn{{Title: "A"}}, 2, func(row, col int) { Textf(c, "x") })
	}, 200, 100)
	tt.Frame()
}

// Moving the current column rebuilds the row that has the cursor, not the rows
// the table keeps: the column is not part of what the kept rows were built with.
func TestCurrentCellMoveRebuildsOneRow(t *testing.T) {
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}, Key: func(i int) any { return i },
		RowVersion: func(i int) uint64 { return uint64(i) + 1 }}
	builds := make([]int, 30)
	cols := []TableColumn{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 30, func(row, col int) {
			if col == 0 {
				builds[row]++
			}
			Textf(c, "r%dc%d", row, col)
		}).Grow(1)
	}, 400, 600)
	for range 3 {
		tt.Frame()
	}
	tt.Click("r3c0")
	for range 3 {
		tt.Frame()
	}
	before := append([]int(nil), builds...)
	tt.Key(0, KeyRight)
	tt.Frame()
	rebuilt := 0
	for i := range builds {
		if builds[i] != before[i] {
			rebuilt++
		}
	}
	if s.Cells.Cursor != "B" {
		t.Fatalf("the column is %q after Right, want B", s.Cells.Cursor)
	}
	if rebuilt > 1 {
		t.Errorf("Right rebuilt %d rows, want only the row with the cursor", rebuilt)
	}
}

// Hiding the current column moves the current cell to a column that shows.
func TestCurrentCellHidingTheCurrentColumn(t *testing.T) {
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	cols := []TableColumn{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	shown := cols
	tt := NewTester(func(c *Context) {
		Table(c, s, shown, 4, func(row, col int) { Textf(c, "r%d%s", row, shown[col].Title) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r1B")
	if s.Cells.Cursor != "B" {
		t.Fatalf("the column is %q, want B", s.Cells.Cursor)
	}
	shown = []TableColumn{{Title: "A"}, {Title: "C"}}
	tt.Frame()
	tt.Frame()
	if got := s.Cells.Cursor; got != "A" && got != "C" {
		t.Errorf("after hiding B the column is %q, want one that shows", got)
	}
}

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

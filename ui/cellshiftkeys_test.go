package ui

import (
	"slices"
	"testing"
)

// Cmd+Shift with Left and Right (Ctrl+Shift on Linux and Windows) extends the
// columns of the block as Shift does: the cells are one block of columns, so
// there are no other columns to keep. With Up and Down it extends the rows
// and keeps the others, as in a list.
func TestCellSelectCmdShiftKeys(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.Click("r1c1")
	tt.Key(Cmd|Shift, KeyRight)
	wantBlock(t, s, sel, "Cmd+Shift+Right", []int{1}, ColumnRange{Cursor: "B", Anchor: "A"}, []string{"A", "B"})
	tt.Key(Cmd|Shift, KeyRight)
	wantBlock(t, s, sel, "Cmd+Shift+Right again", []int{1}, ColumnRange{Cursor: "C", Anchor: "A"}, []string{"A", "B", "C"})
	tt.Key(Cmd|Shift, KeyLeft)
	wantBlock(t, s, sel, "Cmd+Shift+Left", []int{1}, ColumnRange{Cursor: "B", Anchor: "A"}, []string{"A", "B"})

	// The rows: a row chosen before stays, going back takes out the rows
	// the extension chose.
	tt, s, sel = cellTable(t, 8, 3, 0)
	tt.Click("r4c1")
	tt.ClickWith(Cmd, "r1c1")
	for range 3 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	wantBlock(t, s, sel, "Cmd+Shift+Down three times", []int{1, 2, 3, 4}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
	for range 3 {
		tt.Key(Cmd|Shift, KeyUp)
	}
	wantBlock(t, s, sel, "Cmd+Shift+Up three times", []int{1, 4}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
}

// Right to left, Cmd+Shift with Left extends towards the column on the left
// of the screen, as Shift does.
func TestCellSelectCmdShiftKeysRightToLeft(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.SetDirection(RightToLeft)
	tt.Frame()
	tt.Click("r1c2")
	tt.Key(Cmd|Shift, KeyLeft)
	wantBlock(t, s, sel, "Cmd+Shift+Left from B right to left", []int{1}, ColumnRange{Cursor: "C", Anchor: "B"}, []string{"B", "C"})
}

// keyedCellTable shows a table that chooses cells, whose rows are keyed by
// keys, a slice the app may reorder, and chosen by key.
func keyedCellTable(keys *[]int) (*Tester, *ListState, *Selection[int]) {
	s := &ListState{CellSelect: true, Key: func(i int) any { return (*keys)[i] }}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, len(*keys), func(row, col int) {
			Textf(c, "k%dc%d", (*keys)[row], col)
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	return tt, s, sel
}

// What an extension chose is forgotten when the app moves the rows under it,
// in a table choosing cells as in a list.
func TestCellSelectCmdShiftKeysForgetWhenTheRowsMove(t *testing.T) {
	keys := seq(0, 7)
	tt, _, sel := keyedCellTable(&keys)
	tt.Click("k7c1")
	tt.ClickWith(Cmd, "k0c1")
	for range 3 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	if got := chosenRows(sel); !slices.Equal(got, []int{0, 1, 2, 3, 7}) {
		t.Fatalf("setup chose %v", got)
	}
	keys = moveKey(keys, 7, 2)
	tt.Frame()
	tt.Key(Cmd|Shift, KeyHome)
	if got := chosenRows(sel); !slices.Equal(got, []int{0, 1, 2, 3, 7}) {
		t.Fatalf("Cmd+Shift+Home after the rows moved chose %v, want [0 1 2 3 7]", got)
	}
}

// ... and when the app changes the choice.
func TestCellSelectCmdShiftKeysForgetWhenTheAppClearsTheChoice(t *testing.T) {
	tt, _, sel := cellTable(t, 8, 3, 0)
	tt.Click("r6c1")
	tt.ClickWith(Cmd, "r3c1")
	for range 4 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	if got := chosenRows(sel); !slices.Equal(got, []int{3, 4, 5, 6, 7}) {
		t.Fatalf("setup chose %v", got)
	}
	sel.Clear()
	tt.Frame()
	for range 3 {
		tt.Key(Cmd|Shift, KeyUp)
	}
	if got := chosenRows(sel); !slices.Equal(got, []int{3, 4}) {
		t.Fatalf("Cmd+Shift+Up three times after Clear chose %v, want [3 4]", got)
	}
}

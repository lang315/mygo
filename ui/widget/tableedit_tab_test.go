package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func editingAt(t *testing.T, g *DataGridViewWidget[acct, int], key int, col string) {
	t.Helper()
	if k, c, ok := g.EditingCell(); !ok || k != key || c != col {
		t.Fatalf("EditingCell = %v %q %v, want %d %q", k, c, ok, key, col)
	}
}

// Tab writes the draft and edits the next editable cell of the row, skipping
// the read-only Id; at the end of the row it goes to the first editable cell
// of the next one, and Shift+Tab goes back.
func TestTabMovesToTheNextEditableCell(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Key(0, ui.KeyTab)
	settleEdit(tt)
	editingAt(t, grid, 1, "Score")
	if r := rowOf(t, grid, 1); r.name != "beta" {
		t.Fatalf("Tab did not write the Name: %+v", r)
	}
	tt.Key(0, ui.KeyTab)
	settleEdit(tt)
	editingAt(t, grid, 2, "Name")
	tt.Key(ui.Shift, ui.KeyTab)
	settleEdit(tt)
	editingAt(t, grid, 1, "Score")
	if got := endsOf(ends); len(got) != 1 || got[0] != "1:Name" {
		t.Fatalf("ends %v, want only 1:Name (untouched drafts write nothing)", got)
	}
}

// The target is taken before the commit: the commit of the Name re-sorts the
// rows, and Shift+Tab must still go to the row that came before it.
func TestTabTargetIsTakenBeforeTheCommit(t *testing.T) {
	tt, grid, _ := editGrid(t)
	if err := tt.Click("Name"); err != nil { // ascending: ann, bob, cat
		t.Fatal(err)
	}
	grid.BeginEdit(3, "Name")
	settleEdit(tt)
	tt.Type("aaa") // cat becomes "aaa" and sorts first
	tt.Key(ui.Shift, ui.KeyTab)
	settleEdit(tt)
	editingAt(t, grid, 2, "Score")
	if r := rowOf(t, grid, 3); r.name != "aaa" {
		t.Fatalf("row 3 %+v, want name aaa", r)
	}
}

// A refused commit does not move.
func TestTabDoesNotMoveWhenRefused(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("abc")
	tt.Key(0, ui.KeyTab)
	settleEdit(tt)
	editingAt(t, grid, 1, "Score")
	if r := rowOf(t, grid, 1); r.score != 30 {
		t.Fatalf("row %+v was written", r)
	}
}

// Past the last editable cell there is nowhere to go: Tab writes and leaves,
// as in a table that moves the focus on.
func TestTabPastTheLastCellLeaves(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(3, "Score")
	settleEdit(tt)
	tt.Type("55")
	tt.Key(0, ui.KeyTab)
	settleEdit(tt)
	if grid.IsEditing() {
		t.Fatal("still editing past the last cell")
	}
	if r := rowOf(t, grid, 3); r.score != 55 {
		t.Fatalf("row %+v, want score 55", r)
	}
}

// Shift+Tab before the first editable cell leaves too.
func TestShiftTabBeforeTheFirstCellLeaves(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Key(ui.Shift, ui.KeyTab)
	settleEdit(tt)
	if grid.IsEditing() {
		t.Fatal("still editing before the first cell")
	}
}

// A cell CanEdit refuses is skipped.
func TestTabSkipsCellsThatCannotEdit(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.EditColumn("Score", CellEditor[acct]{
		CanEdit: func(a *acct) bool { return a.id != 2 },
		Set:     func(a *acct, s string) {},
	})
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Key(0, ui.KeyTab) // row 2's Name edits
	settleEdit(tt)
	editingAt(t, grid, 2, "Name")
	tt.Key(0, ui.KeyTab) // row 2's Score may not: row 3's Name
	settleEdit(tt)
	editingAt(t, grid, 3, "Name")
}

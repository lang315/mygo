package widget

import (
	"testing"
)

// Clicking another cell commits the edit: a written one closes.
func TestFocusLossCommits(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Click("cat") // another row's cell
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.name != "beta" || grid.IsEditing() {
		t.Fatalf("focus loss: row %+v editing %v; want it written and closed", r, grid.IsEditing())
	}
	if got := endsOf(ends); len(got) != 1 {
		t.Fatalf("ends %v, want one", got)
	}
}

// A refused commit on focus loss keeps the edit open with its draft and
// error, and does not take the focus back from where the user put it.
func TestFocusLossRefusedKeepsTheEditOpen(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("abc")
	tt.Click("cat")
	settleEdit(tt)
	if !grid.IsEditing() {
		t.Fatal("a refused commit on focus loss closed the edit")
	}
	if k, col, _ := grid.EditingCell(); k != 1 || col != "Score" {
		t.Errorf("the edit moved to %v %q", k, col)
	}
	if d := grid.editDraft(); d != "abc" {
		t.Errorf("the draft is %q, want abc", d)
	}
	if active, _ := tt.TextInputActive(); active {
		t.Error("the editor took the focus back after a refused commit")
	}
	// Frames go on without trying to commit again and again.
	for range 3 {
		tt.Frame()
	}
	if !grid.IsEditing() {
		t.Error("the edit closed by itself")
	}
}

func TestDeletingTheRowCancelsTheEdit(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("x")
	tt.Frame()
	grid.DeleteRows(1)
	settleEdit(tt)
	if grid.IsEditing() {
		t.Fatal("the edit of a deleted row survived")
	}
	if got := endsOf(ends); len(got) != 0 {
		t.Fatalf("ends %v for a deleted row", got)
	}
}

// A draft is dropped with the window: nothing is written, and IsEditing was
// the app's way to know there was one.
func TestClosingTheWindowDiscardsTheDraft(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Frame()
	if !grid.IsEditing() {
		t.Fatal("no edit before the window closes")
	}
	// The tester's window goes away with the test: nothing commits the draft.
	if r := rowOf(t, grid, 1); r.name != "ann" {
		t.Fatalf("row %+v was written before any commit", r)
	}
	if got := endsOf(ends); len(got) != 0 {
		t.Fatalf("ends %v", got)
	}
}

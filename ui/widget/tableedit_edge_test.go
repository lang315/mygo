package widget

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// editSource is the editing grid of editGrid over a given source and rows, with
// a searchable Name.
func editSourceGrid(src *RowSource[acct, int]) *DataGridViewWidget[acct, int] {
	return DataGridViewOf(src,
		Column[acct]{Title: "Id", Text: func(a *acct) string { return "#" + strconv.Itoa(a.id) }, Width: 60},
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true, Filterable: true},
	).Expand().EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) { a.name = s }})
}

// A search that hides the edited row cancels the edit; one that keeps it
// leaves the edit open.
func TestSearchHidingTheRowCancelsTheEdit(t *testing.T) {
	src := NewRowSource(acctKey).SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}, {3, "cat", 0}})
	grid := editSourceGrid(src)
	tt := tester(t, grid)
	grid.BeginEdit(2, "Name")
	settleEdit(tt)
	tt.Type("x")
	grid.Search("Name", "b") // bob stays
	settleEdit(tt)
	if !grid.IsEditing() || grid.editDraft() != "x" {
		t.Fatalf("a search that keeps the row ended the edit (editing %v, draft %q)", grid.IsEditing(), grid.editDraft())
	}
	grid.Search("Name", "zzz")
	settleEdit(tt)
	if grid.IsEditing() {
		t.Fatal("the edit of a hidden row stayed open")
	}
	if r, _ := src.Row(2); r.name != "bob" {
		t.Fatalf("row %+v written from a hidden editor", r)
	}
}

// Scrolling the edited row out of view and back keeps the draft: the editor
// is built again with it.
func TestScrollingKeepsTheEdit(t *testing.T) {
	src := NewRowSource(acctKey).SetRows(manyAccts(200))
	grid := editSourceGrid(src)
	tt := tester(t, grid)
	grid.BeginEdit(5, "Name")
	settleEdit(tt)
	tt.Type("x")
	tt.Frame()
	tt.Scroll(100, 100, 0, 4000)
	settleEdit(tt)
	tt.Scroll(100, 100, 0, -4000)
	settleEdit(tt)
	if !grid.IsEditing() || grid.editDraft() != "x" {
		t.Fatalf("after scrolling: editing %v draft %q, want the edit and \"x\"", grid.IsEditing(), grid.editDraft())
	}
	if r, _ := src.Row(5); r.name == "x" {
		t.Fatal("scrolling wrote the draft")
	}
}

// Sorting with the editor open leaves the edit on its row.
func TestSortingFollowsTheEditedRow(t *testing.T) {
	src := NewRowSource(acctKey).SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}, {3, "cat", 0}})
	grid := editSourceGrid(src)
	tt := tester(t, grid)
	if err := tt.Click("Name"); err != nil { // ascending
		t.Fatal(err)
	}
	if err := tt.Click("Name"); err != nil { // descending: cat, bob, ann
		t.Fatal(err)
	}
	grid.BeginEdit(3, "Name")
	settleEdit(tt)
	tt.Type("x")
	grid.ClearSort() // ann, bob, cat: the edited row moves to the end
	settleEdit(tt)
	if k, col, ok := grid.EditingCell(); !ok || k != 3 || col != "Name" || grid.editDraft() != "x" {
		t.Fatalf("EditingCell = %v %q %v draft %q after a sort", k, col, ok, grid.editDraft())
	}
}

// Two tables over one source: a commit in one shows in the other, which has
// no edit.
func TestTwoTablesOverOneSource(t *testing.T) {
	src := NewRowSource(acctKey).SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}})
	a, b := editSourceGrid(src), editSourceGrid(src)
	ta, tb := tester(t, a), tester(t, b)
	a.BeginEdit(1, "Name")
	settleEdit(ta)
	ta.Type("zed")
	ta.Key(0, ui.KeyEnter)
	settleEdit(ta)
	tb.Frame()
	if !tb.HasText("zed") {
		t.Fatal("the other table does not show the committed text")
	}
	if b.IsEditing() || a.IsEditing() {
		t.Fatalf("editing: a %v b %v, want none", a.IsEditing(), b.IsEditing())
	}
}

// Resizing a column keeps the draft.
func TestColumnResizeKeepsTheDraft(t *testing.T) {
	src := NewRowSource(acctKey).SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}})
	grid := editSourceGrid(src)
	tt := tester(t, grid)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("x")
	tt.Frame()
	grid.SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": 320}})
	settleEdit(tt)
	if !grid.IsEditing() || grid.editDraft() != "x" {
		t.Fatalf("after a resize: editing %v draft %q", grid.IsEditing(), grid.editDraft())
	}
}

// A refusal shows its message by the cell, and the next keystroke takes it away.
func TestRefusalMessageIsShownUnderTheCell(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("abc")
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if !tt.HasText("Score must be a number") {
		t.Fatal("the refusal is not shown")
	}
	tt.Type("1")
	settleEdit(tt)
	if tt.HasText("Score must be a number") {
		t.Fatal("the refusal stayed after a keystroke")
	}
}

package widget

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// settle lets the handlers the frames posted run, and the frames they ask for.
func settleEdit(tt *ui.Tester) {
	tt.Frame()
	<-posted()
	tt.Frame()
}

// endsOf returns the "<key>:<column>" events the grid delivered so far.
func endsOf(ends chan string) (got []string) {
	<-posted()
	for {
		select {
		case s := <-ends:
			got = append(got, s)
		default:
			return got
		}
	}
}

func rowOf(t *testing.T, g *DataGridViewWidget[acct, int], k int) acct {
	t.Helper()
	r, ok := g.Row(k)
	if !ok {
		t.Fatalf("no row %d", k)
	}
	return r
}

func TestEnterCommitsOnce(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.name != "beta" {
		t.Fatalf("row %+v, want name beta", r)
	}
	if got := endsOf(ends); grid.IsEditing() || len(got) != 1 || got[0] != "1:Name" {
		t.Fatalf("editing %v, ends %v; want the edit closed after one OnCellEndEdit 1:Name", grid.IsEditing(), got)
	}
}

func TestEscapeCancelsAndWritesNothing(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Key(0, ui.KeyEscape)
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.name != "ann" || grid.IsEditing() {
		t.Fatalf("row %+v editing %v, want ann kept and the edit closed", r, grid.IsEditing())
	}
	if got := endsOf(ends); len(got) != 0 {
		t.Fatalf("ends %v after Escape", got)
	}
}

func TestValidateRefusalKeepsTheEditOpen(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("abc")
	tt.Frame()
	err := grid.EndEdit()
	settleEdit(tt)
	if err == nil || !grid.IsEditing() {
		t.Fatalf("EndEdit = %v, editing %v; want a refusal and the edit open", err, grid.IsEditing())
	}
	if r := rowOf(t, grid, 1); r.score != 30 {
		t.Fatalf("a refused commit wrote %+v", r)
	}
	if got := endsOf(ends); len(got) != 0 {
		t.Fatalf("ends %v after a refusal", got)
	}
	found := false
	for _, a := range tt.Announcements() {
		found = found || a == "Score must be a number"
	}
	if !found {
		t.Errorf("the refusal was not announced: %q", tt.Announcements())
	}
}

// The user pressed Enter without typing while another goroutine changed the
// cell: nothing of theirs is written, and no event is delivered.
func TestUnchangedDraftDoesNotOverwrite(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	grid.UpdateRow(1, func(a *acct) { a.score = 120 }) // was 30
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.score != 120 {
		t.Fatalf("row %+v, want 120 kept", r)
	}
	if got := endsOf(ends); len(got) != 0 || grid.IsEditing() {
		t.Fatalf("ends %v editing %v; want no event and the edit closed", got, grid.IsEditing())
	}
}

// An edited draft is the user's: it wins over a concurrent change of the same
// cell, and the other fields changed meanwhile are kept.
func TestEditedDraftWinsOverConcurrentUpdate(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("7")
	tt.Frame()
	grid.UpdateRow(1, func(a *acct) { a.score = 120; a.name = "zeta" })
	if err := grid.EndEdit(); err != nil {
		t.Fatal(err)
	}
	if r := rowOf(t, grid, 1); r.score != 7 || r.name != "zeta" {
		t.Fatalf("row %+v, want score 7 (the user's) and name zeta (the other change kept)", r)
	}
}

// EndEdit with nothing being edited is nil and does nothing.
func TestEndEditWithoutAnEdit(t *testing.T) {
	_, grid, _ := editGrid(t)
	if err := grid.EndEdit(); err != nil {
		t.Fatalf("EndEdit with no edit = %v", err)
	}
}

// Set panics: the lock is released, the edit is cancelled, OnPanic runs once,
// and the row is as the app's Set left it (this one builds on a copy).
func TestSetPanicReleasesTheLockAndCancelsTheEdit(t *testing.T) {
	panics := make(chan any, 4)
	OnPanic(func(v any, stack []byte) { panics <- v })
	defer OnPanic(nil)
	grid := DataGridView(acctKey,
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200},
	).SetRows([]acct{{1, "ann", 30}}).Expand().
		EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) {
			n := *a
			n.name = s
			panic("boom " + strconv.Itoa(n.id))
		}})
	tt := tester(t, grid)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("x")
	tt.Frame()
	if err := grid.EndEdit(); err != nil {
		t.Fatalf("EndEdit = %v; a panic is reported to OnPanic, not returned", err)
	}
	settleEdit(tt)
	if grid.IsEditing() {
		t.Error("the edit survived a panic in Set")
	}
	grid.Len() // takes the lock: it hangs if the panic left it held
	select {
	case v := <-panics:
		if v != "boom 1" {
			t.Errorf("OnPanic got %v, want boom 1", v)
		}
	default:
		t.Error("OnPanic was not called")
	}
	if r := rowOf(t, grid, 1); r.name != "ann" {
		t.Errorf("row %+v was written", r)
	}
}

// Set writes a field of the row and then panics: the row is as it was, its
// generation did not move, and a sorted view is not left stale.
func TestSetPanicAfterWritingAFieldLeavesTheRowAlone(t *testing.T) {
	OnPanic(func(v any, stack []byte) {})
	defer OnPanic(nil)
	grid := DataGridView(acctKey,
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true},
	).SetRows([]acct{{1, "ann", 30}, {2, "bob", 10}}).Expand().
		EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) {
			a.name = "zz" // written before the panic
			panic("boom")
		}})
	tt := tester(t, grid)
	before := grid.src.Version(1)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("x")
	tt.Frame()
	_ = grid.EndEdit()
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.name != "ann" {
		t.Fatalf("row %+v: Set's partial write stayed", r)
	}
	if v := grid.src.Version(1); v != before {
		t.Errorf("Version moved from %d to %d for a write that did not happen", before, v)
	}
	inOrder(t, tt, "ann", "bob")
}

// EndEdit and CancelEdit on a grid that never had an editable column do
// nothing: a Save handler shared by every form calls them.
func TestEditCallsOnAGridWithoutEditColumn(t *testing.T) {
	grid := testTable(acct{1, "ann", 1})
	if err := grid.EndEdit(); err != nil {
		t.Fatalf("EndEdit = %v", err)
	}
	grid.CancelEdit()
	if grid.IsEditing() || grid.ReadOnly() {
		t.Error("a grid with no editable column is editing or read-only")
	}
	if _, _, ok := grid.EditingCell(); ok {
		t.Error("EditingCell reports a cell")
	}
	if _, _, ok := grid.CurrentCell(); ok {
		t.Error("CurrentCell reports a cell before a frame")
	}
	if grid.SetCurrentCell(1, "Name") {
		t.Error("SetCurrentCell reported success on a grid that does not edit")
	}
	if grid.BeginEdit(1, "Name") {
		t.Error("BeginEdit reported success on a grid that does not edit")
	}
}

// Beginning another edit commits the open one; when that is refused the new
// edit does not open and the refused one keeps its draft.
func TestBeginningAnotherEditCommitsTheOpenOne(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Frame()
	if !grid.BeginEdit(2, "Name") {
		t.Fatal("BeginEdit refused while another edit was open and valid")
	}
	if r := rowOf(t, grid, 1); r.name != "beta" {
		t.Fatalf("row %+v, want the first edit written", r)
	}
	if k, _, _ := grid.EditingCell(); k != 2 {
		t.Fatalf("editing row %d, want 2", k)
	}
	if got := endsOf(ends); len(got) != 1 {
		t.Fatalf("ends %v, want one", got)
	}

	grid.CancelEdit()
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("abc")
	tt.Frame()
	if grid.BeginEdit(3, "Name") {
		t.Fatal("BeginEdit opened a new edit over a refused one")
	}
	if k, col, _ := grid.EditingCell(); k != 1 || col != "Score" || grid.editDraft() != "abc" {
		t.Fatalf("the refused edit is %v %q with draft %q, want 1 Score abc", k, col, grid.editDraft())
	}
}

// Hiding the column being edited cancels the edit: the editor is gone, so a
// later EndEdit must not write a draft nobody sees.
func TestHidingTheEditedColumnCancelsTheEdit(t *testing.T) {
	tt, grid, ends := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	tt.Type("beta")
	tt.Frame()
	grid.SetColumnVisible("Name", false)
	settleEdit(tt)
	if grid.IsEditing() {
		t.Fatal("the edit of a hidden column stayed open")
	}
	_ = grid.EndEdit()
	if r := rowOf(t, grid, 1); r.name != "ann" || len(endsOf(ends)) != 0 {
		t.Fatalf("row %+v written from a hidden editor", r)
	}
}

// A cancel or a commit that the editor posted for an edit does not end an edit
// begun after it.
func TestStaleCancelDoesNotEndANewEdit(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	stale := grid.ed().edit
	grid.CancelEdit()
	grid.BeginEdit(2, "Name")
	settleEdit(tt)
	grid.cancelEdit(stale)
	if err := grid.endEdit(stale); err != nil {
		t.Fatal(err)
	}
	if k, _, ok := grid.EditingCell(); !ok || k != 2 {
		t.Fatalf("the new edit was ended by a stale call: %v %v", k, ok)
	}
}

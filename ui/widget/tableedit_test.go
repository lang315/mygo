package widget

import (
	"errors"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// editGrid shows a grid of accounts whose Name and Score cells edit (Score
// wants a number) and whose Id cells do not; ends gets "<key>:<column>" for
// each edit written, on the event goroutine.
func editGrid(t *testing.T) (*ui.Tester, *DataGridViewWidget[acct, int], chan string) {
	t.Helper()
	ends := make(chan string, 16)
	grid := DataGridView(acctKey,
		Column[acct]{Title: "Id", Text: func(a *acct) string { return "#" + strconv.Itoa(a.id) }, Width: 60},
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) }, Width: 100},
	).SetRows([]acct{{1, "ann", 30}, {2, "bob", 10}, {3, "cat", 20}}).Expand().
		EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) { a.name = s }}).
		EditColumn("Score", CellEditor[acct]{
			Validate: func(a *acct, s string) error {
				if _, err := strconv.Atoi(s); err != nil {
					return errors.New("Score must be a number")
				}
				return nil
			},
			Set: func(a *acct, s string) { a.score, _ = strconv.Atoi(s) }})
	grid.OnCellEndEdit(func(k int, column string) { ends <- strconv.Itoa(k) + ":" + column })
	return tester(t, grid), grid, ends
}

func TestF2EditsTheCurrentCell(t *testing.T) {
	tt, grid, _ := editGrid(t)
	tt.Click("ann") // row 1, column Name
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if !grid.IsEditing() {
		t.Fatal("F2 did not start an edit")
	}
	if k, col, ok := grid.EditingCell(); !ok || k != 1 || col != "Name" {
		t.Fatalf("EditingCell = %v %q %v, want 1 Name", k, col, ok)
	}
}

func TestReadOnlyCellsDoNotEdit(t *testing.T) {
	tt, grid, _ := editGrid(t)
	r := find(t, tt, "ann")
	tt.ClickAt(20, r.Y+r.H/2) // the Id cell of row 1: no Set
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if grid.IsEditing() {
		t.Fatal("F2 edited a read-only column")
	}
	grid.SetReadOnly(true)
	tt.Click("ann")
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if grid.IsEditing() {
		t.Fatal("F2 edited a read-only grid")
	}
	if !grid.ReadOnly() {
		t.Error("ReadOnly() = false after SetReadOnly(true)")
	}
}

func TestEnterEditsWithEditOnEnterOrF2OnlyAndCanEditGates(t *testing.T) {
	tt, grid, _ := editGrid(t)
	tt.Click("ann")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !grid.IsEditing() {
		t.Fatal("Enter did not start an edit")
	}
	grid.CancelEdit()
	tt.Frame()
	grid.SetEditMode(EditOnF2)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if grid.IsEditing() {
		t.Fatal("Enter edited with EditOnF2")
	}
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if !grid.IsEditing() {
		t.Fatal("F2 did not edit with EditOnF2")
	}
}

func TestDoubleClickEdits(t *testing.T) {
	tt, grid, _ := editGrid(t)
	tt.Click("bob")
	tt.Click("bob")
	tt.Frame()
	if k, col, ok := grid.EditingCell(); !ok || k != 2 || col != "Name" {
		t.Fatalf("double click: EditingCell = %v %q %v, want 2 Name", k, col, ok)
	}
}

func TestEditingStartsWithTheCellTextAllSelected(t *testing.T) {
	tt, grid, _ := editGrid(t)
	if !grid.BeginEdit(1, "Name") {
		t.Fatal("BeginEdit refused an editable cell")
	}
	tt.Frame()
	tt.Frame()
	tt.Type("X") // all selected: typing replaces the text
	tt.Frame()
	if d := grid.editDraft(); d != "X" {
		t.Fatalf("draft %q, want X", d)
	}
	if grid.BeginEdit(1, "Id") {
		t.Error("BeginEdit accepted a read-only column")
	}
	if grid.BeginEdit(99, "Name") {
		t.Error("BeginEdit accepted a row that is not there")
	}
}

func TestCellEditorShowsTheCellText(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(2, "Name")
	tt.Frame()
	tt.Frame()
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("no text input is active: the editor did not show or take the focus")
	}
}

// A ListView shows a table's rows but is a list of items, not a grid: a
// column with a Set does not make it edit.
func TestListViewDoesNotEditEditableColumns(t *testing.T) {
	lv := ListView(func(v *vitem) string { return v.id },
		Column[vitem]{Title: "Name", Text: func(v *vitem) string { return v.name }, Width: 200},
		Column[vitem]{Title: "Kind", Text: func(v *vitem) string { return v.kind }, Width: 100}).
		SetRows(fruits)
	lv.t.EditColumn("Name", CellEditor[vitem]{Set: func(v *vitem, s string) { v.name = s }}) // the table under the view
	tt := tester(t, lv)
	tt.Click("Apple")
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if lv.t.IsEditing() {
		t.Fatal("F2 edited a cell of a ListView")
	}
	if lv.t.list.CurrentCell || lv.t.list.EditCell != nil {
		t.Error("a ListView's table was wired for editing")
	}
	if lv.t.BeginEdit("a", "Name") {
		t.Error("BeginEdit accepted a cell of a ListView")
	}
}

// EditColumn is all a grid needs: no SetEditMode, the default mode edits with
// F2; a grid with no EditColumn leaves F2 alone, and a column not given to it
// stays read-only.
func TestEditColumnNeedsNoSetEditMode(t *testing.T) {
	plain := DataGridView(acctKey,
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200},
	).SetRows([]acct{{1, "ann", 30}}).Expand()
	tt := tester(t, plain)
	tt.Click("ann")
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if plain.IsEditing() {
		t.Fatal("F2 edited a grid with no editable column")
	}
	grid := DataGridView(acctKey,
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200},
	).SetRows([]acct{{1, "ann", 30}}).Expand().
		EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) { a.name = s }})
	tt = tester(t, grid)
	tt.Click("ann")
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	if !grid.IsEditing() {
		t.Fatal("F2 did not edit a grid with an editable column, without SetEditMode")
	}
}

// EditColumn without a Set, or for a title no column has, is a programming
// error: it panics, with the lock released.
func TestEditColumnPanicsOnMistakes(t *testing.T) {
	grid := DataGridView(acctKey, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }})
	for name, f := range map[string]func(){
		"no Set":   func() { grid.EditColumn("Name", CellEditor[acct]{}) },
		"no title": func() { grid.EditColumn("Nope", CellEditor[acct]{Set: func(*acct, string) {}}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: EditColumn did not panic", name)
				}
			}()
			f()
		}()
	}
	grid.Len() // takes the lock: it hangs if the panic left it held
}

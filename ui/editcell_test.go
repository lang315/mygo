package ui

import "testing"

type editCalls struct {
	asked, edited, activated int
	row, col                 int
	how                      EditHow
}

func editTable(t *testing.T, edits func(row, col int) bool, calls *editCalls) (*Tester, *ListState) {
	t.Helper()
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	s.EditCell = func(row, col int, how EditHow) bool { calls.asked++; return edits(row, col) }
	s.OnEditCell = func(row, col int, x, y float32, how EditHow) {
		calls.edited++
		calls.row, calls.col, calls.how = row, col, how
	}
	cols := []TableColumn{{Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		if Table(c, s, cols, 4, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1).Submitted() {
			calls.activated++
		}
	}, 400, 300)
	tt.Frame()
	return tt, s
}

// Enter on an editable cell edits it and does not activate the row; on a
// read-only cell it activates, as before.
func TestEditCellEnter(t *testing.T) {
	var calls editCalls
	tt, _ := editTable(t, func(row, col int) bool { return col == 1 }, &calls)
	tt.Click("r1c1") // column B
	tt.Key(0, KeyEnter)
	tt.Frame()
	if calls.edited != 1 || calls.activated != 0 || calls.row != 1 || calls.col != 1 || calls.how != EditKey {
		t.Fatalf("Enter on an editable cell: %+v, want one edit of (1,1) by key and no activation", calls)
	}
	tt.Click("r1c0") // column A, read-only
	tt.Key(0, KeyEnter)
	tt.Frame()
	if calls.edited != 1 || calls.activated != 1 {
		t.Fatalf("Enter on a read-only cell: %+v, want it to activate", calls)
	}
}

func TestEditCellF2(t *testing.T) {
	var calls editCalls
	tt, _ := editTable(t, func(row, col int) bool { return true }, &calls)
	tt.Click("r2c0")
	tt.Key(0, KeyF2)
	tt.Frame()
	if calls.edited != 1 || calls.row != 2 || calls.col != 0 || calls.how != EditF2 {
		t.Fatalf("F2: %+v, want one edit of (2,0) by F2", calls)
	}
}

// A double click on an editable cell edits it and does not activate; with
// no EditCell the double click activates as before.
func TestEditCellDoubleClick(t *testing.T) {
	var calls editCalls
	tt, _ := editTable(t, func(row, col int) bool { return true }, &calls)
	tt.Click("r1c1")
	tt.Click("r1c1")
	tt.Frame()
	if calls.edited != 1 || calls.activated != 0 || calls.how != EditClick || calls.col != 1 {
		t.Fatalf("double click: %+v, want one edit of column 1 by click and no activation", calls)
	}
}

func TestNoEditCellKeepsActivation(t *testing.T) {
	activated := 0
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	tt := NewTester(func(c *Context) {
		if Table(c, s, []TableColumn{{Title: "A"}}, 3, func(row, col int) { Textf(c, "r%d", row) }).Grow(1).Submitted() {
			activated++
		}
	}, 300, 200)
	tt.Frame()
	tt.Click("r1")
	tt.Key(0, KeyEnter)
	tt.Frame()
	if activated != 1 {
		t.Fatalf("Enter without EditCell activated %d times, want 1", activated)
	}
}

// A table that does not opt in to editing leaves F2 and Enter to the window's
// shortcuts: the list is in the focus chain and must not take the key.
func TestTableWithoutEditCellLeavesF2ToTheWindow(t *testing.T) {
	s := &ListState{Selection: &Selection[int]{}}
	f2 := 0
	tt := NewTester(func(c *Context) {
		if c.Shortcut(0, KeyF2) {
			f2++
		}
		Table(c, s, []TableColumn{{Title: "A"}}, 3, func(row, col int) { Textf(c, "r%d", row) }).Grow(1)
	}, 300, 200)
	tt.Frame()
	tt.Click("r1")
	tt.Key(0, KeyF2)
	tt.Frame()
	if f2 != 1 {
		t.Fatalf("the window's F2 shortcut ran %d times, want 1: the table took the key", f2)
	}
}

// The same with CurrentCell but no EditCell: nothing edits, F2 is the app's.
func TestCurrentCellWithoutEditCellLeavesF2ToTheWindow(t *testing.T) {
	s := &ListState{Selection: &Selection[int]{}, CurrentCell: true}
	f2 := 0
	tt := NewTester(func(c *Context) {
		if c.Shortcut(0, KeyF2) {
			f2++
		}
		Table(c, s, []TableColumn{{Title: "A"}}, 3, func(row, col int) { Textf(c, "r%d", row) }).Grow(1)
	}, 300, 200)
	tt.Frame()
	tt.Click("r1")
	tt.Key(0, KeyF2)
	tt.Frame()
	if f2 != 1 {
		t.Fatalf("the window's F2 shortcut ran %d times, want 1", f2)
	}
}

// A double click on a RowHeader cell is not on an editable cell: the row
// activates, and no edit opens in the column the cursor was left on.
func TestEditCellDoubleClickOnRowHeaderActivates(t *testing.T) {
	var calls editCalls
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	s.EditCell = func(row, col int, how EditHow) bool { return true }
	s.OnEditCell = func(row, col int, x, y float32, how EditHow) { calls.edited++ }
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A", Width: 120}}
	tt := NewTester(func(c *Context) {
		if Table(c, s, cols, 4, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1).Submitted() {
			calls.activated++
		}
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c1")
	r, _ := tt.Find("r1c1")
	tt.ClickAt(10, r.Y+r.H/2)
	tt.ClickAt(10, r.Y+r.H/2)
	tt.Frame()
	if calls.edited != 0 || calls.activated != 1 {
		t.Fatalf("double click on the row header: %+v, want an activation and no edit", calls)
	}
}

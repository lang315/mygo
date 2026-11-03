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

package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func accessOn(tt *Tester) *platform.AccessTree {
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	return tt.h.access
}

func accessByID(tree *platform.AccessTree, id uint64) (platform.AccessNode, bool) {
	for _, n := range tree.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return platform.AccessNode{}, false
}

// focusedCell returns the node with the focus, which must be a cell.
func focusedCell(t *testing.T, tree *platform.AccessTree) platform.AccessNode {
	t.Helper()
	n, ok := accessByID(tree, tree.Focus)
	if !ok || n.Role != platform.RoleCell {
		t.Fatalf("focus is %+v (found %v), want a cell", n, ok)
	}
	return n
}

// With a current cell the focus of assistive technology is the cell, named by
// its text and describing its column, and the row stays chosen; Left and Right
// move it, which changes the focus so that readers announce the new cell.
func TestCurrentCellAccessFocus(t *testing.T) {
	tt, s, _ := currentTable(t, 6)
	tt.Click("r1c1")
	tree := accessOn(tt)
	cell := focusedCell(t, tree)
	if cell.Label != "r1c1" || cell.Description != "B" {
		t.Fatalf("focused cell %q (%q), want r1c1 in column B", cell.Label, cell.Description)
	}
	row := tree.Nodes[cell.Parent]
	if row.Role != platform.RoleRow || row.States&platform.AccessChecked == 0 {
		t.Errorf("the cell's parent %+v, want the row, still chosen", row)
	}
	if cell.States&platform.AccessFocusable == 0 || cell.Actions&platform.ActionFocus == 0 {
		t.Errorf("the cell is not focusable: %+v", cell)
	}
	// Its text is the cell's name, not a node read again.
	for _, n := range tree.Nodes {
		if n.Parent >= 0 && tree.Nodes[n.Parent].ID == cell.ID {
			t.Errorf("the cell has a child %+v besides its name", n)
		}
	}
	tt.Key(0, KeyRight)
	tree = accessOn(tt)
	moved := focusedCell(t, tree)
	if moved.ID == cell.ID || moved.Label != "r1c2" || moved.Description != "C" || s.Cells.Cursor != "C" {
		t.Fatalf("after Right: %+v, want r1c2 in column C", moved)
	}
	tt.Key(0, KeyDown)
	if got := focusedCell(t, accessOn(tt)); got.Label != "r2c2" {
		t.Fatalf("after Down: %q, want r2c2", got.Label)
	}
}

// CellSelect has a focused cell too.
func TestCellSelectAccessFocus(t *testing.T) {
	tt, _, _ := cellTable(t, 8, 3, 0)
	tt.Click("r2c1")
	if got := focusedCell(t, accessOn(tt)); got.Label != "r2c1" || got.Description != "A" {
		t.Fatalf("focused cell %q (%q), want r2c1 in column A", got.Label, got.Description)
	}
}

// A table that tracks no cell keeps the row as its focus and its cells as
// they were: unnamed, with their texts as children.
func TestPlainTableAccessUnchanged(t *testing.T) {
	s := &ListState{Selection: &Selection[int]{}}
	cols := []TableColumn{{Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 4, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c1")
	tree := accessOn(tt)
	if n, _ := accessByID(tree, tree.Focus); n.Role != platform.RoleRow {
		t.Fatalf("focus is %+v, want the row", n)
	}
	for i, n := range tree.Nodes {
		if n.Role != platform.RoleCell {
			continue
		}
		if n.Label != "" || n.Description != "" || n.States&platform.AccessFocusable != 0 || n.Actions&^platform.ActionScrollIntoView != 0 {
			t.Errorf("cell %+v of a plain table changed", n)
		}
		if i+1 >= len(tree.Nodes) || tree.Nodes[i+1].Parent != i || tree.Nodes[i+1].Role != platform.RoleText {
			t.Errorf("cell %d lost its text child", i)
		}
	}
}

// Assistive technology focusing a cell does what a click on it does: with a
// current cell it moves the column and chooses the row.
func TestCurrentCellAccessFocusAction(t *testing.T) {
	tt, s, sel := currentTable(t, 6)
	tree := accessOn(tt)
	var target platform.AccessNode
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleCell && n.Label == "r3c2" {
			target = n
		}
	}
	if target.ID == 0 {
		t.Fatal("no cell r3c2")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: target.ID, Action: platform.AccessFocus})
	tt.Frame()
	if s.Cells.Cursor != "C" || !slices.Equal(chosenRows(sel), []int{3}) {
		t.Fatalf("focus on r3c2: column %q rows %v, want C and [3]", s.Cells.Cursor, chosenRows(sel))
	}
	if got := focusedCell(t, accessOn(tt)); got.Label != "r3c2" {
		t.Fatalf("focus is %q, want r3c2", got.Label)
	}
}

// With CellSelect it chooses the cell, by the rules of a click on it: the
// block is that cell alone, not its row's columns.
func TestCellSelectAccessFocusAction(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.Click("r1c1")
	tt.ClickWith(Shift, "r2c2") // a block of four
	tree := accessOn(tt)
	var target platform.AccessNode
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleCell && n.Label == "r5c3" {
			target = n
		}
	}
	if target.ID == 0 {
		t.Fatal("no cell r5c3")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: target.ID, Action: platform.AccessFocus})
	tt.Frame()
	wantBlock(t, s, sel, "focus on r5c3", []int{5}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
}

// A cell being edited holds the focus with its editor.
func TestCurrentCellAccessFocusInEditor(t *testing.T) {
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	editing := false
	draft := "r1c1"
	cols := []TableColumn{{Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 4, func(row, col int) {
			if editing && row == 1 && col == 1 {
				CellInput(c, &draft, CellStart{Fresh: true, SelectAll: true})
				return
			}
			Textf(c, "r%dc%d", row, col)
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c1")
	if got := focusedCell(t, accessOn(tt)); got.Label != "r1c1" {
		t.Fatalf("before editing: focus %q", got.Label)
	}
	editing = true
	tt.Frame()
	tt.Frame()
	tree := accessOn(tt)
	n, ok := accessByID(tree, tree.Focus)
	if !ok || n.Role != platform.RoleTextField {
		t.Fatalf("while editing: focus %+v (found %v), want the editor", n, ok)
	}
}

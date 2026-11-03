package ui

import (
	"fmt"
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

// The twin that keeps the rows of a table with a current cell tells assistive
// technology what the twin that builds them all does: the same cells, names,
// descriptions and focus, as the column and the row move.
func TestRetainCurrentCellAccess(t *testing.T) {
	mk := func() func(c *Context) {
		s := &ListState{CurrentCell: true, Selection: &Selection[int]{}, Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		cols := []TableColumn{{Title: "A"}, {Title: "B"}, {Title: "C"}}
		return func(c *Context) {
			Table(c, s, cols, 40, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
		}
	}
	p := newPair(t, 400, 300, mk)
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	same := func(what string) {
		t.Helper()
		view := func(tt *Tester) (out []string) {
			tree := tt.h.access
			out = append(out, fmt.Sprint("focus ", slices.IndexFunc(tree.Nodes, func(n platform.AccessNode) bool { return n.ID == tree.Focus })))
			for _, n := range tree.Nodes {
				out = append(out, fmt.Sprint(n.Parent, n.Role, n.Label, "|", n.Description, n.States, n.Actions))
			}
			return out
		}
		if a, b := view(p.kept), view(p.built); !slices.Equal(a, b) {
			t.Fatalf("after %s: the trees differ\nkept  %q\nbuilt %q", what, a, b)
		}
	}
	for _, step := range []struct {
		what string
		do   func(tt *Tester)
	}{
		{"click", func(tt *Tester) { tt.Click("r3c1") }},
		{"Right", func(tt *Tester) { tt.Key(0, KeyRight) }},
		{"Down", func(tt *Tester) { tt.Key(0, KeyDown) }},
		{"Left", func(tt *Tester) { tt.Key(0, KeyLeft) }},
	} {
		p.both(step.do)
		p.frame(step.what)
		p.frame(step.what + " +1")
		same(step.what)
	}
	if focusedCell(t, p.kept.h.access).Label != "r4c1" {
		t.Fatalf("focus %q, want r4c1", focusedCell(t, p.kept.h.access).Label)
	}
}

// An OutlineTable keeps the focus on its row, which holds the level and
// whether it is open; Right opens it.
func TestOutlineTableAccessFocusStaysOnRow(t *testing.T) {
	var s OutlineState[string]
	s.List.Selection, s.List.CurrentCell = &Selection[string]{}, true
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}}
	tt := NewTester(func(c *Context) {
		OutlineTable(c, &s, cols, []string{"src", "docs"}, children, func(path string, col int) {
			Textf(c, "%s/%d", baseName(path), col).SingleLine()
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("src/0")
	tree := accessOn(tt)
	if n, _ := accessByID(tree, tree.Focus); n.Role != platform.RoleTreeItem && n.Role != platform.RoleRow {
		t.Fatalf("focus is %+v, want the row", n)
	}
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleCell && (n.Actions&platform.ActionFocus != 0 || n.Description != "") {
			t.Errorf("a cell of an outline changed: %+v", n)
		}
	}
	tt.Key(0, KeyRight)
	if n, _ := accessByID(accessOn(tt), tree.Focus); n.States&platform.AccessExpanded == 0 {
		t.Errorf("the row focused is not expanded after Right: %+v", n)
	}
}

// A cell holding an element of its own is not named by its texts: the element
// names itself, once.
func TestCurrentCellMixedCellsAreNotNamed(t *testing.T) {
	s := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	done := []bool{false, true, false}
	cols := []TableColumn{{Title: "Done"}, {Title: "Progress"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 3, func(row, col int) {
			if col == 0 {
				Checkbox(c, &done[row], fmt.Sprintf("done%d", row))
			} else {
				Progress(c, 0.5)
				Textf(c, "half%d", row)
			}
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("half1")
	tree := accessOn(tt)
	count := func(label string) (n int) {
		for _, x := range tree.Nodes {
			if x.Label == label {
				n++
			}
		}
		return
	}
	if got := count("done0"); got != 1 {
		t.Errorf("done0 is read by %d nodes, want the check box alone", got)
	}
	if got := count("half1"); got != 1 {
		t.Errorf("half1 is read by %d nodes, want its text alone", got)
	}
	cell := focusedCell(t, tree)
	if cell.Description != "Progress" || cell.Label != "" {
		t.Errorf("focused mixed cell %q (%q), want it unnamed in column Progress", cell.Label, cell.Description)
	}
}

// A table that tracks cells beside a plain one: the plain one keeps its tree
// and its row as the focus.
func TestPlainTableBesideTrackedTable(t *testing.T) {
	tracked := &ListState{CurrentCell: true, Selection: &Selection[int]{}}
	plain := &ListState{Selection: &Selection[int]{}}
	cols := []TableColumn{{Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		Column(c).Grow(1).Children(func() {
			Table(c, tracked, cols, 3, func(row, col int) { Textf(c, "t%dc%d", row, col) }).Height(120)
			Table(c, plain, cols, 3, func(row, col int) { Textf(c, "p%dc%d", row, col) }).Height(120)
		})
	}, 400, 300)
	tt.Frame()
	tt.Click("p1c1")
	tree := accessOn(tt)
	if n, _ := accessByID(tree, tree.Focus); n.Role != platform.RoleRow {
		t.Fatalf("focus in the plain table is %+v, want its row", n)
	}
	for i, n := range tree.Nodes {
		if n.Role == platform.RoleCell && n.Label == "" && n.Description == "" && n.Actions&platform.ActionFocus == 0 {
			if i+1 >= len(tree.Nodes) || tree.Nodes[i+1].Parent != i || tree.Nodes[i+1].Role != platform.RoleText {
				t.Errorf("an untracked cell %d lost its text", i)
			}
		}
	}
	named := 0
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleCell && n.Label != "" {
			named++
		}
	}
	if named != 6 { // the tracked table's cells, 3 rows of 2
		t.Errorf("%d cells are named, want the tracked table's 6", named)
	}
	tt.Click("t2c1")
	if got := focusedCell(t, accessOn(tt)); got.Label != "t2c1" {
		t.Errorf("focus in the tracked table is %q, want t2c1", got.Label)
	}
}

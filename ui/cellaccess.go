package ui

import "github.com/egoist/mygo/internal/platform"

// cellAccessHooks are the parts of tables with a current cell that the
// accessibility tree reaches, so that apps without them do not link them.
type cellAccessHooks struct {
	// cell describes a cell and reports whether its texts, which name it,
	// are all it holds, and gives it the focus of the tree while it is the
	// table's current cell. focus does what a click on a cell does for
	// assistive technology focusing it.
	cell  func(rt *engine, t *platform.AccessTree, e *node, n *platform.AccessNode) bool
	focus func(rt *engine, id uint64) bool
}

var cellAccessFuncs = cellAccessHooks{
	cell:  accessCellDetails,
	focus: (*engine).focusCell,
}

// accessCellOf returns the list frame and the column's place of e when it is
// a cell of a Table that keeps a current column (CurrentCell) or chooses
// cells (CellSelect), and its row.
func accessCellOf(e *node) (f *listFrame, k int, row *node) {
	// The cells are in a box inside the row's element.
	var b *listRowInput
	for row = e.parent; row != nil; row = row.parent {
		if row.st != nil {
			if b, _ = row.st.valueBinding.(*listRowInput); b != nil {
				break
			}
		}
		if row != e.parent {
			return nil, 0, nil
		}
	}
	if b == nil || !b.f.tracksCells() {
		return nil, 0, nil
	}
	id, ok := e.key.(string)
	if !ok {
		return nil, 0, nil
	}
	if k = b.f.s.colIndex(id); k < 0 {
		return nil, 0, nil
	}
	return b.f, k, row
}

// accessCellDetails names a cell of such a Table by what is inside it, tells
// its column in the description and lets assistive technology focus it; the
// current cell of the row the keys move from has the tree's focus while the
// table has the keyboard's. It reports whether the texts are all that is
// inside.
func accessCellDetails(rt *engine, t *platform.AccessTree, e *node, n *platform.AccessNode) bool {
	f, k, row := accessCellOf(e)
	if f == nil {
		return false
	}
	if n.Label == "" {
		n.Label = e.innerText()
	}
	if n.Description == "" {
		n.Description = f.s.tableCols[k].title
	}
	if !e.IsDisabled() && !e.st.ignoreInput {
		n.States |= platform.AccessFocusable
		n.Actions |= platform.ActionFocus
	}
	if t.Focus == 0 && rt.windowFocus() && rt.focused == f.owner.id && k == f.s.colIndex(f.s.Cells.Cursor) {
		if c := f.s.cursor(); c != nil && *c == row.st.valueBinding.(*listRowInput).i {
			t.Focus = e.id
		}
	}
	return n.Label != "" && textOnly(e)
}

// textOnly reports whether what is inside e shows no element of assistive
// technology but texts, which a name from content already gives.
func textOnly(e *node) bool {
	for ch := e.first; ch != nil; ch = ch.next {
		if ch.flags&(flagInvisible|flagInert) != 0 || ch.hidden {
			continue
		}
		if r, ok := ch.accessRole(); ok && r != platform.RoleText || !textOnly(ch) {
			return false
		}
	}
	return true
}

// elementByID returns the element with id inside e.
func elementByID(e *node, id uint64) *node {
	if e.id == id {
		return e
	}
	for ch := e.first; ch != nil; ch = ch.next {
		if n := elementByID(ch, id); n != nil {
			return n
		}
	}
	return nil
}

// focusCell does for assistive technology moving to a cell what a click on
// it does: it chooses the row and, with the row's click in the next frame,
// the column (takeAccessPick). It reports whether id is such a cell.
func (rt *engine) focusCell(id uint64) bool {
	if rt.c.root == nil {
		return false
	}
	e := elementByID(rt.c.root, id)
	if e == nil || e.role != RoleCell {
		return false
	}
	f, k, rowNode := accessCellOf(e)
	if f == nil {
		return false
	}
	row := rt.states[rowNode.id]
	if row == nil || row.flags&flagDisabled != 0 || row.ignoreInput || row.seen != rt.frame || row.pass != rt.pass {
		return false
	}
	f.s.accessPick.set, f.s.accessPick.col, f.s.accessPick.frame = true, f.s.tableCols[k].id, rt.frame
	row.click(0)
	return true
}

// takeAccessPick sets the columns as a click on the cell focused (focusCell)
// does, once the row it queued is clicked.
func (f *listFrame) takeAccessPick() {
	p := &f.s.accessPick
	if !p.set {
		return
	}
	p.set = false
	if p.frame+1 != f.frame || f.c.rt.cellColumn == nil {
		return
	}
	f.c.rt.cellColumn(f, f.s.colIndex(p.col), 0)
}

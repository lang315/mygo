package ui

import "github.com/egoist/mygo/internal/platform"

// accessCell is a cell of a table with a current cell or choosing cells, as
// assistive technology sees it: its row's element and its column's id.
type accessCell struct {
	row uint64
	col string
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
// its column in the description and lets assistive technology focus it, and
// notes it for focusCell.
func (rt *engine) accessCellDetails(e *node, n *platform.AccessNode) {
	f, k, row := accessCellOf(e)
	if f == nil {
		return
	}
	if n.Label == "" {
		n.Label = e.innerText()
	}
	if n.Description == "" {
		n.Description = f.s.tableCols[k].title
	}
	if rt.accessCells == nil {
		rt.accessCells = map[uint64]accessCell{}
	}
	rt.accessCells[e.id] = accessCell{row: row.id, col: f.s.tableCols[k].id}
	if !e.IsDisabled() && !e.st.ignoreInput {
		n.States |= platform.AccessFocusable
		n.Actions |= platform.ActionFocus
	}
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

// currentCellID returns the id of the cell in the current column of the
// element of row, 0 when the list keeps none.
func (f *listFrame) currentCellID(row *node) uint64 {
	if !f.tracksCells() {
		return 0
	}
	k := f.s.colIndex(f.s.Cells.Cursor)
	if k < 0 {
		return 0
	}
	id := f.s.tableCols[k].id
	// The cells are in a box inside the row's element.
	for box := row.first; box != nil; box = box.next {
		for ch := box.first; ch != nil; ch = ch.next {
			if key, ok := ch.key.(string); ok && key == id && ch.role == RoleCell {
				return ch.id
			}
		}
	}
	return 0
}

// focusCell does for assistive technology moving to a cell what a click on
// it does: it chooses the row and, with the row's click in the next frame,
// the column (takeAccessPick). It reports whether id is such a cell.
func (rt *engine) focusCell(id uint64) bool {
	c, ok := rt.accessCells[id]
	if !ok {
		return false
	}
	row := rt.states[c.row]
	if row == nil || row.flags&flagDisabled != 0 || row.ignoreInput || row.seen != rt.frame || row.pass != rt.pass {
		return false
	}
	b, ok := row.valueBinding.(*listRowInput)
	if !ok {
		return false
	}
	b.f.s.accessPick.set, b.f.s.accessPick.col, b.f.s.accessPick.frame = true, c.col, rt.frame
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

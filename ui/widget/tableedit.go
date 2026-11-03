package widget

import "github.com/egoist/mygo/ui"

// EditMode says what starts the edit of a cell of a DataGridView whose
// columns have a Set.
type EditMode int

const (
	// EditOnEnterOrF2 starts it with F2, Enter or a double click on an
	// editable cell; on a read-only cell Enter and the double click activate
	// the row as they do in a grid that edits nothing. The default.
	EditOnEnterOrF2 EditMode = iota
	// EditOnF2 starts it with F2 only; Enter and the double click keep
	// activating the row (OnActivate).
	EditOnF2
	// EditProgrammatically starts it with BeginEdit only.
	EditProgrammatically
)

// editState is the edit of a cell in progress: the draft lives here, in the
// table, and never in the row or the source until the edit is written. Under mu.
type editState[K comparable] struct {
	key              K
	col              int    // the column's index in cols
	orig, draft      string // the text the editor started with, and the user's
	err              string // the last refusal, shown by the editor
	fresh, selectAll bool   // the next build gives the editor the focus; all selected
	at               struct {
		has  bool
		x, y float32
	}
	gen     uint64 // versions the edited row while the edit lasts
	focused bool   // the editor held the focus in the last frame
}

// curCell is a current cell asked for by SetCurrentCell or BeginEdit, which
// the next build hands to ui.
type curCell[K comparable] struct {
	key K
	col string
}

// SetReadOnly makes every cell of the grid read-only, as WinForms'
// DataGridView.ReadOnly does: no cell edits, whatever its column's Set.
func (t *DataGridViewWidget[T, K]) SetReadOnly(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.readOnly = on
		if on {
			t.closeEdit()
		}
	})
	return t
}

// ReadOnly reports whether the grid is read-only (SetReadOnly).
func (t *DataGridViewWidget[T, K]) ReadOnly() bool {
	mu.Lock()
	defer mu.Unlock()
	return t.readOnly
}

// SetEditMode sets what starts the edit of a cell: EditOnEnterOrF2 by default.
func (t *DataGridViewWidget[T, K]) SetEditMode(m EditMode) *DataGridViewWidget[T, K] {
	t.change(func() { t.editMode = m })
	return t
}

// CurrentCell returns the key of the row and the title of the column of the
// current cell, as of the last frame: the one the keys act on.
func (t *DataGridViewWidget[T, K]) CurrentCell() (k K, column string, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	return t.curKey, t.curCol, t.curOK
}

// SetCurrentCell makes the cell of the row of key k and the column titled
// column the current one, chooses its row, and scrolls it into view at the
// next frame. It reports false for a row that is not there or a column that
// is hidden or unknown.
func (t *DataGridViewWidget[T, K]) SetCurrentCell(k K, column string) bool {
	ok := false
	t.change(func() {
		if _, _, found := t.cellOf(k, column); found {
			t.pendingCur, ok = &curCell[K]{k, column}, true
		}
	})
	return ok
}

// BeginEdit starts editing the cell of the row of key k and the column titled
// column, which becomes the current cell; the editor shows at the next frame.
// It reports false for a cell that is read-only, hidden, or not there.
func (t *DataGridViewWidget[T, K]) BeginEdit(k K, column string) bool {
	ok := false
	t.change(func() {
		i, ci, found := t.cellOf(k, column)
		if !found || !t.canEdit(&t.src.rows[i], ci) {
			return
		}
		t.beginLocked(k, ci, false, 0, 0)
		t.pendingCur, ok = &curCell[K]{k, column}, true
	})
	return ok
}

// CancelEdit ends the edit in progress and discards the draft.
func (t *DataGridViewWidget[T, K]) CancelEdit() {
	t.change(func() { t.closeEdit() })
}

// IsEditing reports whether a cell is being edited.
func (t *DataGridViewWidget[T, K]) IsEditing() bool {
	mu.Lock()
	defer mu.Unlock()
	return t.edit != nil
}

// EditingCell returns the row's key and the column's title of the cell being
// edited.
func (t *DataGridViewWidget[T, K]) EditingCell() (k K, column string, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	if e := t.edit; e != nil {
		return e.key, t.cols[e.col].Title, true
	}
	return k, "", false
}

// OnCellEndEdit runs fn, on the event goroutine with no lock held, with the
// row's key and the column's title of each edit that was written.
func (t *DataGridViewWidget[T, K]) OnCellEndEdit(fn func(k K, column string)) *DataGridViewWidget[T, K] {
	t.change(func() { t.onEndEdit = fn })
	return t
}

// editDraft returns the draft of the edit in progress, for tests.
func (t *DataGridViewWidget[T, K]) editDraft() string {
	mu.Lock()
	defer mu.Unlock()
	if t.edit == nil {
		return ""
	}
	return t.edit.draft
}

// cellOf finds the row of key k in the source and the visible column titled
// column, as indexes into src.rows and cols. Under mu.
func (t *DataGridViewWidget[T, K]) cellOf(k K, column string) (row, col int, ok bool) {
	row, ok = t.src.index[k]
	if !ok {
		return 0, 0, false
	}
	for _, ci := range t.visCols {
		if t.cols[ci].Title == column {
			return row, ci, true
		}
	}
	return 0, 0, false
}

// canEdit reports whether the cell of row r in column ci may be edited: the
// grid is not read-only, the column has a Set, and CanEdit allows it. Under mu.
func (t *DataGridViewWidget[T, K]) canEdit(r *T, ci int) bool {
	cl := &t.cols[ci]
	if t.readOnly || cl.Set == nil || t.watch != nil { // a ListView never edits
		return false
	}
	if cl.CanEdit == nil {
		return true
	}
	allowed := false
	if v, stack, panicked := runColumnFn(func() { allowed = cl.CanEdit(r) }); panicked {
		Post(func() { report(v, stack) })
		return false
	}
	return allowed
}

// editText returns the text the editor of the cell starts with. Under mu.
func (t *DataGridViewWidget[T, K]) editText(r *T, ci int) string {
	cl := &t.cols[ci]
	text := ""
	if v, stack, panicked := runColumnFn(func() {
		if cl.EditText != nil {
			text = cl.EditText(r)
		} else {
			text = cl.Text(r)
		}
	}); panicked {
		Post(func() { report(v, stack) })
	}
	return text
}

// beginLocked starts the edit of the cell of key k in column ci. Under mu.
func (t *DataGridViewWidget[T, K]) beginLocked(k K, ci int, hasAt bool, x, y float32) {
	i, ok := t.src.index[k]
	if !ok {
		return
	}
	text := t.editText(&t.src.rows[i], ci)
	t.editGen++
	e := &editState[K]{key: k, col: ci, orig: text, draft: text, fresh: true, selectAll: !hasAt, gen: t.editGen}
	e.at.has, e.at.x, e.at.y = hasAt, x, y
	t.edit = e
}

// closeEdit ends the edit without writing it. Under mu.
func (t *DataGridViewWidget[T, K]) closeEdit() {
	if e := t.edit; e != nil {
		t.refocus = t.refocus || e.focused // the table takes the focus the editor held
		t.edit = nil
		t.editGen++
	}
}

// touch redraws the windows that show the table, from a callback that runs
// in a frame, off the lock.
func (t *DataGridViewWidget[T, K]) touch() { t.change(func() {}) }

// editsCell is ui's ListState.EditCell: does the cell at row, in the column
// col of t.columns, edit when started by how?
func (t *DataGridViewWidget[T, K]) editsCell(row, col int, how ui.EditHow) bool {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if t.readOnly || t.editMode == EditProgrammatically || (t.editMode == EditOnF2 && how != ui.EditF2) {
		return false
	}
	ci, ok := t.colAt(col)
	if !ok {
		return false
	}
	r := t.rowAt(row)
	return r != nil && t.canEdit(r, ci)
}

// colAt returns the index in cols of the column ui shows at col of t.columns.
func (t *DataGridViewWidget[T, K]) colAt(col int) (int, bool) {
	if t.numbered {
		col--
	}
	if col < 0 || col >= len(t.visCols) {
		return 0, false
	}
	return t.visCols[col], true
}

// startEdit is ui's ListState.OnEditCell: the cell at row and col edits.
func (t *DataGridViewWidget[T, K]) startEdit(row, col int, x, y float32, how ui.EditHow) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	ci, ok := t.colAt(col)
	r := t.rowAt(row)
	if !ok || r == nil {
		return
	}
	t.beginLocked(t.key(r), ci, how == ui.EditClick, x, y)
	Post(t.touch) // the editor builds in the next frame
}

// wireEdit gives ui the hooks of editing, or takes them away. Under mu.
func (t *DataGridViewWidget[T, K]) wireEdit() {
	s := &t.list
	if !t.editable || t.watch != nil { // a ListView never edits
		return
	}
	s.CurrentCell = true
	s.EditCell, s.OnEditCell = t.editsCell, t.startEdit
}

// followCurrent applies a SetCurrentCell or BeginEdit, and keeps what the
// last frame left as the current cell. Under mu, while building.
func (t *DataGridViewWidget[T, K]) followCurrent() {
	if !t.editable || t.watch != nil {
		return
	}
	s := &t.list
	if p := t.pendingCur; p != nil {
		t.pendingCur = nil
		if line := t.lineOf(p.key); line >= 0 {
			for k := range t.chosen.value.All() {
				t.chosen.set(k, false)
			}
			t.chosen.set(p.key, true)
			t.cursor = line
			s.Cells = ui.ColumnRange{Cursor: p.col, Anchor: p.col}
			s.ScrollIntoView(line)
		}
	}
	t.curOK = false
	if r := t.rowAt(t.cursor); r != nil {
		col := s.Cells.Cursor
		if col == "" && len(t.visCols) > 0 {
			col = t.cols[t.visCols[0]].Title
		}
		t.curKey, t.curCol, t.curOK = t.key(r), col, true
	}
}

// editor builds the editor of the cell being edited in place of its text.
// Under mu, while building.
func (t *DataGridViewWidget[T, K]) editor(c *ui.Context, e *editState[K]) {
	start := ui.CellStart{Fresh: e.fresh, SelectAll: e.selectAll, HasAt: e.at.has, X: e.at.x, Y: e.at.y}
	if e.fresh && !c.Measuring() {
		c.Announce("Editing " + t.cols[e.col].Title)
		e.fresh = false
	}
	in := ui.CellInput(c, &e.draft, start)
	e.focused = in.Focused() || e.focused && e.fresh
	if e.err != "" {
		in.Error(e.err)
	}
	in.OnShortcut(0, ui.KeyEscape, func() { Post(t.CancelEdit) })
}

package widget

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

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

// editState is the edit of a cell in progress: its core, which names no type, and
// the row's key. Under mu.
type editState[K comparable] struct {
	editCore
	key K
}

// curCell is a current cell asked for by SetCurrentCell or BeginEdit, which
// the next build hands to ui.
type curCell[K comparable] struct {
	key K
	col string
}

// enableEdit turns in-cell editing on: it hands the table the parts of editing
// it calls (method values, made once: one made in every build would allocate).
// EditColumn and the other editing methods call it. Nothing else refers to the
// parts, and these methods are called, not reached through an interface, so
// the code of editing is linked only for the table shapes (T, K) of programs
// that call them. Under mu.
func (t *DataGridViewWidget[T, K]) enableEdit() {
	if t.edAny != nil {
		return
	}
	d := &tableEditing[T, K]{}
	t.edAny = d
	t.editCell, t.startCell = t.editsCell, t.startEdit
	t.prepareEdit = func() { t.wireEdit(); t.followCurrent(); t.dropHiddenEdit() }
	t.cellHook = func(c *ui.Context, col int, r any) bool {
		if e := d.edit; e != nil && col == e.col && t.key(r.(*T)) == e.key {
			e.canNext, e.canPrev = t.mayTab(e, 1), t.mayTab(e, -1)
			buildEditor(c, &e.editCore)
			return true
		}
		return false
	}
	t.versionHook = func(r any) uint64 {
		if e := d.edit; e != nil && t.key(r.(*T)) == e.key {
			return e.gen
		}
		return 0
	}
	t.pruneHook = func() {
		if e := d.edit; e != nil {
			if _, ok := t.src.index[e.key]; !ok { // the row of the edit went
				t.closeEdit()
			}
		}
	}
	t.afterTable = func(e ui.Element, c *ui.Context) {
		if d.refocus && !c.Measuring() {
			d.refocus = false
			e.Focus()
		}
	}
}

// tableEditing is the state of in-cell editing of a table, kept in the table as
// an any so that the table's own type does not name it: a program that never
// edits does not link its types. Under mu.
type tableEditing[T any, K comparable] struct {
	edit       *editState[K] // the edit in progress
	editGen    uint64        // versions the edited row
	editable   bool          // a column is Editable
	readOnly   bool
	editMode   EditMode
	onEndEdit  func(K, string)
	pendingCur *curCell[K] // a current cell asked for, for the next build
	curKey     K           // the current cell of the last frame
	curCol     string
	curOK      bool
	refocus    bool           // an edit ended while its editor had the focus: the table takes it
	ended      []endedEdit[K] // edits written, for OnCellEndEdit, delivered by flushEdit
	panics     []reported     // panics of column functions, for OnPanic
}

// ed returns the table's editing state, nil until editing is turned on.
func (t *DataGridViewWidget[T, K]) ed() *tableEditing[T, K] {
	d, _ := t.edAny.(*tableEditing[T, K])
	return d
}

// EditColumn makes the cells of the column titled title editable in place, as
// the CellEditor says: F2, Enter or a double click on a cell of the current row
// edit it (SetEditMode changes that), Enter writes, Escape cancels. A column
// not given to EditColumn is read-only. It panics for a title no column has,
// or an editor without a Set.
//
// Calling it is also what links the code of editing into the program, for this
// table's types only: a table that never calls it pays nothing for editing.
//
//	grid := widget.DataGridView(key, cols...).
//		EditColumn("Name", widget.CellEditor[Account]{Set: func(a *Account, s string) { a.Name = s }})
func (t *DataGridViewWidget[T, K]) EditColumn(title string, e CellEditor[T]) *DataGridViewWidget[T, K] {
	if e.Set == nil {
		panic("widget: EditColumn needs a Set")
	}
	t.change(func() {
		found := false
		for i := range t.cols {
			if t.cols[i].Title == title {
				t.cols[i].edit, found = &e, true
			}
		}
		if !found {
			panic("widget: EditColumn: no column is titled " + strconv.Quote(title))
		}
		t.enableEdit()
		t.ed().editable = true
	})
	return t
}

// SetReadOnly makes every cell of the grid read-only, as WinForms'
// DataGridView.ReadOnly does: no cell edits, whatever its column's Set.
func (t *DataGridViewWidget[T, K]) SetReadOnly(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.enableEdit()
		t.ed().readOnly = on
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
	d := t.ed()
	return d != nil && d.readOnly
}

// SetEditMode sets what starts the edit of a cell: EditOnEnterOrF2 by default.
// The cells of the columns given to EditColumn are the editable ones.
func (t *DataGridViewWidget[T, K]) SetEditMode(m EditMode) *DataGridViewWidget[T, K] {
	t.change(func() { t.enableEdit(); t.ed().editMode = m })
	return t
}

// CurrentCell returns the key of the row and the title of the column of the
// current cell, as of the last frame: the one the keys act on.
func (t *DataGridViewWidget[T, K]) CurrentCell() (k K, column string, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	if d := t.ed(); d != nil {
		return d.curKey, d.curCol, d.curOK
	}
	return k, "", false
}

// SetCurrentCell makes the cell of the row of key k and the column titled
// column the current one, chooses its row, and scrolls it into view at the
// next frame. It reports false for a row that is not there or a column that is
// hidden or unknown, and for a grid with no editable column (EditColumn),
// which keeps no current cell.
func (t *DataGridViewWidget[T, K]) SetCurrentCell(k K, column string) bool {
	ok := false
	t.change(func() {
		d := t.ed()
		if d == nil || !d.editable {
			return
		}
		if _, _, found := t.cellOf(k, column); found {
			d.pendingCur, ok = &curCell[K]{k, column}, true
		}
	})
	return ok
}

// BeginEdit starts editing the cell of the row of key k and the column titled
// column, which becomes the current cell; the editor shows at the next frame.
// An edit already open is written first (EndEdit); if that is refused the new
// one does not open and BeginEdit reports false, the refused edit keeping its
// draft and the focus. It also reports false for a cell that is read-only,
// hidden, or not there, and for a grid with no editable column (EditColumn).
func (t *DataGridViewWidget[T, K]) BeginEdit(k K, column string) bool {
	ok := false
	var wins []*mygo.Window
	t.change(func() {
		d := t.ed()
		if d == nil || !d.editable {
			return
		}
		if _, ci, found := t.cellOf(k, column); !found {
			return
		} else if wins, ok = t.beginLocked(k, ci, false, 0, 0); ok {
			d.pendingCur = &curCell[K]{k, column}
		}
	})
	for _, w := range wins {
		invalidate(w)
	}
	t.flushEdit()
	return ok
}

// CancelEdit ends the edit in progress and discards the draft.
func (t *DataGridViewWidget[T, K]) CancelEdit() { t.cancelEdit(nil) }

// cancelEdit ends the edit only, or the one in progress when only is nil: a
// cancel posted by the editor must not end an edit begun after it.
func (t *DataGridViewWidget[T, K]) cancelEdit(only *editState[K]) {
	t.change(func() {
		if d := t.ed(); d != nil && (only == nil || d.edit == only) {
			t.closeEdit()
		}
	})
}

// IsEditing reports whether a cell is being edited.
func (t *DataGridViewWidget[T, K]) IsEditing() bool {
	mu.Lock()
	defer mu.Unlock()
	d := t.ed()
	return d != nil && d.edit != nil
}

// EditingCell returns the row's key and the column's title of the cell being
// edited.
func (t *DataGridViewWidget[T, K]) EditingCell() (k K, column string, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	if d := t.ed(); d != nil && d.edit != nil {
		return d.edit.key, t.cols[d.edit.col].Title, true
	}
	return k, "", false
}

// OnCellEndEdit runs fn, on the event goroutine with no lock held, with the
// row's key and the column's title of each edit that was written.
func (t *DataGridViewWidget[T, K]) OnCellEndEdit(fn func(k K, column string)) *DataGridViewWidget[T, K] {
	t.change(func() { t.enableEdit(); t.ed().onEndEdit = fn })
	return t
}

// editDraft returns the draft of the edit in progress, for tests.
func (t *DataGridViewWidget[T, K]) editDraft() string {
	mu.Lock()
	defer mu.Unlock()
	if d := t.ed(); d == nil || d.edit == nil {
		return ""
	} else {
		return d.edit.draft
	}
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
	if t.ed().readOnly || cl.edit == nil || cl.editor().Set == nil || t.watch != nil { // a ListView never edits
		return false
	}
	if cl.editor().CanEdit == nil {
		return true
	}
	allowed := false
	if v, stack, panicked := runColumnFn(func() { allowed = cl.editor().CanEdit(r) }); panicked {
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
		if cl.edit != nil && cl.editor().EditText != nil {
			text = cl.editor().EditText(r)
		} else {
			text = cl.Text(r)
		}
	}); panicked {
		Post(func() { report(v, stack) })
	}
	return text
}

// beginLocked starts the edit of the cell of key k in column ci, writing the one
// open before it. It returns the windows that write asks to redraw, and false
// when the open edit was refused (it stays, to be fixed) or the row is gone.
// Under mu.
func (t *DataGridViewWidget[T, K]) beginLocked(k K, ci int, hasAt bool, x, y float32) ([]*mygo.Window, bool) {
	d := t.ed()
	i, found := t.src.index[k]
	if !found {
		return nil, false
	}
	var wins []*mygo.Window
	if old := d.edit; old != nil {
		if old.key == k && old.col == ci { // already editing this cell
			return nil, true
		}
		var err error
		if wins, err = t.commitEditLocked(nil); err != nil {
			// Refused: it stays open, with its error, which is read again; and
			// it takes the focus back if the editor held it, not from elsewhere.
			old.announced = ""
			if old.focused {
				old.fresh, old.selectAll, old.hadFocus, old.at.has = true, false, false, false
			}
			return wins, false
		}
	}
	// Asked once the open edit is written: CanEdit may read what it wrote.
	if !t.canEdit(&t.src.rows[i], ci) {
		return wins, false
	}
	text := t.editText(&t.src.rows[i], ci)
	d.editGen++
	e := &editState[K]{key: k, editCore: editCore{col: ci, title: t.cols[ci].Title, orig: text, draft: text,
		fresh: true, selectAll: !hasAt, gen: d.editGen}}
	e.at.has, e.at.x, e.at.y = hasAt, x, y
	e.end = func() { _ = t.endEdit(e) }
	e.cancel = func() { t.cancelEdit(e) }
	e.tab = func(dir int) { t.tabFrom(e, dir) }
	d.edit, d.refocus = e, false // the new editor takes the focus, not the table
	return wins, true
}

// mayTab is a cheap guess, made while building, at whether Tab (dir 1) or
// Shift+Tab (dir -1) has a cell to go to: another editable column in the row, or
// another row. It does not ask CanEdit; tabFrom does. Under mu.
func (t *DataGridViewWidget[T, K]) mayTab(e *editState[K], dir int) bool {
	at := slices.Index(t.visCols, e.col)
	if at < 0 {
		return false
	}
	for j := at + dir; j >= 0 && j < len(t.visCols); j += dir {
		if t.cols[t.visCols[j]].edit != nil {
			return true
		}
	}
	pos := t.positionOf(e.key)
	return pos >= 0 && t.rowAtPos(pos+dir) != nil
}

// nextEditable finds the cell Tab (dir 1) or Shift+Tab (dir -1) goes to from
// the edit e: the next editable cell of its row, then the first of the next
// row in the order the table shows now, before any commit re-sorts it. Under mu.
func (t *DataGridViewWidget[T, K]) nextEditable(e *editState[K], dir int) (k K, ci int, ok bool) {
	at := slices.Index(t.visCols, e.col)
	row, there := t.src.index[e.key]
	if at < 0 || !there {
		return k, 0, false
	}
	for j := at + dir; j >= 0 && j < len(t.visCols); j += dir {
		if t.canEdit(&t.src.rows[row], t.visCols[j]) {
			return e.key, t.visCols[j], true
		}
	}
	pos := t.positionOf(e.key)
	if pos < 0 {
		return k, 0, false
	}
	for p := pos + dir; ; p += dir {
		r := t.rowAtPos(p)
		if r == nil {
			return k, 0, false
		}
		for j := range t.visCols {
			if dir < 0 {
				j = len(t.visCols) - 1 - j
			}
			if t.canEdit(r, t.visCols[j]) {
				return t.key(r), t.visCols[j], true
			}
		}
	}
}

// tabFrom is Tab (dir 1) or Shift+Tab (dir -1) in the editor of e: the draft is
// written and the next editable cell edits. The cell is chosen before the
// write, which may re-sort the rows. A refusal keeps the edit where it is;
// with no cell to go to the edit is written and ends.
func (t *DataGridViewWidget[T, K]) tabFrom(e *editState[K], dir int) {
	mu.Lock()
	if t.ed().edit != e { // an edit that already ended
		mu.Unlock()
		return
	}
	k, ci, ok := t.nextEditable(e, dir)
	title := t.cols[ci].Title
	mu.Unlock()
	if !ok {
		_ = t.endEdit(e)
		return
	}
	t.BeginEdit(k, title)
}

// closeEdit ends the edit without writing it. Under mu.
func (t *DataGridViewWidget[T, K]) closeEdit() {
	if e := t.ed().edit; e != nil {
		t.ed().refocus = t.ed().refocus || e.focused // the table takes the focus the editor held
		t.ed().edit = nil
		t.ed().editGen++
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
	if t.ed().readOnly || t.ed().editMode == EditProgrammatically || (t.ed().editMode == EditOnF2 && how != ui.EditF2) {
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
	wins, _ := t.beginLocked(t.key(r), ci, how == ui.EditClick, x, y)
	Post(func() { // the editor builds in the next frame
		for _, w := range wins {
			invalidate(w)
		}
		t.flushEdit() // the end of the edit it wrote, and the panics of Validate and Set
		t.touch()
	})
}

// wireEdit gives ui the hooks of editing, or takes them away. Under mu.
func (t *DataGridViewWidget[T, K]) wireEdit() {
	s := &t.list
	if !t.ed().editable || t.watch != nil { // a ListView never edits
		return
	}
	s.CurrentCell = true
	s.EditCell, s.OnEditCell = t.editCell, t.startCell
}

// followCurrent applies a SetCurrentCell or BeginEdit, and keeps what the
// last frame left as the current cell. Under mu, while building.
func (t *DataGridViewWidget[T, K]) followCurrent() {
	if !t.ed().editable || t.watch != nil {
		return
	}
	s := &t.list
	if p := t.ed().pendingCur; p != nil {
		t.ed().pendingCur = nil
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
	t.ed().curOK = false
	if r := t.rowAt(t.cursor); r != nil {
		col := s.Cells.Cursor
		if col == "" && len(t.visCols) > 0 {
			col = t.cols[t.visCols[0]].Title
		}
		t.ed().curKey, t.ed().curCol, t.ed().curOK = t.key(r), col, true
	}
}

// dropHiddenEdit cancels the edit of a column or a row that no longer shows (a
// hidden column, a search that filters the row out): its editor is gone, and a
// later commit would write a draft nobody sees. Under mu, while building.
func (t *DataGridViewWidget[T, K]) dropHiddenEdit() {
	if e := t.ed().edit; e != nil && (!slices.Contains(t.visCols, e.col) || t.positionOf(e.key) < 0) {
		t.closeEdit()
	}
}

// endedEdit is an edit that was written, for OnCellEndEdit.
type endedEdit[K comparable] struct {
	key    K
	column string
}

// EndEdit commits the edit in progress now, with the draft the table holds: it
// asks the column's Validate, and when that allows it writes the draft with
// the column's Set, in the row as it is. It returns the refusal's error, which
// the editor then shows (the edit stays open with its draft); nil when it
// wrote, when the draft is the text the editor started with (nothing is
// written, nothing reported), or when nothing was being edited. A Save
// button's handler calls it before it saves.
//
// A panic in Validate or Set is recovered and reported with OnPanic, and the
// edit is cancelled; EndEdit returns nil then. Do not call it from a column
// function: those run under the package's lock.
func (t *DataGridViewWidget[T, K]) EndEdit() error { return t.endEdit(nil) }

// endEdit is EndEdit for the edit only, or for the one in progress when only is
// nil: a commit posted by the editor must not end an edit begun after it.
func (t *DataGridViewWidget[T, K]) endEdit(only *editState[K]) error {
	var wins []*mygo.Window
	var err error
	func() {
		mu.Lock()
		defer mu.Unlock() // a column function may panic; runColumnFn recovers, and the unlock stays a defer
		wins, err = t.commitEditLocked(only)
	}()
	for _, w := range wins {
		invalidate(w)
	}
	t.flushEdit()
	if err != nil {
		t.touch() // the editor builds again with the error
	}
	return err
}

// commitEditLocked writes the draft of the edit, or refuses it. Under mu. The
// windows to redraw are for the caller to invalidate once it released mu. The
// row is changed through updateLocked on a copy, assigned back only when Set
// returns: a Set that panics after writing a field leaves the row, its
// generation and the views as they were.
func (t *DataGridViewWidget[T, K]) commitEditLocked(only *editState[K]) ([]*mygo.Window, error) {
	d := t.ed()
	if d == nil {
		return nil, nil
	}
	e := d.edit
	if e == nil || only != nil && e != only {
		return nil, nil
	}
	i, rowThere := t.src.index[e.key]
	ed := t.cols[e.col].editor()
	var validate func(string) error
	if rowThere && ed.Validate != nil {
		r := &t.src.rows[i]
		validate = func(draft string) error { return ed.Validate(r, draft) }
	}
	write := func(draft string) []*mygo.Window {
		wins, _ := t.src.updateLocked(e.key, func(row *T) {
			n := *row
			ed.Set(&n, draft)
			*row = n
		})
		return wins
	}
	out := commitCore(&e.editCore, rowThere, validate, write)
	switch out.kind {
	case committedRefused:
		d.editGen++
		e.gen = d.editGen
		return nil, out.err
	case committedWrote:
		d.ended = append(d.ended, endedEdit[K]{e.key, e.title})
	case committedPanicked:
		d.panics = append(d.panics, out.panic)
	}
	t.closeEdit()
	return out.wins, nil
}

// flushEdit delivers what commits collected under the lock: the edits written
// to OnCellEndEdit, the panics to OnPanic, both on the event goroutine.
func (t *DataGridViewWidget[T, K]) flushEdit() {
	mu.Lock()
	d := t.ed()
	if d == nil {
		mu.Unlock()
		return
	}
	ended, panics, fn := d.ended, d.panics, d.onEndEdit
	d.ended, d.panics = nil, nil
	mu.Unlock()
	postReports(panics)
	if fn == nil {
		return
	}
	for _, e := range ended {
		Post(func() { fn(e.key, e.column) })
	}
}

package widget

import (
	"slices"

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
	gen       uint64 // versions the edited row while the edit lasts
	focused   bool   // the editor held the focus in the last frame
	announced string // the refusal the user was told
	hadFocus  bool   // the editor has held the focus: losing it commits
}

// curCell is a current cell asked for by SetCurrentCell or BeginEdit, which
// the next build hands to ui.
type curCell[K comparable] struct {
	key K
	col string
}

// enableEdit turns in-cell editing on: it hands the table the parts of editing
// it calls (method values, made once: one made in every build would allocate).
// Column.Editable has the table call it through the column's enable function,
// and the editing methods that need it call it too. Nothing else refers to
// the parts, so the code of editing is linked only into programs that edit.
// Under mu.
func (t *DataGridViewWidget[T, K]) enableEdit() {
	if t.edAny != nil {
		return
	}
	d := &tableEditing[T, K]{}
	t.edAny = d
	d.editable = slices.ContainsFunc(t.cols, func(cl Column[T]) bool { return cl.edit != nil })
	t.editCell, t.startCell = t.editsCell, t.startEdit
	t.prepareEdit = func() { t.wireEdit(); t.followCurrent() }
	t.cellHook = func(c *ui.Context, col int, r *T) bool {
		if e := d.edit; e != nil && col == e.col && t.key(r) == e.key {
			t.editor(c, e)
			return true
		}
		return false
	}
	t.versionHook = func(r *T) uint64 {
		if e := d.edit; e != nil && t.key(r) == e.key {
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
// The cells of a column are editable when the column is Editable.
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
// next frame. It reports false for a row that is not there or a column that
// is hidden or unknown.
func (t *DataGridViewWidget[T, K]) SetCurrentCell(k K, column string) bool {
	ok := false
	t.change(func() {
		t.enableEdit()
		if _, _, found := t.cellOf(k, column); found {
			t.ed().pendingCur, ok = &curCell[K]{k, column}, true
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
		t.enableEdit()
		i, ci, found := t.cellOf(k, column)
		if !found || !t.canEdit(&t.src.rows[i], ci) {
			return
		}
		t.beginLocked(k, ci, false, 0, 0)
		t.ed().pendingCur, ok = &curCell[K]{k, column}, true
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

// beginLocked starts the edit of the cell of key k in column ci. Under mu.
func (t *DataGridViewWidget[T, K]) beginLocked(k K, ci int, hasAt bool, x, y float32) {
	i, ok := t.src.index[k]
	if !ok {
		return
	}
	text := t.editText(&t.src.rows[i], ci)
	t.ed().editGen++
	e := &editState[K]{key: k, col: ci, orig: text, draft: text, fresh: true, selectAll: !hasAt, gen: t.ed().editGen}
	e.at.has, e.at.x, e.at.y = hasAt, x, y
	t.ed().edit = e
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
	t.beginLocked(t.key(r), ci, how == ui.EditClick, x, y)
	Post(t.touch) // the editor builds in the next frame
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

// editor builds the editor of the cell being edited in place of its text.
// Under mu, while building.
func (t *DataGridViewWidget[T, K]) editor(c *ui.Context, e *editState[K]) {
	start := ui.CellStart{Fresh: e.fresh, SelectAll: e.selectAll, HasAt: e.at.has, X: e.at.x, Y: e.at.y}
	if e.fresh && !c.Measuring() {
		c.Announce("Editing " + t.cols[e.col].Title)
		e.fresh = false
	}
	in := ui.CellInput(c, &e.draft, start)
	if now := in.Focused(); !c.Measuring() {
		e.focused = now
		switch {
		case now:
			e.hadFocus = true
		case e.hadFocus && !start.Fresh:
			// The focus went elsewhere: what was typed is written, or, when
			// Validate refuses it, the edit stays open with its error and the
			// focus stays where the user put it. Once per loss.
			e.hadFocus = false
			Post(func() { _ = t.endEdit(e) })
		}
	}
	if e.err != "" {
		in.Error(e.err)
	}
	in.OnShortcut(0, ui.KeyEscape, func() { Post(t.CancelEdit) })
	if in.Submitted() {
		Post(func() { _ = t.EndEdit() })
	}
	if e.err != "" && e.announced != e.err && !c.Measuring() {
		e.announced = e.err
		c.Announce(e.err)
	}
}

// endedEdit is an edit that was written, for OnCellEndEdit.
type endedEdit[K comparable] struct {
	key    K
	column string
}

// reported is a panic of a column function, for OnPanic.
type reported struct {
	v     any
	stack []byte
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
// windows to redraw are for the caller to invalidate once it released mu.
func (t *DataGridViewWidget[T, K]) commitEditLocked(only *editState[K]) ([]*mygo.Window, error) {
	e := t.ed().edit
	if e == nil || only != nil && e != only {
		return nil, nil
	}
	if e.draft == e.orig { // nothing typed: nothing written
		t.closeEdit()
		return nil, nil
	}
	i, ok := t.src.index[e.key]
	if !ok { // the row went
		t.closeEdit()
		return nil, nil
	}
	r, cl := &t.src.rows[i], &t.cols[e.col]
	if cl.editor().Validate != nil {
		var refusal error
		if v, stack, panicked := runColumnFn(func() { refusal = cl.editor().Validate(r, e.draft) }); panicked {
			t.ed().panics = append(t.ed().panics, reported{v, stack})
			t.closeEdit()
			return nil, nil
		}
		if refusal != nil {
			e.err = refusal.Error()
			t.ed().editGen++
			e.gen = t.ed().editGen
			return nil, refusal
		}
	}
	var wins []*mygo.Window
	if v, stack, panicked := runColumnFn(func() {
		wins, _ = t.src.updateLocked(e.key, func(row *T) { cl.editor().Set(row, e.draft) })
	}); panicked {
		t.ed().panics = append(t.ed().panics, reported{v, stack})
		t.closeEdit()
		return nil, nil
	}
	t.ed().ended = append(t.ed().ended, endedEdit[K]{e.key, cl.Title})
	t.closeEdit()
	return wins, nil
}

// flushEdit delivers what commits collected under the lock: the edits written
// to OnCellEndEdit, the panics to OnPanic, both on the event goroutine.
func (t *DataGridViewWidget[T, K]) flushEdit() {
	mu.Lock()
	ended, panics, fn := t.ed().ended, t.ed().panics, t.ed().onEndEdit
	t.ed().ended, t.ed().panics = nil, nil
	mu.Unlock()
	for _, p := range panics {
		Post(func() { report(p.v, p.stack) })
	}
	if fn == nil {
		return
	}
	for _, e := range ended {
		Post(func() { fn(e.key, e.column) })
	}
}

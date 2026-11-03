package widget

// CellEditor says how the cells of a column are edited in place; Column.Editable
// gives it to the column. Its functions run under the package's lock, as the
// column's Compare does: they read the row and call no widget method and no
// RowSource method (UpdateRow, AddRows, Row, ...), which would wait for the lock
// they run under and hang. A panic in one is recovered and reported with
// OnPanic, and the edit is cancelled.
type CellEditor[T any] struct {
	// Set stores text, the user's, in the row. It parses what the column needs
	// and changes the row in place but not its key. It must not leave the row
	// half written: when it can fail after it changed a field, build the new
	// value on a copy and assign the copy (n := *row; ...; *row = n). Validate
	// runs before it. Required.
	Set func(row *T, text string)
	// Validate, when set, is asked before Set whether text may be written; a
	// non-nil error refuses it: the editor stays open and shows the error's
	// message. It reads the row and changes nothing.
	Validate func(row *T, text string) error
	// CanEdit, when set, tells whether the cell of this row may be edited
	// (nil: it may).
	CanEdit func(*T) bool
	// EditText is the text the editor starts with; nil starts with Text.
	// For a column whose Text is decorated (a unit, a badge).
	EditText func(*T) string
}

// enabler is a table that opts in to editing.
type enabler interface{ enableEdit() }

// Editable returns the column with its cells editable in place: F2, Enter or a
// double click on a cell of the current row edit it (SetEditMode changes
// that), Enter writes, Escape cancels. A column without Editable is read-only.
//
// It is also what opts a table in to editing: the code of editing is linked
// only into programs that call Editable (or another editing method of the
// table), through the function this stores, so a table that never edits pays
// nothing for it.
//
//	widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }}.
//		Editable(widget.CellEditor[Account]{Set: func(a *Account, s string) { a.Name = s }})
func (c Column[T]) Editable(e CellEditor[T]) Column[T] {
	if e.Set == nil {
		panic("widget: Column.Editable needs a Set")
	}
	c.edit, c.enable = &e, enableEditing
	return c
}

// editor returns the column's CellEditor, nil for a column that is not Editable.
func (c *Column[T]) editor() *CellEditor[T] {
	e, _ := c.edit.(*CellEditor[T])
	return e
}

// enableEditing is the Column's enable: it turns editing on in the table the
// column belongs to. It is a function of its own, referred to only by
// Editable, so that the linker keeps the table's enableEdit method only for
// programs that call Editable.
func enableEditing(table any) { table.(enabler).enableEdit() }

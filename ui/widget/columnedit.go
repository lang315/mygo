package widget

// CellEditor says how the cells of a column are edited in place; the table's
// EditColumn gives it to a column. Its functions run under the package's lock, as the
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

// editor returns the column's CellEditor, nil for a column that is not editable.
func (c *Column[T]) editor() *CellEditor[T] {
	e, _ := c.edit.(*CellEditor[T])
	return e
}

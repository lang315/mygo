package widget

import "github.com/egoist/mygo/ui"

// CellFormat is how a grid cell is styled by its row's data.
type CellFormat struct {
	Status Status   // the status family's Subtle fill and Text color; StatusNone for none
	Fill   ui.Color // overrides Status's fill; zero keeps it
	Text   ui.Color // overrides Status's text; zero keeps it
	Bold   bool
}

// SetRowFormat sets one format for every cell of a row; a column's Format wins
// over it field by field. nil removes it.
func (t *DataGridViewWidget[T, K]) SetRowFormat(f func(*T) CellFormat) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.rowFormat = f
		if f != nil {
			t.enableStatus()
		}
		t.look++ // the rows kept build again
	})
	return t
}

// SetEmptyState draws e over the body while the grid shows no row (none, or
// none its search keeps); the headers stay. nil removes it.
func (t *DataGridViewWidget[T, K]) SetEmptyState(e *EmptyStateWidget) *DataGridViewWidget[T, K] {
	t.change(func() { t.empty = e })
	return t
}

// enableStatus hands the table the builder of cells with a badge or a format
// (a method value, made once). The constructor calls it for a column with
// Badge or Format, and SetRowFormat, so that a grid without them links its
// code only for the shapes of the grids that have them. Under mu, or before
// the table is shared.
func (t *DataGridViewWidget[T, K]) enableStatus() {
	if t.statusHook == nil {
		t.statusHook = t.statusCell
	}
}

// statusCell is the statusHook: it builds the cell of the line row in column
// ci, -1 for the index column, when the cell has a badge, and reports
// whether it did. The column's functions run here, as the cell is built, and
// before anything else is: one that panics leaves nothing half done. Under
// mu.
func (t *DataGridViewWidget[T, K]) statusCell(c *ui.Context, row, ci int) bool {
	if ci < 0 {
		return false
	}
	cl, r := &t.cols[ci], t.rowAt(row)
	s := StatusNone
	if cl.Badge != nil {
		s = cl.Badge(r)
	}
	if s == StatusNone {
		return false
	}
	detail := ""
	if s == StatusRunning && cl.BadgeDetail != nil {
		detail = cl.BadgeDetail(r)
	}
	if t.cellHook != nil && t.cellHook(c, ci, r) { // the cell being edited shows its editor
		return true
	}
	text := cl.Text(r)
	name := text
	if detail != "" {
		name = text + ", " + detail // built with the row, which a frame at rest keeps
	}
	badgePill(c, s, text, detail, name, t.disabled, t.textKey)
	return true
}

package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

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
// ci, -1 for the index column, when the cell has a badge or a format, and
// reports whether it did. The column's functions run here, as the cell is
// built, and before anything else is: one that panics leaves nothing half
// done. Under mu.
func (t *DataGridViewWidget[T, K]) statusCell(c *ui.Context, row, ci int) bool {
	r := t.rowAt(row)
	var cl *Column[T]
	if ci >= 0 {
		cl = &t.cols[ci]
	}
	var f CellFormat
	if t.rowFormat != nil {
		f = t.rowFormat(r)
	}
	if cl != nil && cl.Format != nil {
		f = cl.Format(r).over(f)
	}
	s := StatusNone
	if cl != nil && cl.Badge != nil {
		s = cl.Badge(r)
	}
	if f == (CellFormat{}) && s == StatusNone {
		return false
	}
	// The selection's fill and text win over every format, as over
	// Column.Color. The box stays, so that the cell's content keeps its
	// place among the elements: an editor in it keeps the focus as the edit
	// selects its row.
	if c.RowSelected() {
		f = CellFormat{}
	}
	detail := ""
	if s == StatusRunning && cl.BadgeDetail != nil {
		detail = cl.BadgeDetail(r)
	}
	align := End // the index column's
	if cl != nil {
		align = cl.Align
	}
	formatBox(c, f, align, t.rowHeight > 0).Children(func() {
		switch {
		case cl == nil:
			t.indexCell(c, row)
		case s == StatusNone:
			t.cellContent(c, ci, r, f.Text == (ui.Color{})) // Format.Text wins over Column.Color
		case t.cellHook != nil && t.cellHook(c, ci, r): // the cell being edited shows its editor
		default:
			text := cl.Text(r)
			name := text
			if detail != "" {
				name = text + ", " + detail // built with the row, which a frame at rest keeps
			}
			badgePill(c, s, text, detail, name, t.disabled, t.textKey)
		}
	})
	return true
}

// over returns f with the fields it leaves zero taken from under: a column's
// format over its row's.
func (f CellFormat) over(under CellFormat) CellFormat {
	if f.Status == StatusNone {
		f.Status = under.Status
	}
	if f.Fill == (ui.Color{}) {
		f.Fill = under.Fill
	}
	if f.Text == (ui.Color{}) {
		f.Text = under.Text
	}
	f.Bold = f.Bold || under.Bold
	return f
}

// formatBox makes the box of a cell's content, styled by f, which covers the
// cell's whole box inside its border: negative margins as wide as the cell's
// padding (ui's table pads a cell Space(2.5) at the sides and Space(1.5)
// above and below, none with a row height), which it pads back. fixedRows is
// SetRowHeight's. Under mu.
func formatBox(c *ui.Context, f CellFormat, align Align, fixedRows bool) ui.Element {
	th := c.Theme()
	padX, padY := th.Space(2.5), th.Space(1.5)
	if fixedRows || th.Metrics.RowHeight > 0 {
		padY = 0
	}
	box := ui.Row(c).AlignItems(ui.Center).Grow(1).AlignSelf(ui.Stretch).Margin(-padY, -padX).Padding(padY, padX)
	switch align {
	case Center:
		box.Justify(ui.Center)
	case End:
		box.Justify(ui.End)
	}
	var fill, text ui.Color
	bold := f.Bold
	if f.Status != StatusNone {
		fill, text, _, _ = statusColors(c, f.Status)
		if tk := skinhook.Get(th.Dark); tk != nil && tk.FlatStatus {
			// A skin of no subtle fills: the family's text, bold, on the row.
			fill, bold = ui.Color{}, true
		}
	}
	if f.Fill != (ui.Color{}) {
		fill = f.Fill
	}
	if f.Text != (ui.Color{}) {
		text = f.Text
	}
	if fill != (ui.Color{}) {
		box.Background(fill)
	}
	if text != (ui.Color{}) {
		box.TextColor(text)
	}
	if bold {
		box.FontWeight(700)
	}
	return box
}

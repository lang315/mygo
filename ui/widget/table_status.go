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
	t.change(func() {
		t.withEmpty = nil
		if e != nil {
			// The table goes into a box of its own, which holds the empty state
			// over it, rows or not: a box that came and went with the rows
			// would move the table among the elements and lose its state.
			t.withEmpty = func(c *ui.Context) {
				ui.Column(c).Grow(1).Children(func() {
					tb := t.table(c).Grow(1)
					emptyOver(c, e, t.view.Len() == 0, bodyTop(c, tb)+t.headExtra())
				})
			}
		}
	})
	return t
}

// SetEmptyState draws e over the list while it shows no item. nil removes it.
func (l *ListBoxWidget[T, K]) SetEmptyState(e *EmptyStateWidget) *ListBoxWidget[T, K] {
	l.change(func() {
		// The list builds in a box of its own from now on, items or not, and
		// after nil too: a box that came and went would move the list among
		// the elements and lose its state.
		l.overlay = func(*ui.Context) {}
		if e != nil {
			l.overlay = func(c *ui.Context) {
				// Clipped: a state taller than the list stays in its box.
				emptyOver(c, e, len(l.frame) == 0, 0).Clip()
			}
		}
	})
	return l
}

// SetEmptyState draws e over the list while it shows no row; in the details
// view the headers stay. nil removes it.
func (lv *ListViewWidget[T, K]) SetEmptyState(e *EmptyStateWidget) *ListViewWidget[T, K] {
	lv.t.SetEmptyState(e) // Details shows the table's, under its headers
	lv.change(func() {
		// The icon grid's box stays after nil, as a ListBox's; Details' is the
		// table's, which goes.
		lv.overlay = func(*ui.Context) {}
		if e != nil {
			lv.overlay = func(c *ui.Context) {
				emptyOver(c, e, lv.t.view.Len() == 0, 0).Clip()
			}
		}
	})
	return lv
}

// emptyOver builds e over what the box being built holds, top DIPs from its
// top, when shows, and returns its element; else it keeps e as render keeps a
// hidden control, and returns none, whose methods do nothing. Under mu, after
// what it covers is built.
func emptyOver(c *ui.Context, e *EmptyStateWidget, shows bool, top float32) ui.Element {
	if !shows {
		keeping.hold(e.base())
		return ui.Element{}
	}
	outer := axis
	axis = vertical
	el := render(c, e)
	if !e.hidden {
		el.Absolute().Top(top).Left(0).Right(0).Bottom(0)
	}
	axis = outer
	return el
}

// bodyTop is the top of the body of the table tb in its box: under its
// border, its header and the line under the header.
func bodyTop(c *ui.Context, tb ui.Element) float32 {
	th := c.Theme()
	head := th.Metrics.HeaderHeight // ui's table: the header's height, then a line of 1
	if head <= 0 {
		head = th.Metrics.RowHeight
	}
	if head <= 0 {
		head = th.Space(8)
	}
	return tb.BorderWidths()[0] + head + 1
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
	t.boxes++
	detail := ""
	if s == StatusRunning && cl.BadgeDetail != nil {
		detail = cl.BadgeDetail(r)
	}
	align := End // the index column's
	if cl != nil {
		align = cl.Align
	}
	formatBox(c, f, align).Children(func() {
		switch {
		case cl == nil:
			t.indexCell(c, row)
		case s == StatusNone:
			t.cellContent(c, ci, r, f.Text == (ui.Color{})) // Format.Text wins over Column.Color
		case t.cellHook == nil || !t.cellHook(c, ci, r): // the cell being edited shows its editor
			badgeCell(c, s, cl.Text(r), detail, t.disabled, t.textKey)
		}
	})
	return true
}

// badgeCell builds a cell's badge of status s, read as its text, then its
// detail. Under mu.
func badgeCell(c *ui.Context, s Status, text, detail string, disabled bool, key any) {
	name := text
	if detail != "" {
		name = text + ", " + detail // built with the row, which a frame at rest keeps
	}
	badgePill(c, s, text, detail, name, disabled, key)
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
// above and below), which it pads back. A row height (the theme's or
// SetRowHeight's) takes the cell's padding above and below away: the box then
// overhangs the cell, which clips it, and its content stays centered. Under
// mu.
func formatBox(c *ui.Context, f CellFormat, align Align) ui.Element {
	// The selection's fill and text win over every format, as over
	// Column.Color. The box stays, so that the cell's content keeps its
	// place among the elements: an editor in it keeps the focus as the edit
	// selects its row.
	if c.RowSelected() {
		f = CellFormat{}
	}
	th := c.Theme()
	padX, padY := th.Space(2.5), th.Space(1.5)
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
		if skinhook.Get(th.Dark) != nil && skinhook.FlatStatus() {
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

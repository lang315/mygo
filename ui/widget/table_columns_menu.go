package widget

import "github.com/egoist/mygo/ui"

// BestFit fits each column titled in titles to its header and its cells in
// view, as a double click on the edge of its header does; with no titles,
// every column shown. It applies at the next frame, and OnColumnsChange
// gets the new layout. Titles of no column, or of hidden ones, are ignored.
func (t *DataGridViewWidget[T, K]) BestFit(titles ...string) *DataGridViewWidget[T, K] {
	t.change(func() {
		if len(titles) > 0 {
			t.fits = append(t.fits, titles...)
			return
		}
		for _, cl := range t.cols {
			t.fits = append(t.fits, cl.Title) // ui fits those it shows
		}
	})
	return t
}

// SetColumnChooser gives the header a context menu, or takes it away, as by
// default. The menu lists every column, checked when it shows, which
// choosing it hides or shows as SetColumnVisible does; the last column
// shown cannot be hidden. "Best fit" fits the column right-clicked and
// "Best fit (all columns)" every one, as BestFit does; "Clear sorting",
// there while the rows are sorted, does what ClearSort does.
func (t *DataGridViewWidget[T, K]) SetColumnChooser(on bool) *DataGridViewWidget[T, K] {
	t.change(func() { t.chooser = on })
	return t
}

// SetCellHints, when on, shows the full Text of a cell cut off by its
// column as a tooltip, or not, as by default. A column's Hint wins over it.
func (t *DataGridViewWidget[T, K]) SetCellHints(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.hints = on
		t.look++ // the rows kept are built again, with their tips
		if t.headW == nil {
			t.headW = make([]float32, len(t.cols)) // the columns never change in number
		}
	})
	return t
}

// handColumnsOver gives ui the header's menu and the columns BestFit asked
// for, before the table builds. Under mu.
func (t *DataGridViewWidget[T, K]) handColumnsOver(c *ui.Context) {
	s := &t.list
	s.HeaderMenu = nil
	if t.chooser {
		if t.chooserFn == nil {
			t.chooserFn = t.columnsMenu // a method value made in every frame allocates
		}
		s.HeaderMenu = t.chooserFn
	}
	if len(t.fits) > 0 && !c.Measuring() { // the request is for the real window
		s.FitColumns(t.fits...)
		t.fits = t.fits[:0]
	}
}

// columnsMenu is ui's HeaderMenu with the column chooser: col is the index
// in t.columns of the column right-clicked, -1 past the columns. The choices
// are posted, and made by the setters.
func (t *DataGridViewWidget[T, K]) columnsMenu(m *ui.Menu, col int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	shown := 0
	for _, cl := range t.cols {
		if !cl.Hidden {
			shown++
		}
	}
	for _, cl := range t.cols {
		title, on := cl.Title, !cl.Hidden
		it := m.Item(title).ID("column:" + title).Checked(on)
		if it.Disabled(on && shown == 1).Chosen() {
			Post(func() { t.SetColumnVisible(title, !on) })
		}
	}
	m.Separator()
	fit := "" // the column right-clicked, not the index column
	if t.numbered {
		col--
	}
	if col >= 0 { // -1: past the columns, or the index column
		fit = t.cols[t.visCols[col]].Title
	}
	if m.Item("Best fit").ID("best fit").Disabled(fit == "").Chosen() {
		Post(func() { t.BestFit(fit) })
	}
	if m.Item("Best fit (all columns)").ID("best fit all").Chosen() {
		Post(func() { t.BestFit() })
	}
	if len(t.sort.value) > 0 {
		m.Separator()
		if m.Item("Clear sorting").ID("clear sorting").Chosen() {
			Post(func() { t.ClearSort() })
		}
	}
}

// noteHeader keeps the width of the header of column col, an index in
// t.columns, as the last frame laid it out, for the hints of a column sharing
// the room left: the cell under the pointer, which reads it, is built in each
// frame. Under mu.
func (t *DataGridViewWidget[T, K]) noteHeader(cell ui.Element, col int) {
	if t.numbered {
		col--
	}
	if col >= 0 { // not the index column
		t.headW[t.visCols[col]] = cell.Bounds().W
	}
}

// hintedCell builds the cell of r in column ci (an index in t.cols) of a
// column with a Hint, or of a grid with SetCellHints, in a box covering the
// whole cell, as a format's does, which takes its tooltip. Under mu.
func (t *DataGridViewWidget[T, K]) hintedCell(c *ui.Context, ci int, r *T) {
	box := formatBox(c, CellFormat{}, t.cols[ci].Align).Children(func() { t.cellContent(c, ci, r, true) })
	t.hint(c, box, ci, r, true)
}

// hint gives box, the box covering the cell of r in column ci (an index in
// t.cols), its tooltip: the column's Hint, or with full and SetCellHints its
// full Text when it is wider than the column leaves it. Only the cell under
// the pointer gets it: ui never keeps a row holding a tooltip, while it keeps
// one that reads the pointer as long as the pointer is elsewhere, so a frame
// at rest builds no row for a tip.
func (t *DataGridViewWidget[T, K]) hint(c *ui.Context, box ui.Element, ci int, r *T, full bool) {
	if !box.Hovered() {
		if t.tipAt == box.ID() {
			t.tipAt = 0
		}
		return
	}
	if s := t.hintText(c, ci, r, full); s != "" {
		box.Tooltip(s)
		if t.tipAt != box.ID() {
			// ui times the rest of the pointer from the frame that sees the
			// tip: ask for it, as the pointer may not move again.
			t.tipAt = box.ID()
			c.After(0)
		}
	}
}

// hintText returns the tooltip of the cell of r in column ci: its Hint, or
// with full and SetCellHints its full Text when cut off; "" for none.
func (t *DataGridViewWidget[T, K]) hintText(c *ui.Context, ci int, r *T, full bool) string {
	cl := &t.cols[ci]
	if cl.Hint != nil {
		if s := cl.Hint(r); s != "" {
			return s
		}
	}
	if !full || !t.hints {
		return ""
	}
	var lead float32 // an image before the text, and its gap
	if cl.Image != nil {
		lead = iconSide(c) + 4
	}
	w, ok := t.list.Columns.Widths[cl.Title]
	if !ok {
		w = cl.Width
	}
	if w > 0 {
		w = max(w, cl.MinWidth, 16) // as ui's clampWidth
	} else {
		w = t.headW[ci] // sharing the room left: as wide as its header was
	}
	th := c.Theme()
	w -= 2*th.Space(2.5) + lead // ui's cell padding
	if t.grid {
		w-- // the line at the cell's end
	}
	text := cl.Text(r)
	if tw, _ := c.MeasureText(0, ui.Span{Text: text}); tw > w+0.5 {
		return text
	}
	return ""
}

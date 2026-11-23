package widget

import (
	"iter"

	"github.com/egoist/mygo/ui"
)

// enableSummaries gives the grid its row of summaries: a column has a
// Summary. The texts are worked out again when the rows shown change (the
// view's version, or the source's change), not in a frame at rest. Before
// the table is shared.
func (t *DataGridViewWidget[T, K]) enableSummaries() {
	f := &gridFooter{}
	t.footer = f
	var (
		known        bool
		version, seq uint64
		texts        []string // each column's of t.cols
	)
	// The rows the view shows, in its order: made once.
	rows := func(yield func(*T) bool) {
		for i := range t.view.Len() {
			if !yield(&t.src.rows[t.view.Row(i)]) {
				return
			}
		}
	}
	// Under mu, after the table built.
	t.summarize = func(c *ui.Context) {
		if v, s := t.view.Version(), t.src.seq; !known || v != version || s != seq {
			known, version, seq = true, v, s
			texts = texts[:0]
			for i := range t.cols {
				texts = append(texts, summaryOf(t.cols[i].Summary, rows))
			}
		}
		// The header's cells are those of t.columns: the index column first
		// when numbered, then the columns shown.
		f.cells = f.cells[:0]
		for _, h := range f.heads {
			fc := footCell{head: h.cell}
			if ci := h.col; !t.numbered || ci > 0 {
				if t.numbered {
					ci--
				}
				if ci < len(t.visCols) {
					cl := &t.cols[t.visCols[ci]]
					fc.text, fc.align = texts[t.visCols[ci]], cl.Align.ui()
				}
			}
			f.cells = append(f.cells, fc)
		}
		f.build(c, t.headSize, t.headLook, t.disabled, t.textKey)
	}
}

// summaryOf runs a column's Summary over rows; a panic is reported as a
// handler's, on the event goroutine, and leaves the text empty. Under mu.
func summaryOf[T any](f func(iter.Seq[*T]) string, rows iter.Seq[*T]) (text string) {
	if f == nil {
		return ""
	}
	if v, stack, panicked := runColumnFn(func() { text = f(rows) }); panicked {
		Post(func() { report(v, stack) })
		return ""
	}
	return text
}

// headCell is a cell of the header as the table built it: col is its column,
// an index into the columns ui is given.
type headCell struct {
	col  int
	cell ui.Element
}

// footCell is a cell of the row of summaries: under head, its text aligned
// as its column.
type footCell struct {
	head  ui.Element
	text  string
	align ui.Align
}

// gridFooter is the state of a DataGridView's row of summaries. Its cells
// are placed where the header's were laid out in the last frame; a paint
// that finds them laid out elsewhere asks for one more frame, so a resize, a
// move or a scroll settles at once.
type gridFooter struct {
	heads  []headCell // the header's cells of this build, in order (headerCell)
	cells  []footCell
	placed []ui.Rect // the bounds of the header's cells the cells were placed by
	at     ui.Rect   // the row's bounds, which they were placed in
	row    ui.Element
	c      *ui.Context
	check  func(*ui.Painter, ui.Rect) // checkPlaces, made once
	builds int                        // test hook: how many builds built the row
}

// build builds the row under the table: the header's height, its fill (the
// skin's GridHeader look on each cell, else the table's), each cell of its
// header's place and width. size is the skin's header text size, 0 for
// none. Under mu, while building.
func (f *gridFooter) build(c *ui.Context, size float32, look *Style, disabled bool, key any) {
	f.builds++
	th := c.Theme()
	h := th.Metrics.HeaderHeight
	if h <= 0 {
		h = th.Metrics.RowHeight
	}
	if h <= 0 {
		h = th.Space(8)
	}
	row := ui.Row(c).Height(h + 2).Shrink(0).Clip() // its border takes 1 DIP above and below
	frameList(c, row)
	f.row, f.c, f.at = row, c, row.Bounds()
	if f.check == nil {
		f.check = f.checkPlaces
	}
	if !c.Measuring() {
		row.Draw(f.check)
	}
	f.placed = f.placed[:0]
	sp := th.Space(2.5)
	row.Children(func() {
		for i := range f.cells {
			fc := &f.cells[i]
			b := fc.head.Bounds()
			f.placed = append(f.placed, b)
			if b.W <= 0 { // not laid out yet: the paint asks for the frame that places it
				continue
			}
			// Left is from inside the row's border, 1 DIP in.
			cell := ui.Row(c).Absolute().Top(0).Left(b.X-f.at.X-1).Width(b.W).Height(h).
				Padding(0, sp).AlignItems(ui.Center).Justify(fc.align).Clip().Role(ui.RoleCell)
			if size > 0 {
				cell.FontSize(size)
			}
			if look != nil {
				st := look.effective(th.Dark, false, false, false, false, disabled)
				st.apply(c, cell, key)
			}
			if fc.text != "" {
				cell.Children(func() { ui.Text(c, fc.text).SingleLine().FontWeight(600) })
			}
		}
	})
}

// checkPlaces is the row's paint: it asks for one more frame when the
// header's cells, or the row, are laid out elsewhere than the cells were
// placed by, as after a column was resized, moved or scrolled.
func (f *gridFooter) checkPlaces(*ui.Painter, ui.Rect) {
	moved := f.row.Bounds().X != f.at.X || len(f.placed) != len(f.cells)
	for i := 0; !moved && i < len(f.cells); i++ {
		b, p := f.cells[i].head.Bounds(), f.placed[i]
		moved = b.X != p.X || b.W != p.W
	}
	if moved {
		f.c.Invalidate()
	}
}

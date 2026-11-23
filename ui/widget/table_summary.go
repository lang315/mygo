package widget

import (
	"github.com/egoist/mygo/ui"
)

// enableSummaries gives the grid its row of summaries: a column has a
// Summary. Every grid of an app links this, as Summary is a field, once for
// each shape of its rows: so it is only the adapter that runs a column's
// Summary over the rows shown; the row's state and work are gridFooter's,
// which reads the grid's through pointers. Before the table is shared.
func (t *DataGridViewWidget[T, K]) enableSummaries() {
	f := &gridFooter{n: len(t.cols), version: &t.list.Version, seq: &t.src.seq, columns: &t.columns,
		vis: &t.visCols, numbered: &t.numbered, size: &t.headSize, look: &t.headLook, disabled: &t.disabled, key: &t.textKey}
	f.summary = func(i int) string {
		s := t.cols[i].Summary
		if s == nil {
			return ""
		}
		return s(func(yield func(*T) bool) { // the rows shown, in their order
			for p := range t.view.Len() {
				if !yield(&t.src.rows[t.view.Row(p)]) {
					return
				}
			}
		})
	}
	t.footer, t.summarize = f, f.frame
}

// headCell is a cell of the header as the table built it: col is its column,
// an index into the columns ui is given.
type headCell struct {
	col  int
	cell ui.Element
}

// gridFooter is the state of a DataGridView's row of summaries. Its cells
// are placed where the header's were laid out in the last frame; a paint
// that finds them laid out elsewhere asks for one more frame, so a resize, a
// move or a scroll settles at once.
type gridFooter struct {
	heads []headCell // the header's cells of this build, in order (headerCell)
	// summary runs the Summary of the grid's column i of n, "" for none.
	summary func(i int) string
	n       int
	// The grid's: the view's version as given to ui (ListState.Version), the
	// source's change, the columns ui is given, the indexes of those shown,
	// whether the index column shows, the header's text size, the skin's
	// GridHeader look, disabled, and the text key.
	version, seq *uint64
	columns      *[]ui.TableColumn
	vis          *[]int
	numbered     *bool
	size         *float32
	look         **Style
	disabled     *bool
	key          *any
	// texts are the columns' Summary texts, as of known's version and seq.
	texts        []string
	known        bool
	atVer, atSeq uint64
	placed       []ui.Rect // the bounds of the header's cells the cells were placed by
	at           ui.Rect   // the row's bounds, which they were placed in
	row          ui.Element
	c            *ui.Context
	check        func(*ui.Painter, ui.Rect) // checkPlaces, made once
	builds       int                        // test hook: how many builds built the row
}

// frame is the grid's summarize: it works the texts out again when the rows
// shown changed, and builds the row. A Summary's panic is reported as a
// handler's, on the event goroutine, and leaves its text empty. Under mu,
// after the table built.
func (f *gridFooter) frame(c *ui.Context) {
	if v, s := *f.version, *f.seq; !f.known || v != f.atVer || s != f.atSeq {
		f.known, f.atVer, f.atSeq = true, v, s
		f.texts = f.texts[:0]
		for i := range f.n {
			text := ""
			if v, stack, panicked := runColumnFn(func() { text = f.summary(i) }); panicked {
				Post(func() { report(v, stack) })
			}
			f.texts = append(f.texts, text)
		}
	}
	f.build(c)
}

// build builds the row under the table: the header's height, its fill (the
// skin's GridHeader look on each cell, else the table's), a cell under each
// of the header's, as wide, the index column's empty. The cells follow one
// another, with a gap where the header has one (its check boxes); a cell the
// header scrolled past the row's start keeps its part in view. Under mu,
// while building.
func (f *gridFooter) build(c *ui.Context) {
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
	row.Children(func() {
		x := float32(0) // where the next cell goes, from inside the row's border
		for _, hc := range f.heads {
			b := hc.cell.Bounds()
			f.placed = append(f.placed, b)
			left := b.X - f.at.X - 1
			from := max(left, x)
			if b.W <= 0 || left+b.W <= from { // not laid out yet (the paint asks for the frame that places it), or scrolled past
				continue
			}
			if from > x {
				ui.Row(c).Width(from - x).Shrink(0)
			}
			f.cell(c, hc.col, left+b.W-from, h)
			x = left + b.W
		}
	})
}

// cell builds the cell under the header's column col, w wide and h high.
func (f *gridFooter) cell(c *ui.Context, col int, w, h float32) {
	text := ""
	if i := col; !*f.numbered || i > 0 {
		if *f.numbered {
			i--
		}
		if vis := *f.vis; i < len(vis) && vis[i] < len(f.texts) {
			text = f.texts[vis[i]]
		}
	}
	th := c.Theme()
	cell := ui.Row(c).Width(w).Height(h).Shrink(0).Padding(0, th.Space(2.5)).AlignItems(ui.Center).
		Justify((*f.columns)[col].Align).Clip().Role(ui.RoleCell)
	if *f.size > 0 {
		cell.FontSize(*f.size)
	}
	if look := *f.look; look != nil {
		st := look.effective(th.Dark, false, false, false, false, *f.disabled)
		st.apply(c, cell, *f.key)
	}
	if text != "" {
		cell.Children(func() { ui.Text(c, text).SingleLine().FontWeight(600) })
	}
}

// checkPlaces is the row's paint: it asks for one more frame when the
// header's cells, or the row, are laid out elsewhere than the cells were
// placed by, as after a column was resized, moved or scrolled.
func (f *gridFooter) checkPlaces(*ui.Painter, ui.Rect) {
	moved := f.row.Bounds().X != f.at.X || len(f.placed) != len(f.heads)
	for i := 0; !moved && i < len(f.heads); i++ {
		b, p := f.heads[i].cell.Bounds(), f.placed[i]
		moved = b.X != p.X || b.W != p.W
	}
	if moved {
		f.c.Invalidate()
	}
}

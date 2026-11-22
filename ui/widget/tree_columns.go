package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// SetColumns shows the tree in columns under a header (DevExpress's
// TreeList): the first, headed treeTitle, holds each node's arrow, check
// box, image and text; then one column of cols each, showing its Text. Of
// a Column it reads Title, Text, Width, MinWidth, Align and Hidden. The
// user resizes and moves the columns. SetColumns("") shows the tree
// without columns again.
func (t *TreeViewWidget[T, K]) SetColumns(treeTitle string, cols ...Column[T]) *TreeViewWidget[T, K] {
	t.change(func() {
		t.look++ // every row builds again, in its new columns or without any
		if treeTitle == "" && len(cols) == 0 {
			t.inColumns = nil
			return
		}
		// New slices, never changed: rows ui builds late hold them.
		columns := []ui.TableColumn{{Title: treeTitle, MinWidth: 120}}
		var shown []Column[T]
		for _, cl := range cols {
			if !cl.Hidden {
				shown = append(shown, cl)
				columns = append(columns, ui.TableColumn{Title: cl.Title, Width: cl.Width, MinWidth: cl.MinWidth, Align: cl.Align.ui()})
			}
		}
		tc := &treeColumns[T, K]{t: t, columns: columns, cols: shown}
		tc.header, tc.cellFn, tc.kids, tc.label, tc.skip = tc.headerCell, tc.cell, t.kidsOf, t.labelAt, t.skip
		t.inColumns = tc.outline
	})
	return t
}

// treeColumns is what a TreeView in columns builds with: its columns, and
// the functions ui calls, made once by SetColumns so that a frame makes none.
type treeColumns[T any, K comparable] struct {
	t       *TreeViewWidget[T, K]
	columns []ui.TableColumn
	cols    []Column[T] // the columns after the tree's, those shown
	header  func(cell ui.Element, col int)
	cellFn  func(it treeItem[K], col int)
	kids    func(treeItem[K]) []treeItem[K]
	label   func(row int) string
	skip    func(row int) bool
	// c is the context of the build, which the cells and the header use;
	// size and look are the skin's header text size (0 for none) and
	// GridHeader look, as DataGridView's header takes them.
	c    *ui.Context
	size float32
	look *Style
}

// outline is the TreeView's inColumns: ui's outline in columns, on the
// TreeView's state. Under mu, while the tree builds. Not named build: the
// linker keeps a method of Control's name and type wherever the type is.
func (tc *treeColumns[T, K]) outline(c *ui.Context) ui.Element {
	tc.c, tc.size = c, 0
	if tk, scale := skinOf(c); tk != nil {
		tc.size = tk.Label.Size * scale
	}
	tc.look = lookOf(skinhook.KindGridHeader)
	s := &tc.t.state
	s.List.HeaderCell, s.List.Label, s.List.Skip = tc.header, tc.label, tc.skip
	return ui.OutlineTable(c, s, tc.columns, tc.t.roots, tc.kids, tc.cellFn)
}

// headerCell is ui's HeaderCell, as DataGridView's: a skin's Label sizes the
// title, and its GridHeader look styles the cell.
func (tc *treeColumns[T, K]) headerCell(cell ui.Element, _ int) {
	if tc.size > 0 {
		cell.FontSize(tc.size)
	}
	if look := tc.look; look != nil {
		t := tc.t
		st := look.effective(tc.c.Theme().Dark, cell.Hovered(), false, false, cell.Pressed(), t.disabled)
		st.apply(tc.c, cell, t.textKey)
	}
}

// cell builds the cell of it in column col: the tree's row in the first,
// spaced as without columns, else the column's Text, empty on the Loading
// row. While the tree builds, or when ui lays out rows it did not build then.
func (tc *treeColumns[T, K]) cell(it treeItem[K], col int) {
	c, t := tc.c, tc.t
	if col == 0 {
		sp := c.Theme().Space(1)
		ui.Row(c).Gap(sp).MarginStart(sp).AlignItems(ui.Center).Children(func() { t.row(c, it) })
		return
	}
	if lockOutsideView() {
		defer mu.Unlock()
	}
	n := t.nodes[it.k]
	if it.wait || n == nil {
		return
	}
	ui.Text(c, tc.cols[col-1].Text(&n.data)).SingleLine()
}

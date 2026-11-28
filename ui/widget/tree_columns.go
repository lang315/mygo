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
		// New slices, never changed: rows ui builds late hold them. Only the
		// Text functions are kept, and cols read in place: a copy of a Column
		// would keep Column's type in every app with a TreeView, through the
		// tree's dictionary.
		columns := []ui.TableColumn{{Title: treeTitle, MinWidth: 120}}
		var texts []func(*T) string
		for i := range cols {
			if cl := &cols[i]; !cl.Hidden {
				texts = append(texts, cl.Text)
				columns = append(columns, ui.TableColumn{Title: cl.Title, Width: cl.Width, MinWidth: cl.MinWidth, Align: cl.Align.ui()})
			}
		}
		t.inColumns = treeInColumns(t, columns, texts)
	})
	return t
}

// treeInColumns returns the TreeView's inColumns: ui's outline in columns,
// on the TreeView's state, whose functions it makes once so that a frame
// makes none. texts are the Text of the columns after the tree's.
func treeInColumns[T any, K comparable](t *TreeViewWidget[T, K], columns []ui.TableColumn, texts []func(*T) string) func(*ui.Context) ui.Element {
	// c is the context of the build, which the cells and the header use;
	// size and look are the skin's header text size (0 for none) and
	// GridHeader look, as DataGridView's header takes them.
	var (
		c    *ui.Context
		size float32
		look *Style
	)
	header := func(cell ui.Element, _ int) {
		if size > 0 {
			cell.FontSize(size)
		}
		if look != nil {
			st := look.effective(c.Theme().Dark, cell.Hovered(), false, false, cell.Pressed(), t.disabled)
			st.apply(c, cell, t.textKey)
		}
	}
	// The tree's row in the first column, spaced as without columns, else
	// the column's Text, empty on the Loading row. While the tree builds, or
	// when ui lays out rows it did not build then.
	cell := func(it treeItem[K], col int) {
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
		ui.Text(c, texts[col-1](&n.data)).SingleLine()
	}
	kids, label, skip := t.kidsOf, t.labelAt, t.skip
	// Under mu, while the tree builds.
	return func(ctx *ui.Context) ui.Element {
		c, size = ctx, 0
		if tk, scale := skinOf(c); tk != nil {
			size = tk.Label.Size * scale
		}
		look = lookOf(skinhook.KindGridHeader)
		s := &t.state
		s.List.HeaderCell, s.List.Label, s.List.Skip = header, label, skip
		return ui.OutlineTable(c, s, columns, t.shownRoots(), kids, cell)
	}
}

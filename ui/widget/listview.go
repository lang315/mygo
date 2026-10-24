package widget

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// ViewMode is how a ListView shows its rows.
type ViewMode int

const (
	// ViewDetails shows the rows in columns that sort, as a DataGridView does.
	// It is the zero value: WinForms' default is large icons.
	ViewDetails ViewMode = iota
	// ViewLargeIcon shows each row as an icon over its text, in a grid.
	ViewLargeIcon
	// ViewSmallIcon shows each row as a small icon before its text, in a grid.
	ViewSmallIcon
	// ViewList shows each row as a small icon before its text, in one column.
	ViewList
)

func (m ViewMode) String() string {
	switch m {
	case ViewDetails:
		return "Details"
	case ViewLargeIcon:
		return "LargeIcon"
	case ViewSmallIcon:
		return "SmallIcon"
	case ViewList:
		return "List"
	}
	return fmt.Sprintf("ViewMode(%d)", int(m))
}

// ListViewWidget shows rows in the view the app or the user picks, as
// WinForms' ListView: in columns (ViewDetails), as large or small icons in a
// grid, or as a list. The rows, their choice and their checks are the same in
// every view, kept by key, so switching the view at run time keeps what was
// chosen and checked.
//
// Details is a DataGridView underneath (Details), and the icon views show its
// rows, in its order, over the same sets of keys.
type ListViewWidget[T any, K comparable] struct {
	common[ListViewWidget[T, K]]
	t        *DataGridViewWidget[T, K]
	mode     ViewMode
	image    func(*T) *ui.Bitmap
	iconSize float32

	onSelect func([]K)
	onCheck  func(K, bool)

	// cursorKey is the row the keys move from in the icon views; fixed says a
	// setter chose it, not ui's cursor.
	cursorKey K
	hasCursor bool
	fixed     bool

	// ui's, on the main thread:
	grid   ui.GridState
	cursor int
}

// ListView makes a list view of rows told apart by key, whose Details
// columns are cols; the first column's text is the text of a row in the
// other views. key and the columns' functions run under the package's lock.
func ListView[T any, K comparable](key func(*T) K, cols ...Column[T]) *ListViewWidget[T, K] {
	return ListViewOf(NewRowSource(key), cols...)
}

// ListViewOf makes a list view of the rows of src, which other tables may
// show too.
func ListViewOf[T any, K comparable](src *RowSource[T, K], cols ...Column[T]) *ListViewWidget[T, K] {
	lv := &ListViewWidget[T, K]{t: DataGridViewOf(src, cols...), iconSize: 48, cursor: -1}
	lv.init(lv)
	mu.Lock()
	lv.t.watch = lv.watched
	mu.Unlock()
	return lv
}

// Details returns the DataGridView underneath, for the options of its
// columns: sorting, Search, SetColumnsLayout, OnColumnsChange, BottomPanel and
// the rest. Sort and filter drive every view: the icons show the rows in the
// order of the table and leave out those its search hides. Do not add it to a
// window yourself: the list view shows it.
func (lv *ListViewWidget[T, K]) Details() *DataGridViewWidget[T, K] { return lv.t }

// SetView shows the rows in view m, keeping the choice and the checks, and
// firing no event.
func (lv *ListViewWidget[T, K]) SetView(m ViewMode) *ListViewWidget[T, K] {
	lv.change(func() { lv.mode = m })
	return lv
}

// View returns the view the rows show in.
func (lv *ListViewWidget[T, K]) View() ViewMode {
	mu.Lock()
	defer mu.Unlock()
	return lv.mode
}

// SetImage sets the icon of a row, for the icon views (large, 48 DIPs square
// by default; small, 16) and before the text of the first Details column; nil
// for none. fn runs under the package's lock and calls no widget method. Without
// it the first column's Image is used, if it has one.
func (lv *ListViewWidget[T, K]) SetImage(fn func(*T) *ui.Bitmap) *ListViewWidget[T, K] {
	lv.change(func() { lv.image = fn })
	return lv
}

// SetIconSize sets the side of the large icons in DIPs, 48 by default.
func (lv *ListViewWidget[T, K]) SetIconSize(size float32) *ListViewWidget[T, K] {
	lv.change(func() { lv.iconSize = max(size, 16) })
	return lv
}

// SetRows shows a copy of rows instead of those shown; of rows with the same
// key the last stays, where the first was.
func (lv *ListViewWidget[T, K]) SetRows(rows []T) *ListViewWidget[T, K] {
	lv.t.SetRows(rows)
	return lv
}

// AddRow shows row after the others, or instead of the row of its key.
func (lv *ListViewWidget[T, K]) AddRow(row T) *ListViewWidget[T, K] {
	lv.t.AddRow(row)
	return lv
}

// AddRows shows rows after the others, each instead of the row of its key
// if one shows.
func (lv *ListViewWidget[T, K]) AddRows(rows []T) *ListViewWidget[T, K] {
	lv.t.AddRows(rows)
	return lv
}

// UpdateRow runs fn on the row of key k, if one shows, to change it in place.
// fn runs under the package's lock: it changes the row but not its key, and
// calls no widget method.
func (lv *ListViewWidget[T, K]) UpdateRow(k K, fn func(*T)) *ListViewWidget[T, K] {
	lv.t.UpdateRow(k, fn)
	return lv
}

// DeleteRows removes the rows of keys ks, with their choice and checks and
// without an event.
func (lv *ListViewWidget[T, K]) DeleteRows(ks ...K) *ListViewWidget[T, K] {
	lv.t.DeleteRows(ks...)
	return lv
}

// Clear removes every row.
func (lv *ListViewWidget[T, K]) Clear() *ListViewWidget[T, K] {
	lv.t.Clear()
	return lv
}

// Row returns the row of key k.
func (lv *ListViewWidget[T, K]) Row(k K) (T, bool) { return lv.t.Row(k) }

// Len returns how many rows the list view has, the hidden by a search too.
func (lv *ListViewWidget[T, K]) Len() int { return lv.t.Len() }

// Selected returns the keys of the rows chosen that show, in the order shown.
func (lv *ListViewWidget[T, K]) Selected() []K { return lv.t.Selected() }

// CountSelected returns how many rows are chosen.
func (lv *ListViewWidget[T, K]) CountSelected() int { return lv.t.CountSelected() }

// SetSelected chooses the rows of keys ks instead of those chosen, firing no
// event; the last is the one the keys move from in the icon views. A key that
// is not a row is ignored.
func (lv *ListViewWidget[T, K]) SetSelected(ks ...K) *ListViewWidget[T, K] {
	t := lv.t
	lv.change(func() {
		t.catchUp()
		for _, k := range slices.Collect(t.chosen.value.All()) {
			t.chosen.set(k, false)
		}
		for _, k := range ks {
			if _, ok := t.src.index[k]; ok {
				t.chosen.set(k, true)
				lv.cursorKey, lv.hasCursor, lv.fixed = k, true, true
			}
		}
	})
	return lv
}

// CheckBoxes shows a check box on each row, in every view.
func (lv *ListViewWidget[T, K]) CheckBoxes() *ListViewWidget[T, K] {
	lv.t.CheckboxColumn()
	return lv
}

// Checked returns the keys of the rows checked that show, in the order shown.
func (lv *ListViewWidget[T, K]) Checked() []K { return lv.t.Checked() }

// IsChecked reports whether the row of key k is checked.
func (lv *ListViewWidget[T, K]) IsChecked(k K) bool { return lv.t.IsChecked(k) }

// SetChecked checks, or unchecks, the rows of keys ks, firing no event.
func (lv *ListViewWidget[T, K]) SetChecked(ks []K, on bool) *ListViewWidget[T, K] {
	lv.t.SetChecked(ks, on)
	return lv
}

// OnSelect runs fn on the event goroutine with the keys of the rows chosen,
// in the order shown, after the user changes the choice in any view
// (WinForms' SelectedIndexChanged). Setters, and switching the view, do not
// run it.
func (lv *ListViewWidget[T, K]) OnSelect(fn func(selected []K)) *ListViewWidget[T, K] {
	lv.change(func() { lv.onSelect = fn })
	return lv
}

// OnCheck runs fn on the event goroutine with the key of a row the user
// checked or unchecked, and whether it is checked (WinForms' ItemCheck).
func (lv *ListViewWidget[T, K]) OnCheck(fn func(k K, on bool)) *ListViewWidget[T, K] {
	lv.change(func() { lv.onCheck = fn })
	return lv
}

// OnActivate runs fn with the key of a row double-clicked, or chosen when
// Enter is pressed, in any view.
func (lv *ListViewWidget[T, K]) OnActivate(fn func(k K)) *ListViewWidget[T, K] {
	lv.t.OnActivate(fn)
	return lv
}

// ContextMenu shows items when rows are right-clicked, in every view; see
// DataGridView.ContextMenu.
func (lv *ListViewWidget[T, K]) ContextMenu(items ...MenuItem) *ListViewWidget[T, K] {
	lv.t.ContextMenu(items...)
	return lv
}

// ContextMenuFunc shows the items fn makes for the keys of the rows chosen,
// in every view; see DataGridView.ContextMenuFunc.
func (lv *ListViewWidget[T, K]) ContextMenuFunc(fn func(selected []K) []MenuItem) *ListViewWidget[T, K] {
	lv.t.ContextMenuFunc(fn)
	return lv
}

// CopyFunc sets what Ctrl+C (Cmd+C) and the Edit menu's Copy copy, in every
// view; see DataGridView.CopyFunc.
func (lv *ListViewWidget[T, K]) CopyFunc(fn func(selected []K) string) *ListViewWidget[T, K] {
	lv.t.CopyFunc(fn)
	return lv
}

// watched is the underlying table's watch: it posts OnSelect and OnCheck for
// what the user changed. Under mu.
func (lv *ListViewWidget[T, K]) watched(checks bool, on, off []K) {
	if checks {
		if h := lv.onCheck; h != nil {
			for _, k := range on {
				Post(func() { h(k, true) })
			}
			for _, k := range off {
				Post(func() { h(k, false) })
			}
		}
		return
	}
	if h := lv.onSelect; h != nil {
		ks := lv.t.shown(&lv.t.chosen.value)
		Post(func() { h(ks) })
	}
}

func (lv *ListViewWidget[T, K]) build(c *ui.Context) ui.Element {
	t := lv.t
	if lv.mode == ViewDetails {
		return render(c, t)
	}
	// The table is not built, but its changes redraw this window.
	keeping.hold(&t.core)
	// The row the keys move from stays the same row as the order changes.
	if lv.fixed {
		lv.fixed = false
	} else if c := lv.cursor; c >= 0 {
		if r := t.rowAt(c); r != nil {
			lv.cursorKey, lv.hasCursor = t.key(r), true
		}
	}
	t.catchUp()
	t.sync()
	t.frame, t.kept, t.gone = t.src.rows, false, false
	n := t.view.Len()

	lv.cursor = -1
	if lv.hasCursor {
		lv.cursor = t.positionOf(lv.cursorKey)
	}
	g := &lv.grid
	g.Key, g.Label, g.Selected = t.keyAt, t.labelAt, &lv.cursor
	g.Selection = t.chosen.bind()
	g.Checked = nil
	if t.checks {
		g.Checked = t.checked.bind()
	}
	g.Copy, g.ItemMenu = nil, nil
	if t.copyFn != nil {
		g.Copy = func(items []int) string { return t.copyRows(items) }
	}
	switch menu := t.menu; {
	case t.menuFunc != nil:
		g.ItemMenu = func(m *ui.Menu, items []int) { t.lateMenu(m, items) }
	case len(menu) > 0:
		g.ItemMenu = func(m *ui.Menu, _ []int) { buildMenu(m, menu) }
	}

	minWidth, height := float32(104), float32(92)
	switch lv.mode {
	case ViewSmallIcon:
		minWidth, height = 180, 30
	case ViewList:
		minWidth, height = 1e6, 30 // one column
	default:
		height = lv.iconSize + 44
		minWidth = lv.iconSize + 56
	}
	e := ui.GridView(c, g, n, minWidth, height, func(i int) { lv.item(c, i) }).Grow(1).
		OnChange(func() { t.commit(&t.chosen) }).
		OnCheckedChange(func() { t.commit(&t.checked) })
	if fn := t.onActivate; fn != nil && e.Submitted() && lv.cursor >= 0 {
		if r := t.rowAt(lv.cursor); r != nil {
			k := t.key(r)
			Post(func() { fn(k) })
		}
	}
	return e
}

// item builds the content of the item shown at i: its icon and text. While the
// view builds, or when ui lays out items it did not build then.
func (lv *ListViewWidget[T, K]) item(c *ui.Context, i int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	t := lv.t
	r := t.rowAt(i)
	if r == nil || len(t.cols) == 0 {
		ui.Text(c, "")
		return
	}
	text := t.cols[0].Text(r)
	var img *ui.Bitmap
	switch {
	case lv.image != nil:
		img = lv.image(r)
	case t.cols[0].Image != nil:
		img = t.cols[0].Image(r)
	}
	th := c.Theme()
	if lv.mode == ViewLargeIcon {
		ui.Column(c).AlignItems(ui.Center).Gap(th.Space(1)).Padding(th.Space(1)).Children(func() {
			if img != nil {
				ui.Image(c, img).Size(lv.iconSize, lv.iconSize)
			} else {
				ui.Box(c).Size(lv.iconSize, lv.iconSize)
			}
			ui.Text(c, text).MaxLines(2).TextAlign(ui.Center)
		})
		return
	}
	start := th.Space(2)
	if t.checks {
		start = th.Space(8) // room for the check box at the start corner
	}
	ui.Row(c).AlignItems(ui.Center).Gap(th.Space(2)).PaddingStart(start).Children(func() {
		if img != nil {
			ui.Image(c, img).Size(16, 16)
		}
		ui.Text(c, text).SingleLine()
	})
}

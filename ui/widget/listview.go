package widget

import (
	"cmp"
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
	// colImage is the Image the first column had when the list view was made.
	colImage func(*T) *ui.Bitmap
	// imageSize is the side an image list says its images are drawn at, 0 for
	// the defaults: iconSize for large icons, 16 for the others.
	imageSize float32

	onSelect func([]K)
	onCheck  func(K, bool)

	// cursorKey is the row the keys move from, and anchorKey the row Shift
	// extends the choice from, in every view: they cross a switch of the view
	// by key. fixed says a setter chose them, not ui's cursor and pivot.
	cursorKey, anchorKey K
	hasCursor, hasAnchor bool
	fixed                bool
	// last is the view built last, and focused whether it held the keyboard
	// focus, to give it to the view that follows.
	last    ViewMode
	built   bool
	focused bool

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
	if len(cols) > 0 {
		lv.colImage = cols[0].Image
	}
	lv.init(lv)
	mu.Lock()
	lv.t.watch = lv.watched
	mu.Unlock()
	return lv
}

// fillsAcross: a list view fills the cross axis of the box that holds it, so
// it stays virtualized there, as the controls that hold many rows do.
func (lv *ListViewWidget[T, K]) fillsAcross() bool { return true }

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
// by default; small, 16) and before the text of the first Details column (16);
// nil for none. fn runs under the package's lock and calls no widget method.
// Without it the first column's Image is used, if it has one.
func (lv *ListViewWidget[T, K]) SetImage(fn func(*T) *ui.Bitmap) *ListViewWidget[T, K] {
	lv.setImage(fn, 0)
	return lv
}

// SetImageList shows the image of the list under key(row) as a row's icon, drawn
// at the list's Size in the icon views and the list, as WinForms' ImageList.ImageSize
// says (Details draws 16 DIPs); no image for a key the list lacks. key runs under
// the package's lock.
func (lv *ListViewWidget[T, K]) SetImageList(il *ImageListWidget, key func(*T) string) *ListViewWidget[T, K] {
	lv.setImage(ImagesBy(il, key), float32(il.Size()))
	return lv
}

// setImage sets the icon function and the side it is drawn at, and gives the
// first Details column the function too.
func (lv *ListViewWidget[T, K]) setImage(fn func(*T) *ui.Bitmap, side float32) {
	lv.t.change(func() {
		lv.image, lv.imageSize = fn, side
		if len(lv.t.cols) > 0 {
			if fn == nil {
				fn = lv.colImage // the column's own, if the app gave one
			}
			lv.t.cols[0].Image = fn
		}
	})
	lv.change(func() {})
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
				lv.anchorKey, lv.hasAnchor = k, true
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
	switched := lv.built && lv.last != lv.mode
	lv.takeKeys()
	wantFocus := switched && lv.focused
	var e ui.Element
	if lv.mode == ViewDetails {
		if switched || !lv.built {
			lv.giveDetails()
		}
		e = render(c, t)
		if wantFocus {
			t.list.Focus(c) // the table, not the column around it and its panel
		}
	} else {
		e = lv.buildIcons(c, wantFocus)
	}
	lv.last, lv.built = lv.mode, true
	lv.focused = e.FocusWithin()
	return e
}

// keyOfRow returns the key of the row shown at row in the last frame built,
// whichever view built it. Under mu.
func (lv *ListViewWidget[T, K]) keyOfRow(row int) (k K, ok bool) {
	if row < 0 {
		return k, false
	}
	if r := lv.t.rowAt(row); r != nil {
		return lv.t.key(r), true
	}
	return k, false
}

// takeKeys reads where the cursor and the pivot are, as keys, from the view
// built last, which the next build gives to the view it builds. Under mu.
func (lv *ListViewWidget[T, K]) takeKeys() {
	if !lv.built {
		return
	}
	cursor, pivot := lv.cursor, lv.grid.Pivot()
	if lv.last == ViewDetails {
		cursor, pivot = lv.t.cursor, lv.t.list.Pivot()
	}
	if lv.fixed {
		lv.fixed = false
		return
	}
	if k, ok := lv.keyOfRow(cursor); ok {
		lv.cursorKey, lv.hasCursor = k, true
	}
	if k, ok := lv.keyOfRow(pivot); ok {
		lv.anchorKey, lv.hasAnchor = k, true
	}
}

// giveDetails puts the cursor and the pivot of the view built before into the
// table, which ui keeps by key from there. Under mu.
func (lv *ListViewWidget[T, K]) giveDetails() {
	t := lv.t
	t.catchUp()
	t.sync()
	t.cursor = -1
	if lv.hasCursor {
		t.cursor = t.positionOf(lv.cursorKey)
	}
	pivot := -1
	if lv.hasAnchor {
		pivot = t.positionOf(lv.anchorKey)
	}
	t.list.SetPivot(pivot)
}

// buildIcons builds the rows as a grid of items, with the table's bottom panel
// under it, and the keys of the table's OnKey over it. Under mu.
func (lv *ListViewWidget[T, K]) buildIcons(c *ui.Context, focus bool) ui.Element {
	t := lv.t
	// The table is not built, but its changes redraw this window.
	keeping.hold(&t.core)
	t.catchUp()
	t.sync()
	t.frame, t.kept, t.gone = t.src.rows, false, false
	n := t.view.Len()

	lv.cursor = -1
	if lv.hasCursor {
		lv.cursor = t.positionOf(lv.cursorKey)
	}
	g := &lv.grid
	if lv.hasAnchor {
		g.SetPivot(t.positionOf(lv.anchorKey))
	} else {
		g.SetPivot(-1)
	}
	g.Key, g.Label, g.Selected = t.keyAt, t.labelAt, &lv.cursor
	g.Selection = t.chosen.bind()
	g.AccentChoice = true
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

	side := cmp.Or(lv.imageSize, lv.iconSize)
	minWidth, height := side+56, side+44
	small := lv.mode == ViewSmallIcon || lv.mode == ViewList
	if small {
		minWidth, height = 180, max(30, cmp.Or(lv.imageSize, 16)+8)
		if lv.mode == ViewList {
			minWidth = 1e6 // one column
		}
	}
	// The boxes of the large icons are the grid's, at the corner of the
	// picture; the small items hold their own, in the row of the text.
	g.Checked = nil
	if t.checks {
		checks := t.checked.bind() // takes what setters checked, for the boxes of every view
		if !small {
			g.Checked = checks
		}
	}
	g.PlainChecks = lv.image == nil && (len(t.cols) == 0 || t.cols[0].Image == nil)
	var grid ui.Element
	col := ui.Column(c).Gap(8).Grow(1).Children(func() {
		grid = ui.GridView(c, g, n, minWidth, height, func(i int) { lv.item(c, i) }).Grow(1).Padding(0, c.Theme().Space(2)).
			OnChange(func() { t.commit(&t.chosen) }).
			OnCheckedChange(func() { t.commit(&t.checked) })
		if t.panel != nil {
			t.counts()
			render(c, t.panel)
		}
	})
	for _, sc := range t.shortcuts {
		fn := sc.fn
		grid.OnShortcut(sc.mods, sc.key, func() { Post(fn) })
	}
	if t.checks && small {
		grid.OnShortcut(0, ui.KeySpace, lv.toggleChecks)
	}
	if focus {
		grid.Focus()
	}
	if fn := t.onActivate; fn != nil && grid.Submitted() && lv.cursor >= 0 {
		if r := t.rowAt(lv.cursor); r != nil {
			k := t.key(r)
			Post(func() { fn(k) })
		}
	}
	return col
}

// toggleChecks is Space in the small views, whose boxes are the items' own: it
// checks the rows chosen, or unchecks them when all are checked. On the main
// thread.
func (lv *ListViewWidget[T, K]) toggleChecks() {
	t := lv.t
	chosen := slices.Collect(t.chosen.shown.All())
	checked := &t.checked.shown
	all := len(chosen) > 0
	for _, k := range chosen {
		if !checked.Has(k) {
			all = false
			break
		}
	}
	for _, k := range chosen {
		if all {
			checked.Remove(k)
		} else {
			checked.Add(k)
		}
	}
	t.commit(&t.checked)
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
		side := cmp.Or(lv.imageSize, lv.iconSize)
		ui.Column(c).AlignItems(ui.Center).Gap(th.Space(1)).Padding(th.Space(1)).Children(func() {
			if img != nil {
				ui.Image(c, img).Size(side, side)
			} else {
				ui.Box(c).Size(side, side)
			}
			ui.Text(c, text).MaxLines(2).TextAlign(ui.Center)
		})
		return
	}
	// One row that fills the item and centres what it holds, the box and the
	// text on the same line, as a Details cell lays them out.
	side := cmp.Or(lv.imageSize, 16)
	ui.Row(c).Grow(1).AlignItems(ui.Center).Gap(th.Space(2)).PaddingStart(th.Space(2)).Children(func() {
		if t.checks {
			k := t.key(r)
			on := t.checked.shown.Has(k)
			ui.Checkbox(c, &on, "").Label("Check " + text).Unfocusable().OnChange(func() {
				if on {
					t.checked.shown.Add(k)
				} else {
					t.checked.shown.Remove(k)
				}
				t.commit(&t.checked)
			})
		}
		if img != nil {
			ui.Image(c, img).Size(side, side)
		}
		ui.Text(c, text).SingleLine()
	})
}

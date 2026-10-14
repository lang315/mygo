package widget

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Column is a column of a table of rows of type T. Its functions get a
// pointer to a row and run under the package's lock, while the table
// builds, sorts or searches: they read the row and call no widget method.
type Column[T any] struct {
	// Title heads the column; it is also its ID in a ui.TableLayout.
	Title string
	// Text is what a cell shows. It is required.
	Text func(*T) string
	// Compare sorts by the column, as cmp.Compare does; nil compares Text.
	Compare func(a, b *T) int
	// Width and MinWidth are in DIPs; a Width of 0 shares the room left.
	Width, MinWidth float32
	Align           Align
	Sortable        bool
	// Filterable lists the column in the bottom panel's search.
	Filterable bool
	// Filter keeps a row for a search's keyword; nil keeps the rows whose
	// Text holds the keyword, whatever its case.
	Filter func(row *T, keyword string) bool
	// Color is the color of the text; nil, or a zero Color, keeps the
	// theme's.
	Color func(*T) ui.Color
	// Image shows before the text, 16 DIPs square; nil for none.
	Image func(*T) *ui.Bitmap
}

// TableWidget shows rows of type T, each told apart by a key of type K,
// in columns that sort, for as many as a million rows. The table keeps
// its own copy of the rows. A table shows in one window at a time.
type TableWidget[T any, K comparable] struct {
	common[TableWidget[T, K]]
	key      func(*T) K
	cols     []Column[T]
	columns  []ui.TableColumn // cols, after the index column if any
	rows     []T
	index    map[K]int // a key's row in rows
	view     *ui.TableView[T]
	sort     mirror[ui.SortOrder]
	numbered bool // the index column shows
	// ui's, on the main thread:
	list   ui.ListState
	cursor int
}

// Table makes a table of cols whose rows key tells apart. key runs under
// the package's lock, as the columns' functions do.
func Table[T any, K comparable](key func(*T) K, cols ...Column[T]) *TableWidget[T, K] {
	compare := map[string]func(a, b *T) int{}
	for _, cl := range cols {
		switch {
		case !cl.Sortable:
		case cl.Compare != nil:
			compare[cl.Title] = cl.Compare
		default:
			text := cl.Text
			compare[cl.Title] = func(a, b *T) int { return strings.Compare(text(a), text(b)) }
		}
	}
	t := &TableWidget[T, K]{key: key, cols: cols, index: map[K]int{}, view: ui.NewTableView(compare), cursor: -1}
	t.columns = t.uiColumns()
	t.list.Key, t.list.Label, t.list.Selected = t.keyAt, t.labelAt, &t.cursor
	t.init(t)
	return t
}

// uiColumns returns the columns ui shows. Under mu, or before the table
// is shared.
func (t *TableWidget[T, K]) uiColumns() []ui.TableColumn {
	var cs []ui.TableColumn
	if t.numbered {
		cs = append(cs, ui.TableColumn{Title: "#", Width: 56, Align: ui.End, Fixed: true})
	}
	for _, cl := range t.cols {
		cs = append(cs, ui.TableColumn{Title: cl.Title, Width: cl.Width, MinWidth: cl.MinWidth,
			Align: cl.Align.ui(), Sortable: cl.Sortable})
	}
	return cs
}

// IndexColumn shows first the position of each row as shown, from 1.
func (t *TableWidget[T, K]) IndexColumn() *TableWidget[T, K] {
	change(func() { t.numbered = true; t.columns = t.uiColumns() })
	return t
}

// SetRows shows a copy of rows instead of the rows shown. Of rows with
// the same key, the last stays, where the first was.
func (t *TableWidget[T, K]) SetRows(rows []T) *TableWidget[T, K] {
	change(func() {
		t.rows, t.index = make([]T, 0, len(rows)), make(map[K]int, len(rows))
		t.add(rows)
		t.view.Reset()
	})
	return t
}

// AddRow shows row after the others, or instead of the row of its key.
func (t *TableWidget[T, K]) AddRow(row T) *TableWidget[T, K] { return t.AddRows([]T{row}) }

// AddRows shows rows after the others, each instead of the row of its
// key if one shows.
func (t *TableWidget[T, K]) AddRows(rows []T) *TableWidget[T, K] {
	change(func() { t.add(rows) })
	return t
}

// add appends rows, replacing those of keys shown. Under mu.
func (t *TableWidget[T, K]) add(rows []T) {
	for i := range rows {
		k := t.key(&rows[i])
		if j, ok := t.index[k]; ok {
			t.rows[j] = rows[i]
			t.view.Changed(j)
			continue
		}
		t.index[k] = len(t.rows)
		t.rows = append(t.rows, rows[i])
	}
}

// UpdateRow runs fn on the row of key k, if one shows, to change it in
// place. fn runs under the package's lock: it changes the row but not its
// key, and calls no widget method.
func (t *TableWidget[T, K]) UpdateRow(k K, fn func(*T)) *TableWidget[T, K] {
	change(func() {
		if i, ok := t.index[k]; ok {
			fn(&t.rows[i])
			t.view.Changed(i)
		}
	})
	return t
}

// DeleteRows removes the rows of keys ks.
func (t *TableWidget[T, K]) DeleteRows(ks ...K) *TableWidget[T, K] {
	change(func() {
		n := len(t.index)
		for _, k := range ks {
			delete(t.index, k)
		}
		if len(t.index) == n {
			return
		}
		t.rows = slices.DeleteFunc(t.rows, func(r T) bool { _, ok := t.index[t.key(&r)]; return !ok })
		for i := range t.rows {
			t.index[t.key(&t.rows[i])] = i
		}
		t.view.Reset()
	})
	return t
}

// Clear removes every row.
func (t *TableWidget[T, K]) Clear() *TableWidget[T, K] {
	change(func() {
		t.rows = nil
		clear(t.index)
		t.view.Reset()
	})
	return t
}

// Row returns the row of key k.
func (t *TableWidget[T, K]) Row(k K) (T, bool) {
	mu.Lock()
	defer mu.Unlock()
	if i, ok := t.index[k]; ok {
		return t.rows[i], true
	}
	var zero T
	return zero, false
}

// Len returns how many rows the table has, the hidden by a search too.
func (t *TableWidget[T, K]) Len() int { mu.Lock(); defer mu.Unlock(); return len(t.rows) }

// ClearSort shows the rows in the order added, as before a header click.
func (t *TableWidget[T, K]) ClearSort() *TableWidget[T, K] {
	change(func() { t.sort.set(ui.SortOrder{}) })
	return t
}

// sync brings the view up to date with the rows. Under mu.
func (t *TableWidget[T, K]) sync() { t.view.Sync(t.rows, t.sort.value) }

// keyOf returns the key of the row shown at row, the zero K for a row no
// longer there: a goroutine may change the rows between a frame and ui
// applying input to it. Under mu.
func (t *TableWidget[T, K]) keyOf(row int) K {
	if row < t.view.Len() {
		if i := t.view.Row(row); i < len(t.rows) {
			return t.key(&t.rows[i])
		}
	}
	var zero K
	return zero
}

// keyAt is ui's ListState.Key: ui asks while the table builds, holding
// mu, and while it applies input, not.
func (t *TableWidget[T, K]) keyAt(row int) any {
	if !viewing.Load() {
		mu.Lock()
		defer mu.Unlock()
	}
	return t.keyOf(row)
}

// labelAt is ui's ListState.Label, the first column's text: it names a
// row's check box, and typing finds rows by it.
func (t *TableWidget[T, K]) labelAt(row int) string {
	if !viewing.Load() {
		mu.Lock()
		defer mu.Unlock()
	}
	if len(t.cols) == 0 || row >= t.view.Len() {
		return ""
	}
	if i := t.view.Row(row); i < len(t.rows) {
		return t.cols[0].Text(&t.rows[i])
	}
	return ""
}

func (t *TableWidget[T, K]) build(c *ui.Context) ui.Element {
	return ui.Column(c).Children(func() { t.table(c).Grow(1) })
}

// table builds the ui table. Under mu.
func (t *TableWidget[T, K]) table(c *ui.Context) ui.Element {
	s := &t.list
	s.Sort, _ = t.sort.bind()
	t.sync()
	s.Version = t.view.Version()
	return ui.Table(c, s, t.columns, t.view.Len(), func(row, col int) { t.cell(c, row, col) }).
		OnSortChange(onChange(&t.sort, nil))
}

// cell builds the cell of the row shown at row in column col, an index
// into t.columns. Under mu.
func (t *TableWidget[T, K]) cell(c *ui.Context, row, col int) {
	if t.numbered {
		if col == 0 {
			ui.Text(c, strconv.Itoa(row+1)).SingleLine()
			return
		}
		col--
	}
	cl, r := &t.cols[col], &t.rows[t.view.Row(row)]
	if cl.Image == nil {
		cellText(c, cl, r)
		return
	}
	ui.Row(c).Gap(4).Children(func() {
		ui.Image(c, cl.Image(r)).Size(16, 16)
		cellText(c, cl, r)
	})
}

func cellText[T any](c *ui.Context, cl *Column[T], r *T) {
	e := ui.Text(c, cl.Text(r)).SingleLine()
	if cl.Color != nil {
		if col := cl.Color(r); col != (ui.Color{}) {
			e.TextColor(col)
		}
	}
}

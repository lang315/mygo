package ui

import (
	"cmp"
	"slices"
	"unsafe"
)

// A TableView is the order a Table shows an app's rows in: the rows of a
// slice its filter keeps, sorted by the table's SortOrder. It keeps that
// order from frame to frame, and only places again the rows the app says
// changed, so a table of 100,000 rows updating many times a second sorts
// and filters them cheaply. Use it on the main thread, as the view:
//
//	app.view.Sync(app.accounts, app.sort)
//	app.list.Sort = &app.sort
//	app.list.Key = func(i int) any { return app.accounts[app.view.Row(i)].ID }
//	app.list.Version = app.view.Version()
//	ui.Table(c, &app.list, cols, app.view.Len(), func(row, col int) {
//		a := &app.accounts[app.view.Row(row)]
//		// …
//	})
type TableView[T any] struct {
	compare map[string]func(a, b T) int
	keep    func(T) bool
	// base, n and sort are the slice and the order the view last showed.
	base *T
	n    int
	sort SortOrder
	// perm holds the rows shown, by index into the slice; out, spare and
	// ins are buffers merges reuse.
	perm, out, spare, ins []int
	// dirty are the rows of the slice the app changed (or appended) since
	// the last Sync; mark is the scratch set merge uses to drop them from
	// the order and count each once.
	dirty   []int
	mark    []bool
	reset   bool
	version uint64
}

// NewTableView returns a view sorting by the columns of compare, by their
// IDs (TableColumn.ID, or its Title when empty): each returns a negative
// number, zero or a positive number as a sorts before, with or after b, as
// cmp.Compare does. Rows a comparator finds equal keep the slice's order,
// and a column without one shows the rows in it.
func NewTableView[T any](compare map[string]func(a, b T) int) *TableView[T] {
	return &TableView[T]{compare: compare, reset: true}
}

// SetFilter shows only the rows keep keeps, or all of them for nil. It
// places all rows again, as functions cannot be compared: call it when
// what the filter keeps changes, not in every frame.
func (v *TableView[T]) SetFilter(keep func(T) bool) { v.keep, v.reset = keep, true }

// Changed tells the view that the rows of the slice at those indices
// changed in place, for the next Sync to place them again. Rows appended
// to the slice need not be named.
func (v *TableView[T]) Changed(rows ...int) { v.dirty = append(v.dirty, rows...) }

// Reset has the next Sync place all rows again, as after rows were
// removed or moved within the slice: Sync cannot tell when rows removed
// and others appended bring the slice back to its length.
func (v *TableView[T]) Reset() { v.reset = true }

// Len returns how many rows the view shows.
func (v *TableView[T]) Len() int { return len(v.perm) }

// Row returns the index into the slice of the row shown at i.
func (v *TableView[T]) Row(i int) int { return v.perm[i] }

// Version returns a number that changes whenever the rows shown, or their
// order, change: give it to ListState.Version.
func (v *TableView[T]) Version() uint64 { return v.version }

// Sync brings the view up to date with rows sorted by sort: call it in
// every frame, before building the table. It costs nothing when nothing
// changed; it places again only the rows changed and appended since; and
// it places all of them when the sort, the filter or the slice changed
// (rows removed, or another slice), or after Reset.
func (v *TableView[T]) Sync(rows []T, sort SortOrder) {
	if v.reset || sort != v.sort || unsafe.SliceData(rows) != v.base || len(rows) < v.n {
		v.rebuild(rows, sort)
		return
	}
	for i := v.n; i < len(rows); i++ {
		v.dirty = append(v.dirty, i)
	}
	v.n = len(rows)
	switch {
	case len(v.dirty) == 0:
	case len(v.dirty) > len(rows)/8:
		v.rebuild(rows, sort)
	default:
		v.merge(rows)
	}
}

// order returns how rows sort: by the column sorted by, then by their
// index, so that no two rows are equal.
func (v *TableView[T]) order(rows []T) func(i, j int) int {
	f := v.compare[v.sort.Column]
	if f == nil {
		return cmp.Compare[int]
	}
	desc := v.sort.Descending
	return func(i, j int) int {
		// Descending swaps the arguments: negating a result of
		// math.MinInt would overflow.
		var d int
		if desc {
			d = f(rows[j], rows[i])
		} else {
			d = f(rows[i], rows[j])
		}
		if d != 0 {
			return d
		}
		return cmp.Compare(i, j)
	}
}

// rebuild places all rows.
func (v *TableView[T]) rebuild(rows []T, sort SortOrder) {
	v.reset, v.sort, v.base, v.n = false, sort, unsafe.SliceData(rows), len(rows)
	v.dirty = v.dirty[:0]
	out := v.out[:0]
	for i := range rows {
		if v.keep == nil || v.keep(rows[i]) {
			out = append(out, i)
		}
	}
	if v.compare[sort.Column] != nil {
		slices.SortFunc(out, v.order(rows))
	}
	// A rebuild can yield the order it had: the version then stays.
	if !slices.Equal(out, v.perm) {
		v.version++
	}
	v.perm, v.out = out, v.perm
}

// merge places again the rows changed: it takes them out of the order,
// sorts those the filter keeps, and puts each where a binary search finds
// its place, copying the rows between.
func (v *TableView[T]) merge(rows []T) {
	by := v.order(rows)
	if len(v.mark) < len(rows) {
		v.mark = make([]bool, len(rows))
	}
	// The rows changed, once each, and those of the slice only.
	dirty := v.dirty[:0]
	for _, i := range v.dirty {
		if i >= 0 && i < len(rows) && !v.mark[i] {
			v.mark[i] = true
			dirty = append(dirty, i)
		}
	}
	rest := v.spare[:0]
	for _, i := range v.perm {
		if !v.mark[i] {
			rest = append(rest, i)
		}
	}
	ins := v.ins[:0]
	for _, i := range dirty {
		v.mark[i] = false
		if v.keep == nil || v.keep(rows[i]) {
			ins = append(ins, i)
		}
	}
	slices.SortFunc(ins, by)
	// out cannot be perm, which it is compared with: the three buffers
	// rotate, perm's array becoming the next out.
	out := v.out[:0]
	from := 0
	for _, i := range ins {
		at, _ := slices.BinarySearchFunc(rest[from:], i, by)
		out = append(out, rest[from:from+at]...)
		out = append(out, i)
		from += at
	}
	out = append(out, rest[from:]...)
	if !slices.Equal(out, v.perm) {
		v.version++
	}
	v.perm, v.out, v.spare, v.ins, v.dirty = out, v.perm, rest, ins, dirty[:0]
}

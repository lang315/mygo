package ui

import (
	"cmp"
	"slices"
	"unsafe"
)

// A TableView is the order a Table shows an app's rows in: the rows of a
// slice its filter keeps, sorted by the table's SortOrder, or by the
// several of its SortBy. It keeps that order from frame to frame, and only
// places again the rows the app says changed, so a table of 100,000 rows
// updating many times a second sorts and filters them cheaply. Use it on
// the main thread, as the view:
//
//	app.view.Sync(app.accounts, app.sort)
//	app.list.Sort = &app.sort
//	app.list.Key = func(i int) any { return app.accounts[app.view.Row(i)].ID }
//	app.list.Version = app.view.Version()
//	ui.Table(c, &app.list, cols, app.view.Len(), func(row, col int) {
//		a := &app.accounts[app.view.Row(row)]
//		// …
//	})
//
// With ListState.SortBy, Sync sorts by its columns, the first first:
//
//	app.view.Sync(app.accounts, app.sortBy...)
//	app.list.SortBy = &app.sortBy
type TableView[T any] struct {
	compare map[string]func(a, b *T) int
	// keep is the filter the order shown applies; next is the one
	// SetFilter gave since the last Sync, when newFilter.
	keep, next func(*T) bool
	newFilter  bool
	// filterKey is the key SetFilterKey last gave, when keyed.
	filterKey any
	keyed     bool
	// base, n and sort are the slice and the order the view last showed,
	// sort a copy of the columns sorted by; keys are those of them with a
	// comparator.
	base *T
	n    int
	sort []SortOrder
	keys []tvKey[T]
	// perm holds the rows shown, by index into the slice; out, spare and
	// ins are buffers merges reuse. With no filter, perm is the order of
	// all rows.
	perm, out, spare, ins []int
	// While a filter hides rows, all is the order of every row when
	// hasAll, kept up to date as perm is, so that a new filter goes
	// through it instead of sorting the rows again; kept marks the rows a
	// filter keeps.
	all    []int
	hasAll bool
	kept   []bool
	// dirty are the rows of the slice the app changed (or appended) since
	// the last Sync; mark is the scratch set merge uses to drop them from
	// the order and count each once.
	dirty   []int
	mark    []bool
	reset   bool
	version uint64
}

// NewTableView returns a view sorting by the columns of compare, by their
// IDs (TableColumn.ID, or its Title when empty): each gets pointers to two
// rows of the slice given to Sync, and returns a negative number, zero or
// a positive number as a sorts before, with or after b, as cmp.Compare
// does. Pointers spare copying a large row at each comparison; a
// comparator must not change the rows, nor keep the pointers. Rows a
// comparator finds equal keep the slice's order, and a column without one
// shows the rows in it.
func NewTableView[T any](compare map[string]func(a, b *T) int) *TableView[T] {
	return &TableView[T]{compare: compare, reset: true}
}

// SetFilter shows only the rows keep keeps, or all of them for nil, from
// the next Sync. Like a comparator, keep gets a pointer to a row of the
// slice, which it must not change nor keep. As functions cannot be
// compared, that Sync goes through all rows, so call it when what the
// filter keeps changes, not in every frame. It sorts none of them again,
// as a filter does not change their order, which the view keeps; except
// after a sort under a filter keeping half the rows or fewer (at the first
// Sync, or after a change of the sort, Reset or another slice): the view
// then sorted only the rows kept, and the next filter sorts again. While a
// filter hides rows, the view keeps the order of all of them up to date
// too, so placing rows changed costs about half again as much.
func (v *TableView[T]) SetFilter(keep func(*T) bool) {
	v.next, v.newFilter, v.keyed = keep, true, false
}

// SetFilterKey is SetFilter for a filter named by key, a comparable value
// such as the search typed: it does nothing when key is the one it was
// last given, so that a view may call it in every frame,
//
//	app.view.SetFilterKey(app.search, matches(app.search))
//
// and the rows are filtered again only when the search changes. SetFilter
// forgets the key.
func (v *TableView[T]) SetFilterKey(key any, keep func(*T) bool) {
	if v.keyed && v.filterKey == key {
		return
	}
	v.SetFilter(keep)
	v.filterKey, v.keyed = key, true
}

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

// Sync brings the view up to date with rows sorted by the columns of by,
// the first first, as ListState.SortBy holds them, or by none: call it in
// every frame, before building the table. It costs nothing when nothing
// changed; it places again only the rows changed and appended since; a new
// filter goes through the rows once; and it sorts all of them when the
// sort or the slice changed (rows removed, or another slice), or after
// Reset.
func (v *TableView[T]) Sync(rows []T, by ...SortOrder) {
	// A new filter without the order of all rows to go through sorts them
	// again: placing the rows changed first would be lost.
	if v.reset || !slices.Equal(by, v.sort) || unsafe.SliceData(rows) != v.base || len(rows) < v.n ||
		v.newFilter && v.keep != nil && !v.hasAll {
		v.rebuild(rows, by)
		return
	}
	for i := v.n; i < len(rows); i++ {
		v.dirty = append(v.dirty, i)
	}
	v.n = len(rows)
	switch {
	case len(v.dirty) == 0:
	case len(v.dirty) > len(rows)/8:
		v.rebuild(rows, by)
	default:
		v.merge(rows)
	}
	if v.newFilter {
		v.refilter(rows)
	}
}

// A tvKey is a column a TableView sorts by: its comparator, and whether
// descending.
type tvKey[T any] struct {
	f    func(a, b *T) int
	desc bool
}

// order returns how rows sort: by the columns sorted by that have a
// comparator, in turn, then by their index, so that no two rows are equal.
func (v *TableView[T]) order(rows []T) func(i, j int) int {
	keys := v.keys
	switch len(keys) {
	case 0:
		return cmp.Compare[int]
	case 1:
		// One column, the common case, spares the loop.
		f, desc := keys[0].f, keys[0].desc
		return func(i, j int) int {
			// Descending swaps the arguments: negating a result of
			// math.MinInt would overflow.
			var d int
			if desc {
				d = f(&rows[j], &rows[i])
			} else {
				d = f(&rows[i], &rows[j])
			}
			if d != 0 {
				return d
			}
			return cmp.Compare(i, j)
		}
	}
	return func(i, j int) int {
		for _, k := range keys {
			var d int
			if k.desc {
				d = k.f(&rows[j], &rows[i])
			} else {
				d = k.f(&rows[i], &rows[j])
			}
			if d != 0 {
				return d
			}
		}
		return cmp.Compare(i, j)
	}
}

// rebuild places all rows, by the filter given last. It sorts the rows
// shown, or all of them when a filter keeps more than half: sorting those
// costs at most about twice as much, and their order lets the next filter
// sort none again. After a narrower filter, the order of all rows comes
// back with the next rebuild.
func (v *TableView[T]) rebuild(rows []T, by []SortOrder) {
	if v.newFilter {
		v.keep, v.next, v.newFilter = v.next, nil, false
	}
	v.reset, v.sort, v.base, v.n = false, append(v.sort[:0], by...), unsafe.SliceData(rows), len(rows)
	v.keys = v.keys[:0]
	for _, o := range by {
		if f := v.compare[o.Column]; f != nil {
			v.keys = append(v.keys, tvKey[T]{f, o.Descending})
		}
	}
	v.dirty = v.dirty[:0]
	out := v.out[:0]
	if v.keep == nil {
		for i := range rows {
			out = append(out, i)
		}
	} else {
		v.filter(rows, v.keep)
		for i, k := range v.kept[:len(rows)] {
			if k {
				out = append(out, i)
			}
		}
	}
	// Rows in a column without a comparator show in the slice's order, which
	// a new filter finds again without all.
	sorted := len(v.keys) > 0
	v.hasAll = false
	switch {
	case sorted && v.keep != nil && len(out) > len(rows)/2:
		v.all = v.all[:0]
		for i := range rows {
			v.all = append(v.all, i)
		}
		slices.SortFunc(v.all, v.order(rows))
		out = v.keptIn(out[:0], v.all)
		v.hasAll = true
	case sorted:
		slices.SortFunc(out, v.order(rows))
	}
	// A rebuild can yield the order it had: the version then stays.
	v.show(out)
}

// refilter shows the rows the filter SetFilter gave keeps. A filter
// changes which rows show, not their order, so it goes through the order
// of all rows once, comparing none: perm when no filter hid rows, all
// otherwise, which Sync makes sure the view has.
func (v *TableView[T]) refilter(rows []T) {
	keep := v.next
	switch {
	case v.keep == nil && keep == nil: // no filter before, none after
		v.next, v.newFilter = nil, false
		return
	case v.keep == nil:
		v.all, v.hasAll = append(v.all[:0], v.perm...), true
	}
	v.keep, v.next, v.newFilter = keep, nil, false
	var out []int
	if keep == nil {
		out = append(v.out[:0], v.all...)
	} else {
		v.filter(rows, keep)
		out = v.keptIn(v.out[:0], v.all)
	}
	v.hasAll = keep != nil
	v.show(out)
}

// filter sets kept to what keep says of each row. The filter reads the rows
// in the slice's order, kinder to the cache than the order shown.
func (v *TableView[T]) filter(rows []T, keep func(*T) bool) {
	if len(v.kept) < len(rows) {
		v.kept = make([]bool, len(rows))
	}
	for i := range rows {
		v.kept[i] = keep(&rows[i])
	}
}

// keptIn appends to out the rows of order that kept marks, in that order.
func (v *TableView[T]) keptIn(out, order []int) []int {
	for _, i := range order {
		if v.kept[i] {
			out = append(out, i)
		}
	}
	return out
}

// show makes out, built in the buffer out, the order shown: the version
// changes only when the rows shown or their order do.
func (v *TableView[T]) show(out []int) {
	if !slices.Equal(out, v.perm) {
		v.version++
	}
	v.perm, v.out = out, v.perm
}

// merge places again the rows changed: it takes them out of the order,
// sorts those the filter keeps, and puts each where a binary search finds
// its place, copying the rows between. While a filter hides rows, it does
// the same to the order of all of them, for the next filter.
func (v *TableView[T]) merge(rows []T) {
	dirty := v.unique(rows)
	by := v.order(rows)
	// out cannot be the order placed again, which show compares it with:
	// the buffers rotate, the old order's array becoming the next out.
	if v.hasAll {
		all, rest, ins := v.place(rows, v.all, v.out, v.spare, v.ins, dirty, nil, by)
		v.all, v.out, v.spare, v.ins = all, v.all, rest, ins
	}
	out, rest, ins := v.place(rows, v.perm, v.out, v.spare, v.ins, dirty, v.keep, by)
	v.spare, v.ins = rest, ins
	v.show(out)
	v.dirty = dirty[:0]
}

// unique returns the rows changed, once each, and those of the slice only.
func (v *TableView[T]) unique(rows []T) []int {
	if len(v.mark) < len(rows) {
		v.mark = make([]bool, len(rows))
	}
	dirty := v.dirty[:0]
	for _, i := range v.dirty {
		if i >= 0 && i < len(rows) && !v.mark[i] {
			v.mark[i] = true
			dirty = append(dirty, i)
		}
	}
	for _, i := range dirty {
		v.mark[i] = false
	}
	return dirty
}

// place returns order with the rows dirty taken out and those keep keeps
// (all of them for nil) put back where by sorts them, built in out; rest
// and ins are the buffers it uses, returned to be used again.
func (v *TableView[T]) place(rows []T, order, out, rest, ins, dirty []int, keep func(*T) bool, by func(i, j int) int) ([]int, []int, []int) {
	// In a local, as v.mark would be loaded again at every row.
	mark := v.mark
	for _, i := range dirty {
		mark[i] = true
	}
	rest = rest[:0]
	for _, i := range order {
		if !mark[i] {
			rest = append(rest, i)
		}
	}
	ins = ins[:0]
	for _, i := range dirty {
		mark[i] = false
		if keep == nil || keep(&rows[i]) {
			ins = append(ins, i)
		}
	}
	slices.SortFunc(ins, by)
	out = out[:0]
	from := 0
	for _, i := range ins {
		at, _ := slices.BinarySearchFunc(rest[from:], i, by)
		out = append(out, rest[from:from+at]...)
		out = append(out, i)
		from += at
	}
	return append(out, rest[from:]...), rest, ins
}

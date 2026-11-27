package widget

import "github.com/egoist/mygo/ui"

// SetRowReorder lets the user drag rows to another place, or not, as by
// default: the rows chosen, when the row dragged is one, else that row, move
// before the row they drop on, in the table's own rows (Rows returns them in
// the new order), and OnRowsReordered's fn runs. It does nothing in a table
// over a RowSource (DataGridViewOf), whose rows are the app's, and a drag does
// not start while the rows are sorted, grouped or searched, as their order
// shown is then not theirs. A table cannot both reorder and drag rows out
// (DragRows): the later call wins.
func (t *DataGridViewWidget[T, K]) SetRowReorder(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.reorder = on
		if on {
			t.dragRows = nil
			if t.reorderFn == nil {
				t.reorderFn = t.reorderLines // a method value made in every frame allocates
			}
		}
	})
	return t
}

// OnRowsReordered runs fn on the event goroutine after the user moved rows
// (SetRowReorder): once for each drop that changed their order. Setters do not
// run it.
func (t *DataGridViewWidget[T, K]) OnRowsReordered(fn func()) *DataGridViewWidget[T, K] {
	t.change(func() { t.onReordered = fn })
	return t
}

// DragRows lets the user drag rows out of the table: the rows chosen, when the
// row dragged is one, else that row, in the order shown, drag the value fn
// returns, for an OnDrop or OnDragOver of its type elsewhere to take. fn gets
// copies of the rows, and runs on the main thread, off the package's lock, as
// the drag starts: it may read widgets, but should be quick. A nil result
// means those rows do not drag; a nil fn stops dragging rows out. It works in
// a table over a RowSource too, and sorted, grouped or searched. A table
// cannot both drag rows out and reorder them (SetRowReorder): the later call
// wins.
func (t *DataGridViewWidget[T, K]) DragRows(fn func(rows []*T) any) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.dragRows = fn
		if fn != nil {
			t.reorder = false
			if t.dragOutFn == nil {
				t.dragOutFn = t.dragLines
			}
		}
	})
	return t
}

// handDrag gives ui the table's row dragging, before it builds. Under mu.
func (t *DataGridViewWidget[T, K]) handDrag(s *ui.ListState) {
	s.Reorder, s.DragRows = nil, nil
	if t.reorder && t.reorderable() {
		s.Reorder = t.reorderFn
	}
	if t.dragRows != nil {
		s.DragRows = t.dragOutFn
	}
}

// reorderable reports whether the user may move the rows: the table's own,
// shown in their order. Under mu.
func (t *DataGridViewWidget[T, K]) reorderable() bool {
	return t.ownRows && len(t.sort.value) == 0 && t.groupBy == nil && t.filter == nil
}

// reorderLines is ui's ListState.Reorder: the lines dropped before the line
// to. It takes the keys of the rows and of the row at to, and posts the move,
// which changes the source: the frame ui builds still reads the view.
func (t *DataGridViewWidget[T, K]) reorderLines(lines []int, to int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	ks := make([]K, 0, len(lines))
	for _, l := range lines {
		if r := t.rowAt(l); r != nil {
			ks = append(ks, t.key(r))
		}
	}
	var before K
	r := t.rowAt(to)
	end := r == nil
	if !end {
		before = t.key(r)
	}
	Post(func() {
		if t.moveRows(ks, before, end) {
			mu.Lock()
			fn := t.onReordered
			mu.Unlock()
			if fn != nil {
				fn()
			}
		}
	})
}

// moveRows moves the rows of keys ks before the row of key before, or to the
// end, while the table still reorders, and reports whether their order
// changed.
func (t *DataGridViewWidget[T, K]) moveRows(ks []K, before K, end bool) (moved bool) {
	t.src.change(func() {
		if t.reorder && t.reorderable() {
			moved = t.src.move(ks, before, end)
		}
	})
	return moved
}

// move moves the rows of keys ks, in their order, before the row of key
// before (where it is, when it is one of them), or after the last row when
// end, and reports whether the order changed. A new slice, as InsertRows makes: a
// table's frame holds the old one. Under mu.
func (s *RowSource[T, K]) move(ks []K, before K, end bool) bool {
	at := len(s.rows)
	if !end {
		i, ok := s.index[before]
		if !ok {
			return false
		}
		at = i
	}
	take := make(map[K]bool, len(ks))
	for _, k := range ks {
		if _, ok := s.index[k]; ok {
			take[k] = true
		}
	}
	moved := make([]T, 0, len(take))
	for i := range s.rows {
		if take[s.key(&s.rows[i])] {
			moved = append(moved, s.rows[i])
		}
	}
	out := make([]T, 0, max(cap(s.rows), len(s.rows)))
	for i := range s.rows {
		if i == at {
			out = append(out, moved...)
		}
		if !take[s.key(&s.rows[i])] {
			out = append(out, s.rows[i])
		}
	}
	if at == len(s.rows) {
		out = append(out, moved...)
	}
	same := true
	for i := range out {
		if k := s.key(&out[i]); s.index[k] != i {
			s.index[k], same = i, false
		}
	}
	if same {
		return false
	}
	s.rows = out
	s.record(-1)
	return true
}

// dragLines is ui's ListState.DragRows: DragRows's fn, off the lock, with
// copies of the rows shown at lines.
func (t *DataGridViewWidget[T, K]) dragLines(lines []int) any {
	fn, rows := t.linesRows(lines)
	if fn == nil { // DragRows(nil) since the frame ui drags from
		return nil
	}
	ptrs := make([]*T, len(rows))
	for i := range rows {
		ptrs[i] = &rows[i]
	}
	return fn(ptrs)
}

// linesRows returns DragRows's fn and copies of the rows shown at lines.
func (t *DataGridViewWidget[T, K]) linesRows(lines []int) (func([]*T) any, []T) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	rows := make([]T, 0, len(lines))
	for _, l := range lines {
		if r := t.rowAt(l); r != nil {
			rows = append(rows, *r)
		}
	}
	return t.dragRows, rows
}

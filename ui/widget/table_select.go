package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// keys mirrors a set of row keys that ui changes, as mirror does a value:
// value is the table's, under mu; shown is ui's, on the main thread, which
// commit copies when ui reports a change.
type keys[K comparable] struct {
	value, shown ui.Selection[K]
	// changed holds the keys setters changed in value since the last
	// build, which shown then takes alone; nil with dirty, shown takes
	// all of value.
	changed []K
	dirty   bool
}

// bind returns the set to hand ui. Under mu, while building.
func (s *keys[K]) bind() *ui.Selection[K] {
	s.bindInto(&s.shown)
	return &s.shown
}

// bindInto brings dst, a set ui holds in place of shown, up to date with
// value. Under mu, while building.
func (s *keys[K]) bindInto(dst *ui.Selection[K]) {
	if s.dirty {
		dst.CopyFrom(&s.value)
	} else {
		for _, k := range s.changed {
			if s.value.Has(k) {
				dst.Add(k)
			} else {
				dst.Remove(k)
			}
		}
	}
	dst.TakeChanges() // what bind did is the table's, not the user's
	s.changed, s.dirty = s.changed[:0], false
}

// set adds or removes k. Under mu.
func (s *keys[K]) set(k K, on bool) {
	if on {
		s.value.Add(k)
	} else {
		s.value.Remove(k)
	}
	if !s.dirty {
		// Past an eighth of value, copying all of it costs less: a key
		// added costs about what cloning eight does.
		if s.changed = append(s.changed, k); len(s.changed)*8 > s.value.Len() {
			s.dirty = true
		}
	}
}

// keep drops the keys of rows gone from index. Under mu.
func (s *keys[K]) keep(index map[K]int) {
	var gone []K
	for k := range s.value.All() {
		if _, ok := index[k]; !ok {
			gone = append(gone, k)
		}
	}
	for _, k := range gone {
		s.set(k, false)
	}
}

// CheckboxColumn shows a check box first in each row, and one in the
// header that checks the rows shown.
func (t *DataGridViewWidget[T, K]) CheckboxColumn() *DataGridViewWidget[T, K] {
	t.change(func() { t.checks = true })
	return t
}

// shown returns the keys in that the rows shown hold, in the order shown.
// Under mu.
func (t *DataGridViewWidget[T, K]) shown(in *ui.Selection[K]) []K {
	t.catchUp()
	n := in.Len()
	if n == 0 {
		return nil
	}
	t.sync()
	if n <= t.view.Len()/8 {
		return t.shownByPosition(in)
	}
	var ks []K
	for i := range t.view.Len() {
		if k := t.key(&t.src.rows[t.view.Row(i)]); in.Has(k) {
			if ks = append(ks, k); len(ks) == n {
				break
			}
		}
	}
	return ks
}

// at is a key and the place of its row in the view.
type at[K comparable] struct {
	pos int32
	key K
}

// placed returns the keys of in whose rows show in the view as it is, with
// their places, in order. Under mu.
func (t *DataGridViewWidget[T, K]) placed(in *ui.Selection[K]) []at[K] {
	if t.posOf == nil || t.posVersion != t.view.Version() || len(t.posOf) != len(t.src.rows) {
		t.posOf = slices.Grow(t.posOf[:0], len(t.src.rows))[:len(t.src.rows)]
		for i := range t.posOf {
			t.posOf[i] = -1
		}
		for i := range t.view.Len() {
			t.posOf[t.view.Row(i)] = int32(i)
		}
		t.posVersion = t.view.Version()
	}
	found := make([]at[K], 0, in.Len())
	for k := range in.All() {
		if row, ok := t.src.index[k]; ok && row < len(t.posOf) && t.posOf[row] >= 0 {
			found = append(found, at[K]{t.posOf[row], k})
		}
	}
	slices.SortFunc(found, func(a, b at[K]) int { return int(a.pos) - int(b.pos) })
	return found
}

// shownByPosition is shown for few keys: it looks each key's row up and
// sorts them by their place in the view, instead of walking the view.
func (t *DataGridViewWidget[T, K]) shownByPosition(in *ui.Selection[K]) []K {
	found := t.placed(in)
	ks := make([]K, len(found))
	for i, f := range found {
		ks[i] = f.key
	}
	return ks
}

// chosenRows is ui's ListState.RowsOf: the rows of the frame ui shows that
// the user chose, when few and the frame is the view as it is; else false.
func (t *DataGridViewWidget[T, K]) chosenRows() ([]int, bool) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	in := &t.chosen.shown
	if t.kept || t.gone || in.Len() > t.view.Len()/8 {
		return nil, false
	}
	found := t.placed(in)
	rows := make([]int, len(found))
	for i, f := range found {
		rows[i] = int(f.pos)
	}
	return rows, true
}

// checkRows is ui's ListState.CheckRows: it checks or unchecks the rows
// first to last of the frame ui shows in one pass under the lock, and has
// the next frame take the checks as they are. Rows gone are left out.
func (t *DataGridViewWidget[T, K]) checkRows(first, last int, on bool) bool {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	c, before := &t.checked, t.checked.value.Len()
	// Clear is right exactly when the frame held every row: rows added
	// since, or a filter or a view that changed since, are not its.
	if !on && !t.gone && first == 0 && last+1 == len(t.src.rows) {
		c.value.Clear()
		c.dirty = true
	} else {
		for i := first; i <= last; i++ {
			r := t.rowAt(i)
			if r == nil {
				continue
			}
			k := t.key(r)
			if _, ok := t.src.index[k]; !ok && t.gone {
				continue
			}
			if c.value.Has(k) != on {
				c.set(k, on)
			}
		}
	}
	return c.value.Len() != before
}

// Selected returns the keys of the rows selected that show, in the order
// shown.
func (t *DataGridViewWidget[T, K]) Selected() []K {
	mu.Lock()
	defer mu.Unlock()
	return t.shown(&t.chosen.value)
}

// CountSelected returns how many rows are selected.
func (t *DataGridViewWidget[T, K]) CountSelected() int {
	mu.Lock()
	defer mu.Unlock()
	t.catchUp()
	return t.chosen.value.Len()
}

// Checked returns the keys of the rows checked that show, in the order
// shown.
func (t *DataGridViewWidget[T, K]) Checked() []K {
	mu.Lock()
	defer mu.Unlock()
	return t.shown(&t.checked.value)
}

// IsChecked reports whether the row of key k is checked.
func (t *DataGridViewWidget[T, K]) IsChecked(k K) bool {
	mu.Lock()
	defer mu.Unlock()
	t.catchUp()
	return t.checked.value.Has(k)
}

// SetChecked checks, or unchecks, the rows of keys ks.
func (t *DataGridViewWidget[T, K]) SetChecked(ks []K, on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		for _, k := range ks {
			if _, ok := t.src.index[k]; ok || !on {
				t.checked.set(k, on)
			}
		}
	})
	return t
}

// AutoCheck unchecks the rows selected that show when at least half of
// them are checked, and checks them all otherwise.
func (t *DataGridViewWidget[T, K]) AutoCheck() *DataGridViewWidget[T, K] {
	t.change(func() {
		ks, on := t.shown(&t.chosen.value), 0
		for _, k := range ks {
			if t.checked.value.Has(k) {
				on++
			}
		}
		check := on*2 < len(ks)
		for _, k := range ks {
			t.checked.set(k, check)
		}
	})
	return t
}

// checkedCount is ui's ListState.CheckedCount: how many of the rows shown
// are checked, of how many. Under mu, while building, after bind: the
// checks hold keys of rows only, so without a search, all of them show.
func (t *DataGridViewWidget[T, K]) checkedCount() (count, total int) {
	ks := &t.checked.shown
	if t.filter == nil {
		return ks.Len(), t.view.Len()
	}
	for k := range ks.All() {
		if i, ok := t.src.index[k]; ok && t.filter(&t.src.rows[i]) {
			count++
		}
	}
	return count, t.view.Len()
}

// commit takes what the user changed in s: only the keys ui changed,
// unless they were too many, and no key of a row gone: a goroutine may
// delete rows between a frame and ui applying input to it. Changes setters
// made since the last build stay pending, if ui changed only some keys, or
// replace the set; a setter's bulk change (dirty) gives way to a
// replacement. When a setter and the user changed the same key since the
// last build, the setter wins if ui replaced the set (the pending keys are
// replayed over it), and the user wins if ui changed keys one by one.
// On the main thread, without mu.
func (t *DataGridViewWidget[T, K]) commit(s *keys[K]) {
	mu.Lock()
	defer mu.Unlock()
	ks, all := s.shown.TakeChanges()
	if all {
		var on []bool
		if !s.dirty {
			for _, k := range s.changed {
				on = append(on, s.value.Has(k))
			}
		}
		s.value.CopyFrom(&s.shown)
		for i, k := range s.changed[:len(on)] {
			if on[i] {
				s.value.Add(k)
			} else {
				s.value.Remove(k)
			}
		}
		if s.dirty {
			s.changed, s.dirty = s.changed[:0], false
		}
	} else {
		for _, k := range ks {
			if s.shown.Has(k) {
				s.value.Add(k)
			} else {
				s.value.Remove(k)
			}
		}
	}
	if t.gone {
		s.keep(t.src.index)
	}
}

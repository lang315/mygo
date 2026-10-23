package widget

import "github.com/egoist/mygo/ui"

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
	switch {
	case s.dirty:
		s.shown.CopyFrom(&s.value)
	default:
		for _, k := range s.changed {
			if s.value.Has(k) {
				s.shown.Add(k)
			} else {
				s.shown.Remove(k)
			}
		}
	}
	s.changed, s.dirty = s.changed[:0], false
	return &s.shown
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
	n := in.Len()
	if n == 0 {
		return nil
	}
	t.sync()
	var ks []K
	for i := range t.view.Len() {
		if k := t.key(&t.rows[t.view.Row(i)]); in.Has(k) {
			if ks = append(ks, k); len(ks) == n {
				break
			}
		}
	}
	return ks
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
	return t.checked.value.Has(k)
}

// SetChecked checks, or unchecks, the rows of keys ks.
func (t *DataGridViewWidget[T, K]) SetChecked(ks []K, on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		for _, k := range ks {
			if _, ok := t.index[k]; ok || !on {
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
		if i, ok := t.index[k]; ok && t.filter(&t.rows[i]) {
			count++
		}
	}
	return count, t.view.Len()
}

// commit takes what the user changed in s, but no key of a row gone: a
// goroutine may delete rows between a frame and ui applying input to it.
// On the main thread, without mu.
func (t *DataGridViewWidget[T, K]) commit(s *keys[K]) {
	mu.Lock()
	defer mu.Unlock()
	s.value.CopyFrom(&s.shown)
	s.changed, s.dirty = s.changed[:0], false
	if t.gone {
		s.keep(t.index)
	}
}

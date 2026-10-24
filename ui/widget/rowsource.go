package widget

import (
	"weak"

	"github.com/egoist/mygo"
)

// A RowSource is the rows of tables, each told apart by a key of type K:
// one store for as many tables as show them (DataGridViewOf), each with its
// own sort, search, selection and checks. Its methods are safe from any
// goroutine; a change redraws every window that shows a table over it. The
// rows a source holds are never moved or removed in place: a table's
// frame keeps the rows it shows until it builds again.
type RowSource[T any, K comparable] struct {
	key   func(*T) K
	rows  []T
	index map[K]int // a key's row in rows

	// log holds the changes after seq start, oldest first: a table reads
	// those after its cursor. seq numbers the changes made, removedSeq is
	// the last that removed or replaced rows. Past its bound the log is
	// dropped, and a table that fell behind it starts over.
	log        []srcChange
	start, seq uint64
	removedSeq uint64
	tables     []weak.Pointer[tableLink]
}

// srcChange is one change of a source's rows: row changed in place, or, for
// row -1, rows removed or replaced.
type srcChange struct{ row int }

// tableLink is what a source knows of a table over it: where the table has
// read the log to, and the table's id, to find its windows. The source
// holds it weakly: a table dropped by the app is collected.
type tableLink struct {
	id  int
	seq uint64
}

// NewRowSource makes a source of rows told apart by key, which runs under
// the package's lock. It holds no rows.
func NewRowSource[T any, K comparable](key func(*T) K) *RowSource[T, K] {
	return &RowSource[T, K]{key: key, index: map[K]int{}}
}

// attach adds a table's link to the source, up to date. Under mu.
func (s *RowSource[T, K]) attach(l *tableLink) {
	l.seq = s.seq
	s.tables = append(s.tables, weak.Make(l))
}

// bound is how many changes the log holds.
func (s *RowSource[T, K]) bound() int { return max(64, len(s.rows)/8) }

// record logs a change. Under mu.
func (s *RowSource[T, K]) record(row int) {
	s.seq++
	if row < 0 {
		s.removedSeq = s.seq
	}
	s.log = append(s.log, srcChange{row})
	s.trim()
}

// trim drops what every live table has read, forgets the tables dropped,
// and the whole log past its bound. Under mu.
func (s *RowSource[T, K]) trim() {
	low := s.seq
	live := s.tables[:0]
	for _, w := range s.tables {
		if l := w.Value(); l != nil {
			live = append(live, w)
			low = min(low, l.seq)
		}
	}
	clear(s.tables[len(live):])
	s.tables = live
	if drop := int(low - s.start); drop > 0 {
		s.log, s.start = s.log[drop:], low
	}
	if len(s.log) > s.bound() {
		s.log, s.start = s.log[:0], s.seq
	}
}

// since returns the changes after seq, and false when they are gone from
// the log. Under mu.
func (s *RowSource[T, K]) since(seq uint64) ([]srcChange, bool) {
	if seq < s.start {
		return nil, false
	}
	return s.log[min(int(seq-s.start), len(s.log)):], true
}

// change runs fn under mu, then redraws the windows that show a table over
// the source.
func (s *RowSource[T, K]) change(fn func()) {
	mu.Lock()
	fn()
	var wins []*mygo.Window
	for w, f := range shown {
		for _, t := range s.tables {
			if l := t.Value(); l != nil {
				if _, ok := f.keep.widgets[l.id]; ok {
					wins = append(wins, w)
					break
				}
			}
		}
	}
	mu.Unlock()
	for _, w := range wins {
		invalidate(w)
	}
}

// logLen and attached are for tests.
func (s *RowSource[T, K]) logLen() int {
	mu.Lock()
	defer mu.Unlock()
	return len(s.log)
}

func (s *RowSource[T, K]) attached() (n int) {
	mu.Lock()
	defer mu.Unlock()
	for _, w := range s.tables {
		if w.Value() != nil {
			n++
		}
	}
	return n
}

// SetRows holds a copy of rows instead of its rows. Of rows with the same
// key, the last stays, where the first was.
func (s *RowSource[T, K]) SetRows(rows []T) *RowSource[T, K] {
	s.change(func() {
		s.rows, s.index = make([]T, 0, len(rows)), make(map[K]int, len(rows))
		s.add(rows)
		s.record(-1)
	})
	return s
}

// AddRows holds rows after the others, each instead of the row of its key
// if one is held.
func (s *RowSource[T, K]) AddRows(rows []T) *RowSource[T, K] {
	s.change(func() { s.add(rows) })
	return s
}

// add appends rows, replacing those of keys held. Under mu.
func (s *RowSource[T, K]) add(rows []T) {
	for i := range rows {
		k := s.key(&rows[i])
		if j, ok := s.index[k]; ok {
			s.rows[j] = rows[i]
			s.record(j)
			continue
		}
		s.index[k] = len(s.rows)
		s.rows = append(s.rows, rows[i])
	}
}

// UpdateRow runs fn on the row of key k, if one is held, to change it in
// place. fn runs under the package's lock: it changes the row but not its
// key, and calls no widget method.
func (s *RowSource[T, K]) UpdateRow(k K, fn func(*T)) *RowSource[T, K] {
	s.change(func() {
		if i, ok := s.index[k]; ok {
			fn(&s.rows[i])
			s.record(i)
		}
	})
	return s
}

// DeleteRows removes the rows of keys ks.
func (s *RowSource[T, K]) DeleteRows(ks ...K) *RowSource[T, K] {
	s.change(func() {
		first := len(s.rows) // the first row removed
		for _, k := range ks {
			if i, ok := s.index[k]; ok {
				first = min(first, i)
				delete(s.index, k)
			}
		}
		if first == len(s.rows) {
			return
		}
		rows := append(make([]T, 0, len(s.rows)), s.rows[:first]...)
		for i := first; i < len(s.rows); i++ {
			k := s.key(&s.rows[i])
			if _, ok := s.index[k]; ok {
				s.index[k] = len(rows)
				rows = append(rows, s.rows[i])
			}
		}
		s.rows = rows
		s.record(-1)
	})
	return s
}

// Row returns the row of key k.
func (s *RowSource[T, K]) Row(k K) (T, bool) {
	mu.Lock()
	defer mu.Unlock()
	if i, ok := s.index[k]; ok {
		return s.rows[i], true
	}
	var zero T
	return zero, false
}

// Len returns how many rows the source holds.
func (s *RowSource[T, K]) Len() int { mu.Lock(); defer mu.Unlock(); return len(s.rows) }

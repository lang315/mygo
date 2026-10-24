package ui

import (
	"fmt"
	"iter"
	"maps"
	"sync/atomic"
)

// A Selection is the rows chosen in a List or a Table that lets the user
// choose several (ListState.Selection), held by the keys of their items
// (ListState.Key), of type K: the choice stays with its items as rows come,
// go and move. Without a Key, rows are their indices, of type int. Its
// zero value is empty. The keys of items gone from the list stay until
// removed.
//
//	type app struct {
//		files  []File
//		chosen ui.Selection[string] // their paths
//		list   ui.ListState
//	}
//
//	app.list.Key = func(i int) any { return app.files[i].Path }
//	app.list.Selection = &app.chosen
type Selection[K comparable] struct {
	keys map[K]struct{}
	// gen changes with the keys, for those caching what they counted of
	// it: drawn from selectionGen, it differs from every other
	// Selection's, so assigning a new Selection changes it too.
	gen uint64
	// log holds the keys added or removed since TakeChanges, for those who
	// mirror the set; over says it gave up (Clear, CopyFrom, or past its
	// bound): read the whole set instead. Bounded, so a Selection nobody
	// takes changes from stays small.
	log  []K
	over bool
}

var selectionGen atomic.Uint64

// Has reports whether the row of key k is chosen.
func (s *Selection[K]) Has(k K) bool {
	_, ok := s.keys[k]
	return ok
}

// Len returns how many rows are chosen.
func (s *Selection[K]) Len() int { return len(s.keys) }

// Add chooses the row of key k.
func (s *Selection[K]) Add(k K) {
	if s.keys == nil {
		s.keys = map[K]struct{}{}
	}
	n := len(s.keys)
	s.keys[k] = struct{}{}
	if len(s.keys) != n {
		s.gen = selectionGen.Add(1)
		s.note(k)
	}
}

// Remove takes the row of key k out of the choice.
func (s *Selection[K]) Remove(k K) {
	if !s.Has(k) {
		return
	}
	s.gen = selectionGen.Add(1)
	delete(s.keys, k)
	s.note(k)
}

// note records k as changed.
func (s *Selection[K]) note(k K) {
	if s.over {
		return
	}
	if len(s.log) >= max(64, len(s.keys)/8) {
		s.over, s.log = true, s.log[:0]
		return
	}
	s.log = append(s.log, k)
}

// TakeChanges returns the keys added or removed since its last call, in
// order and maybe repeated, and resets the record: all is true when they
// were too many to list, or the set was cleared or replaced, and then the
// whole set must be read. The slice is reused: use it before changing s.
func (s *Selection[K]) TakeChanges() (keys []K, all bool) {
	keys, all = s.log, s.over
	s.log, s.over = s.log[:0], false
	return keys, all
}

// Clear chooses no row.
func (s *Selection[K]) Clear() {
	if len(s.keys) == 0 {
		return
	}
	s.gen = selectionGen.Add(1)
	clear(s.keys)
	s.over, s.log = true, s.log[:0]
}

// CopyFrom makes s choose the rows src does, in one step: faster than
// adding src's keys one by one when they are many.
func (s *Selection[K]) CopyFrom(src *Selection[K]) {
	if len(s.keys) == 0 && len(src.keys) == 0 {
		return
	}
	s.keys = maps.Clone(src.keys)
	s.gen = selectionGen.Add(1)
	s.over, s.log = true, s.log[:0]
}

// All returns the keys of the rows chosen, in no order.
func (s *Selection[K]) All() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range s.keys {
			if !yield(k) {
				return
			}
		}
	}
}

// A Selector is what ListState.Selection takes: a *Selection.
type Selector interface {
	has(key any) bool
	// set chooses the row of key, or takes it out, and reports whether
	// that changed the choice.
	set(key any, on bool) bool
	// clear chooses no row, and reports whether one was.
	clear() bool
	size() int
	// generation changes whenever the keys chosen do.
	generation() uint64
}

func (s *Selection[K]) has(key any) bool { return s.Has(s.key(key)) }

func (s *Selection[K]) set(key any, on bool) bool {
	k := s.key(key)
	if s.Has(k) == on {
		return false
	}
	if on {
		s.Add(k)
	} else {
		s.Remove(k)
	}
	return true
}

func (s *Selection[K]) clear() bool {
	n := len(s.keys)
	s.Clear()
	return n > 0
}

func (s *Selection[K]) size() int { return len(s.keys) }

func (s *Selection[K]) generation() uint64 { return s.gen }

// key returns a row's key as the selection holds it.
func (s *Selection[K]) key(key any) K {
	k, ok := key.(K)
	if !ok {
		panic(fmt.Sprintf("ui: the key of a row or an item is a %T, not the %T of its Selection", key, *new(K)))
	}
	return k
}

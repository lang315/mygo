package ui

import (
	"maps"
	"slices"
	"sync"
	"weak"
)

// globalOwners maps the ID of each element keyed globally to the engine
// that built it last, so that another window's engine building it takes
// over its state: a panel moved between windows keeps its scroll offsets,
// text and selection, and app locals.
var globalOwners struct {
	sync.Mutex
	m map[uint64]weak.Pointer[engine]
}

// noteGlobalOwner records rt as the engine building the global ID.
func noteGlobalOwner(id uint64, rt *engine) {
	globalOwners.Lock()
	if globalOwners.m == nil {
		globalOwners.m = map[uint64]weak.Pointer[engine]{}
	}
	globalOwners.m[id] = rt.owner
	globalOwners.Unlock()
}

// adopt copies into rt the state that the engine owning the global ID has
// of it and of the elements inside it, as that engine's window showed them
// last. It reports whether there was such an engine and state.
func (rt *engine) adopt(id uint64) bool {
	globalOwners.Lock()
	from := globalOwners.m[id].Value()
	globalOwners.Unlock()
	if from == nil || from == rt || from.states[id] == nil {
		return false
	}
	for sid, s := range from.states {
		if sid != id && !from.inPage(s, id) {
			continue
		}
		d := rt.states[sid]
		if d == nil {
			d = &state{id: sid, born: rt.frame}
			rt.states[sid] = d
		}
		copyMovable(d, s)
	}
	// The focus follows: it goes to the element as it is built (prune),
	// unless another took it.
	for _, f := range []*uint64{&from.focused, &from.lostFocus} {
		if *f != 0 && (*f == id || from.inPage(from.states[*f], id)) {
			rt.lostFocus, *f = *f, 0
		}
	}
	return true
}

// inPage reports whether s is inside the page id: id is on the chain of
// its pages (at most 64 up, as keptAlive walks).
func (rt *engine) inPage(s *state, id uint64) bool {
	for n := 0; s != nil && n < 64; n++ {
		if s.page == id {
			return true
		}
		if s.page == 0 {
			return false
		}
		s = rt.states[s.page]
	}
	return false
}

// copyMovable copies into d what of s does not depend on the window:
// scroll offsets, an editor's text, selection and undo history, and the
// element's locals, with its popup closed. The rest is laid out, cached
// and bound again by d's first frame.
func copyMovable(d, s *state) {
	d.scrollX, d.scrollY = s.scrollX, s.scrollY
	if e := s.editor; e != nil {
		// A composition is committed, and the input not yet published to
		// the app's value is published by the first frame that binds it,
		// as native input is.
		dirty, before := e.nativeDirty, e.nativeValue
		if e.compose != "" && !dirty {
			dirty, before = true, e.buf.string()
		}
		e.commitCompose()
		if d.editor == nil {
			d.editor = newEditor()
		}
		d.editor.nativeDirty, d.editor.nativeValue = dirty, before
		d.editor.buf = e.buf
		d.editor.editorSelection = e.editorSelection.clone()
		d.editor.undo, d.editor.redo = slices.Clone(e.undo), slices.Clone(e.redo)
		d.editor.scrollX = e.scrollX
	}
	if s.locals != nil {
		d.locals = maps.Clone(s.locals)
		if _, ok := d.locals[popupOpen{}]; ok {
			d.locals[popupOpen{}] = new(bool)
		}
	}
}

package widget

// mirror is a value a widget binds to ui. ui writes its bound pointer after
// the view returns, outside mu, so it gets shown, which only the drawing
// thread touches (while building and in ui's callbacks); value is the
// widget's, under mu, for getters and setters.
type mirror[T comparable] struct {
	value, shown T
	dirty        bool // a setter changed value since the last build
}

// set changes the value. Under mu.
func (m *mirror[T]) set(v T) { m.value, m.dirty = v, true }

// bind returns the pointer to hand ui, under mu while building. changed
// reports that ui changed the value without an OnChange, as Tabs clamps
// its choice, so the caller runs its change handler.
func (m *mirror[T]) bind() (p *T, changed bool) {
	if m.dirty {
		m.shown, m.dirty = m.value, false
		return &m.shown, false
	}
	changed = m.shown != m.value
	m.value = m.shown
	return &m.shown, changed
}

// commit takes what the user entered, in ui's OnChange callback, and
// reports whether it changed the value. The user wins over a setter that
// ran since the last build.
func (m *mirror[T]) commit() bool {
	mu.Lock()
	defer mu.Unlock()
	changed := m.value != m.shown
	m.value, m.dirty = m.shown, false
	return changed
}

// committer is a mirror of any type.
type committer interface{ commit() bool }

// onChange returns ui's OnChange callback for m: it commits and runs h,
// if any, on the event goroutine.
func onChange(m committer, h func()) func() {
	return func() {
		if m.commit() && h != nil {
			Post(h)
		}
	}
}

// editedOnce returns ui's OnChange callback for m, a control's value, made
// once and kept in *slot for the life of the control: a closure made in each
// build is an allocation for each control in each frame. It takes the
// control's OnChange (*h) and its binding at the moment of the edit.
func editedOnce[T comparable](slot *func(), m *mirror[T], b *binding, h *func()) func() {
	if *slot == nil {
		*slot = func() {
			mu.Lock()
			fn := edited(m, b, *h)
			mu.Unlock()
			fn()
		}
	}
	return *slot
}

package ui

type actionKind uint8

const (
	actionClick actionKind = iota
	actionChange
	actionSubmit
	actionShortcut
	actionWindowShortcut
	actionEvent
)

type action struct {
	element Element
	handle  *Handle
	window  *Context
	kind    actionKind
	mods    Modifiers
	key     Key
	fn      func()
	// event is what an actionEvent waits for.
	event tableEvent
	ready bool
}

func (e Element) on(kind actionKind, mods Modifiers, key Key, fn func()) Element {
	if n := e.node(); n != nil && fn != nil {
		if kind == actionClick {
			n.flags |= flagClickable
		}
		n.c.rt.actions = append(n.c.rt.actions, action{element: e, kind: kind, mods: mods, key: key, fn: fn})
	}
	return e
}

// OnClick runs after construction and bound-value input.
func (e Element) OnClick(fn func()) Element { return e.on(actionClick, 0, 0, fn) }

// OnChange runs after input changes the bound value, before rebuilding the
// view. The callback can commit a derived value local to this build pass.
func (e Element) OnChange(fn func()) Element { return e.on(actionChange, 0, 0, fn) }

// OnSubmit runs after bound-value input handles a submission gesture,
// before rebuilding the view.
func (e Element) OnSubmit(fn func()) Element { return e.on(actionSubmit, 0, 0, fn) }

func (e Element) onEvent(ev tableEvent, fn func()) Element {
	if n := e.node(); n != nil && fn != nil {
		n.c.rt.actions = append(n.c.rt.actions, action{element: e, kind: actionEvent, event: ev, fn: fn})
	}
	return e
}

// OnSelectionChange runs after input changes the rows chosen in a List or
// a Table, before rebuilding the view.
func (e Element) OnSelectionChange(fn func()) Element { return e.onEvent(eventSelection, fn) }

// OnSortChange runs after a click on a header changes a Table's sort,
// before rebuilding the view.
func (e Element) OnSortChange(fn func()) Element { return e.onEvent(eventSort, fn) }

// OnColumnsChange runs after the user resizes, moves or fits a column of
// a Table, before rebuilding the view: save ListState.Columns there.
func (e Element) OnColumnsChange(fn func()) Element { return e.onEvent(eventColumns, fn) }

// OnCheckedChange runs after a checkbox of a Table's rows or header
// changes ListState.Checked, before rebuilding the view.
func (e Element) OnCheckedChange(fn func()) Element { return e.onEvent(eventChecked, fn) }

// OnActivate runs with the row of a List or a Table double-clicked, or
// chosen as Enter is pressed, before rebuilding the view.
func (e Element) OnActivate(fn func(row int)) Element {
	if n := e.node(); n != nil && fn != nil {
		rt := n.c.rt
		e.onEvent(eventActivated, func() {
			if n := e.nodeFor(rt); n != nil {
				fn(n.st.activated)
			}
		})
	}
	return e
}

// OnShortcut declares a key handler within the element's focus subtree.
func (e Element) OnShortcut(mods Modifiers, key Key, fn func()) Element {
	return e.on(actionShortcut, mods, key, fn)
}

// OnShortcut declares a handler in the current context's parent scope.
func (c *Context) OnShortcut(mods Modifiers, key Key, fn func()) {
	if raw := c.build(); raw != nil && fn != nil {
		if raw.parent != raw.root {
			wrapElement(raw.parent).OnShortcut(mods, key, fn)
			return
		}
		raw.rt.actions = append(raw.rt.actions, action{window: c, kind: actionWindowShortcut, mods: mods, key: key, fn: fn})
	}
}
func (rt *engine) actionNode(a action) *node {
	if a.handle != nil {
		b := a.handle.binding(rt, false)
		if b == nil || b.closed {
			return nil
		}
		return b.element.lookup()
	}
	return a.element.nodeFor(rt)
}
func (rt *engine) runNoticeActions() {
	// Select every observer before marking notices delivered, so several
	// callbacks on one element all run once for the same input.
	for i := range rt.actions {
		a := &rt.actions[i]
		a.ready = false
		if rt.closing {
			return
		}
		if a.kind != actionChange && a.kind != actionSubmit && a.kind != actionEvent {
			continue
		}
		n := rt.actionNode(*a)
		if n == nil || n.disabled() {
			continue
		}
		a.ready = a.kind == actionChange && n.st.changed && !n.st.changeDelivered ||
			a.kind == actionSubmit && n.st.submitted && !n.st.submitDelivered ||
			a.kind == actionEvent && n.st.events&a.event != 0 && n.st.eventsDelivered&a.event == 0
	}
	for _, a := range rt.actions {
		if !a.ready {
			continue
		}
		n := rt.actionNode(a)
		switch a.kind {
		case actionChange:
			n.st.changeDelivered = true
		case actionSubmit:
			n.st.submitDelivered = true
		default:
			n.st.eventsDelivered |= a.event
		}
	}
	for _, a := range rt.actions {
		if rt.closing {
			return
		}
		if a.ready {
			rt.consumed = true
			a.fn()
		}
	}
}
func (rt *engine) runActions() {
	for _, a := range rt.actions {
		if rt.closing {
			return
		}
		if a.kind == actionChange || a.kind == actionSubmit || a.kind == actionEvent {
			continue
		}
		handled := false
		if a.kind == actionWindowShortcut {
			handled = rt.c.Shortcut(a.mods, a.key)
		} else if n := rt.actionNode(a); n != nil && !n.disabled() {
			switch a.kind {
			case actionClick:
				handled = n.Clicked()
			case actionShortcut:
				handled = n.Shortcut(a.mods, a.key)
			}
		}
		if handled {
			rt.consumed = true
			a.fn()
		}
	}
}

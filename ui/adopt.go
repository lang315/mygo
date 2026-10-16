package ui

import (
	"sync"
	"weak"
)

// owners maps the ID of each element keyed globally to the engine that
// built it last, among the engines of one app: the windows of a process
// share one, each Tester has its own. Another engine building the element
// takes over its state: a panel moved between windows keeps its scroll
// offsets, text and selection, and app locals.
type owners struct {
	sync.Mutex
	m map[uint64]weak.Pointer[engine]
}

// windowOwners are the owners of the windows' engines.
var windowOwners = &owners{}

// note records rt as the engine building the global ID.
func (o *owners) note(id uint64, rt *engine) {
	o.Lock()
	if o.m == nil {
		o.m = map[uint64]weak.Pointer[engine]{}
	}
	o.m[id] = rt.owner
	o.Unlock()
}

// drop forgets rt as the owner of the ID, as rt forgets its state.
func (o *owners) drop(id uint64, rt *engine) {
	o.Lock()
	if p, ok := o.m[id]; ok && p == rt.owner {
		delete(o.m, id)
	}
	o.Unlock()
}

// owner returns the engine that built the global ID last, if it is alive.
func (o *owners) owner(id uint64) *engine {
	o.Lock()
	defer o.Unlock()
	p, ok := o.m[id]
	if !ok {
		return nil
	}
	e := p.Value()
	if e == nil {
		delete(o.m, id) // its window is gone
	}
	return e
}

// adopt moves into rt the state that another engine of the app keeps of
// the global ID and of the elements inside it, out of sight (KeepAlive),
// as that engine's window showed them last. An engine still showing the
// element keeps its state: rt starts afresh, and each shows its own. It
// reports whether there was such a state.
func (rt *engine) adopt(id uint64) bool {
	from := rt.owners.owner(id)
	if from == nil || from == rt {
		return false
	}
	top := from.states[id]
	if top == nil || top.seen == from.frame && top.pass == from.pass {
		return false
	}
	inside := func(s *state) bool { return s.page == id }
	for sid, s := range from.states {
		if sid != id && !from.upPages(s, inside) {
			continue
		}
		d := rt.states[sid]
		if d == nil {
			d = &state{id: sid, born: rt.frame}
			rt.states[sid] = d
		}
		moveState(d, s)
		from.dropState(sid, s)
	}
	// The focus follows: it goes to the element as it is built (prune),
	// unless another took it.
	if f := from.lostFocus; f != 0 && rt.states[f] != nil {
		rt.lostFocus, from.lostFocus = f, 0
	}
	return true
}

// moveState moves into d what of s does not depend on the window: scroll
// offsets, an editor with its text, selection and undo history, and the
// element's locals. s is kept out of sight, so its composition is
// committed and its popup closed (hideKept). The rest is laid out, cached
// and bound again by d's first frame; s is dropped after.
func moveState(d, s *state) {
	d.scrollX, d.scrollY = s.scrollX, s.scrollY
	if e := s.editor; e != nil {
		e.client, e.layout, e.area = nil, nil, nil
		for i := range e.buf.paras { // laid out at the other window's width and scale
			e.buf.paras[i].layout, e.buf.paras[i].h = nil, 0
		}
		d.editor = e
	}
	d.locals = s.locals
}

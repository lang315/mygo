package ui

// The queries that read where an element is, or whether it shows (Bounds,
// LayoutBounds, LocalToWindow, WindowToLocal, InView), read the layout of the
// frame before. A row that asks is kept only while each answer is the one a
// build would get now: the list records (element, query, answer) while the
// row is built, and keep asks again of the states.

// layoutKind tells the queries apart.
type layoutKind uint8

const (
	lkBounds layoutKind = iota
	lkLayout
	lkToWindow
	lkToLocal
	lkInView
)

// maxLayoutReads is how many different answers a row records; a row that
// asks for more is built in every frame.
const maxLayoutReads = 8

// layoutVal is the answer of a query.
type layoutVal struct {
	a, b, c, d float32
	ok         bool
}

func (v layoutVal) rect() Rect { return Rect{v.a, v.b, v.c, v.d} }

// layoutRead is a query of a row's callback with the answer it got.
type layoutRead struct {
	id     uint64
	kind   layoutKind
	x, y   float32
	answer layoutVal
}

// layoutOf is the answer of query kind (with the point x, y) about the element
// whose state is s, for the engine rt in its frame: what a build gets. A
// missing state answers as an element the frame before lacked.
func layoutOf(rt *engine, s *state, kind layoutKind, x, y float32) (v layoutVal) {
	if s == nil || rt == nil || rt.closed {
		return v
	}
	switch kind {
	case lkBounds:
		r := transformRect(s.world, Rect{s.x, s.y, s.w, s.h})
		v = layoutVal{a: r.X, b: r.Y, c: r.W, d: r.H}
	case lkLayout:
		v = layoutVal{a: s.x, b: s.y, c: s.w, d: s.h}
	case lkToWindow:
		v.a, v.b = s.world.Point(s.x+x, s.y+y)
	case lkToLocal:
		if m, ok := s.world.Inverse(); ok {
			px, py := m.Point(x, y)
			v = layoutVal{a: px - s.x, b: py - s.y, ok: true}
		}
	case lkInView:
		v.ok = s.seen+1 >= rt.frame && s.vw > 0 && s.vh > 0
	}
	return v
}

// readsLayout answers a query about e and notes the answer for the row being
// built, which is linked later only while the answer is the same.
func (e *node) readsLayout(kind layoutKind, x, y float32) layoutVal {
	if e == nil || e.c == nil || e.c.rt == nil {
		return layoutVal{}
	}
	var s *state
	if e.hasState() {
		s = e.st
	}
	v := layoutOf(e.c.rt, s, kind, x, y)
	if rb := e.c.row; rb != nil {
		rb.noteLayout(layoutRead{id: e.id, kind: kind, x: x, y: y, answer: v})
	}
	return v
}

// noteLayout records r for the row, once, and gives up on a row that asks too
// many things: it is built in every frame.
func (rb *rowBuild) noteLayout(r layoutRead) {
	for _, o := range rb.layouts {
		if o.id == r.id && o.kind == r.kind && o.x == r.x && o.y == r.y {
			return
		}
	}
	if len(rb.layouts) >= maxLayoutReads {
		rb.outside = true
		return
	}
	rb.layouts = append(rb.layouts, r)
}

// sameLayouts reports whether every query the row asked would get the answer
// it got, from the states now.
func (rt *engine) sameLayouts(reads []layoutRead) bool {
	for i := range reads {
		r := &reads[i]
		if layoutOf(rt, rt.states[r.id], r.kind, r.x, r.y) != r.answer {
			return false
		}
	}
	return true
}

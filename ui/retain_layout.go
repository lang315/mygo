package ui

import (
	"slices"

	"github.com/egoist/mygo/internal/scene"
)

// A kept row (retain.go) that stays where it was, at the same width and with
// the same surroundings, keeps the results of its last layout: its nodes are
// not zeroed when it is linked, layoutList neither compares nor lays it out
// (settle), and place finds it where it was (keptPlace). Nothing else of the
// frame reads the row's nodes before commit, which walks them as it walks a
// built row.
//
// The layout of a row is kept only when every input of it is as in the last
// layout (settle): the layout it was recorded in was the one before, its
// width, the list's layout flags and the text styles the row inherits are
// the same, the direction of its root is, no reveal asks for an element in
// it, and its root is the one its state records. Each has a test in
// retain_layout_test.go that flips it alone.

// rowLay is what a kept row keeps of its last layout.
type rowLay struct {
	at     uint64  // the layout whose results the nodes hold, 0 for none
	fullAt uint64  // the last layout that compared the row (sameRow)
	setAt  uint64  // the layout that decided about the nodes (settle, late, moved)
	lw     float32 // the width the row was laid out with
	hm     float32 // and the height
	ax, ay float32 // where place put the root
	rlay   uint8   // the root's lay bits after the layout
	rdir   uint8   // the root's direction, which an ancestor's Direction sets again after the link
	// What commit left of the row, which it appends again while commitEnv is
	// the same (commitRow).
	cok    bool
	env    commitEnv
	hits   []hit
	labels []labelNode
}

// commitEnv is what the commit of a row reads outside it.
type commitEnv struct {
	clip      Rect
	hidden    bool
	parentDis bool
	scope     focusScope
	page      uint64
	world     scene.Affine
}

// zero makes the nodes what an element just built holds before layout.
func (k *rowKeep) zero() {
	for j := range k.nodes {
		n := &k.nodes[j]
		// What layout and commit write is zero, as in an element just built: a
		// row laid out writes it, and one that is not (below the view) holds
		// nothing of the frame it was copied in. The measures of text and
		// heights (nmeasure, leafOK, intrOK) are of one frame, and depend on
		// the text style the row inherits, which may have changed; sameRow
		// compares the row by the record in its state (lay), which stays.
		n.lay, n.tl, n.nmeasure, n.leafOK, n.intrOK = 0, nil, 0, false, 0
		n.x, n.y, n.w, n.h, n.contentW, n.contentH, n.ink, n.world = 0, 0, 0, 0, 0, 0, [4]float32{}, scene.Affine{}
	}
	k.lay.at, k.lay.cok = 0, false
}

// unlink clears the pointer of the root's state to the record, when it is
// this one's.
func (k *rowKeep) unlink() {
	if len(k.states) > 0 && k.states[0] != nil && k.states[0].krec == k {
		k.states[0].krec = nil
	}
}

// rewind lets the layouts skipped count for the states of the row: the next
// layout compares the row with the one before it.
func (k *rowKeep) rewind(n uint64) {
	if k.lay.fullAt == 0 || k.lay.fullAt == n-1 {
		return
	}
	for _, st := range k.states {
		if st.lay.layout == k.lay.fullAt {
			st.lay.layout = n - 1
		}
	}
}

// revealsIn reports whether a reveal asks for an element of the row: reveal
// sums the relative boxes of the elements, and those of a row kept hold
// window coordinates.
func (rt *engine) revealsIn(rec *rowKeep) bool {
	if len(rt.revealIDs) == 0 {
		return false
	}
	for _, st := range rec.states {
		if slices.Contains(rt.revealIDs, st.id) {
			return true
		}
	}
	return false
}

// keptRec returns the record of the kept row whose root is e, nil for a row
// built.
func keptRec(e *node) *rowKeep {
	if rec := e.st.krec; rec != nil && rec.root == e {
		return rec
	}
	return nil
}

// settle decides, in layoutList, whether the layout of the kept row r is kept
// as it was (and then its nodes are not compared nor laid out), or the row is
// laid out as a row built.
func (f *listFrame) settle(list *node, r *listRow, cw float32) bool {
	rec, rt := keptRec(r.e), list.c.rt
	if rec == nil {
		return false
	}
	k := &rec.lay
	k.setAt = rt.layouts
	if k.at != 0 && k.at+1 == rt.layouts && k.lw == cw && list.lay&layRows != 0 && !rt.noLayoutCache &&
		k.rlay&layReuse != 0 && uint8(r.e.dir) == k.rdir && inheritsPaint != nil && inheritsPaint(list) &&
		!rt.revealsIn(rec) && r.e.st.krec == rec {
		r.e.lay |= layKept
		return true
	}
	rec.rewind(rt.layouts)
	rec.zero()
	k.fullAt = rt.layouts
	return false
}

// keptPlace is place for the root of a kept row whose layout was kept: it
// stays where it is, or is laid out now.
func keptPlace(e *node, x, y float32) bool {
	rec := e.st.krec
	if rec == nil || rec.root != e {
		e.lay &^= layKept
		return false
	}
	rt := e.c.rt
	if e.x+x == rec.lay.ax && e.y+y == rec.lay.ay {
		if rt.layoutCheck {
			return checkKeptPlace(e, rec, x, y)
		}
		e.x, e.y = rec.lay.ax, rec.lay.ay
		e.lay |= layPlaced
		rt.rowsLaidKept++
		return true
	}
	// Moved: laid out now, as if layoutList had.
	e.lay &^= layKept
	rec.rewind(rt.layouts)
	rx, ry := e.x, e.y // relative to the parent, which place adds
	rec.zero()
	e.x, e.y = rx, ry
	rec.lay.fullAt = rt.layouts
	sameRow(e.parent, e)
	layoutBox(e, rec.lay.lw, rec.lay.hm)
	return false
}

// heldRow reports whether c is a kept row whose layout results the nodes
// still hold (not zeroed).
func heldRow(c *node) bool {
	rec := c.st.krec
	return rec != nil && rec.root == c && rec.lay.at != 0
}

// commitRow commits the root of a kept row: by the record when it is where it
// was and commit reads the same around it, or by walking it, recording it.
func (rt *engine) commitRow(e *node, clip Rect, hidden bool) {
	rec := e.st.krec
	if rec == nil || rec.root != e {
		rt.walkRow(e, clip, hidden)
		return
	}
	k := &rec.lay
	if k.setAt != rt.layouts {
		rec.zero() // no layout of this frame saw the row: it holds nothing
	}
	env := commitEnv{clip: clip, hidden: hidden || rt.commitHidden, scope: rt.commitScope, page: rt.commitPage}
	if p := e.parent; p != nil {
		env.parentDis, env.world = p.st.flags&flagDisabled != 0, p.world
	}
	// zero clears cok whenever layout ran on the row, so a record that is set
	// is the layout of the last frame, kept (settle) and placed where it was.
	const both = layKept | layPlaced
	if k.cok && k.env == env {
		if rt.layoutCheck {
			rec.checkReplay(e, clip, hidden)
		} else {
			rt.hits = append(rt.hits, k.hits...)
			rt.labels = append(rt.labels, k.labels...)
		}
		k.at = rt.layouts
		e.lay &^= both
		rt.rowsCommitReplayed++
		return
	}
	placed := e.lay&both == both || e.lay&layKept == 0
	e.lay &^= both
	h0, l0, f0, p0, g0, m0 := len(rt.hits), len(rt.labels), len(rt.focusOrder), len(rt.posed), len(rt.groups), rt.modal
	rt.walkRow(e, clip, hidden)
	// placed: place reached the row (layout.go), the hook there is not ours.
	// !env.hidden: a hidden row is not placed, and zero covers it only through
	// setAt. A focusable or an animated pose never reaches here (retainable,
	// state.pending), so the focusOrder and posed conditions guard a row that
	// is not held today.
	if placed && !env.hidden && e.lay&layReuse != 0 {
		k.ax, k.ay, k.rlay, k.rdir, k.at = e.x, e.y, e.lay, uint8(e.dir), rt.layouts
		if len(rt.focusOrder) == f0 && len(rt.posed) == p0 && len(rt.groups) == g0 && rt.modal == m0 {
			k.hits = append(k.hits[:0], rt.hits[h0:]...)
			k.labels = append(k.labels[:0], rt.labels[l0:]...)
			k.env, k.cok = env, true
		} else {
			k.cok = false
		}
	} else {
		k.at, k.cok = 0, false
	}
}

// layoutCheck (tests only) does the work a record replaces, as replayCheck
// does for painting: keptPlace lays out the row it keeps and commitRow walks
// the row it replays, and each counts in layoutMismatch what differs.

// nodeBox is what layout writes of a node.
type nodeBox struct{ x, y, w, h, contentW, contentH float64 }

func boxesOf(rec *rowKeep) []nodeBox {
	b := make([]nodeBox, len(rec.nodes))
	for j := range rec.nodes {
		n := &rec.nodes[j]
		b[j] = nodeBox{float64(n.x), float64(n.y), float64(n.w), float64(n.h), float64(n.contentW), float64(n.contentH)}
	}
	return b
}

// stateSnap is what commit writes of a state.
type stateSnap struct {
	x, y, w, h, vx, vy, vw, vh, cx, cy, cw, ch float32
	flags                                      uint32
	world                                      scene.Affine
	clips                                      bool
	clipRect                                   Rect
	rtl                                        bool
	page                                       uint64
	scope                                      uint64
	anchor                                     uint64
	cursor                                     Cursor
	tip                                        bool
	contentW, contentH                         float64
}

func snapOf(st *state) stateSnap {
	return stateSnap{st.x, st.y, st.w, st.h, st.vx, st.vy, st.vw, st.vh, st.cx, st.cy, st.cw, st.ch, uint32(st.flags),
		st.world, st.clips, st.clipRect, st.rtl, st.page, st.scope, st.anchor, st.cursor, st.tip, st.contentW, st.contentH}
}

// checkKeptPlace is the stay path of keptPlace with the layout done anyway:
// the row is laid out and placed as a row built, and the nodes are compared
// with those kept. The row is then kept as it was, so that commitRow replays.
func checkKeptPlace(e *node, rec *rowKeep, x, y float32) bool {
	rt := e.c.rt
	saved, rx, ry := rec.lay, e.x, e.y // the root's box is relative to its parent until place
	e.x, e.y = saved.ax, saved.ay
	want := boxesOf(rec)
	rec.rewind(rt.layouts)
	rec.zero()
	e.x, e.y = rx, ry
	sameRow(e.parent, e)
	layoutBox(e, saved.lw, saved.hm)
	e.lay &^= layKept
	place(e, x, y)
	if !slices.Equal(want, boxesOf(rec)) {
		rt.layoutMismatch++
	}
	rec.lay = saved
	e.lay |= layKept | layPlaced
	rt.rowsLaidKept++
	return true
}

// checkReplay appends what the record holds, and compares it with what
// walking the row appends and writes to its states, which stand.
func (k *rowKeep) checkReplay(e *node, clip Rect, hidden bool) {
	rt := e.c.rt
	before := make([]stateSnap, len(k.states))
	for i, st := range k.states {
		before[i] = snapOf(st)
	}
	h0, l0 := len(rt.hits), len(rt.labels)
	rt.walkRow(e, clip, hidden)
	bad := !slices.Equal(rt.hits[h0:], k.lay.hits) || !slices.Equal(rt.labels[l0:], k.lay.labels)
	for i, st := range k.states {
		bad = bad || before[i] != snapOf(st)
	}
	if bad {
		rt.layoutMismatch++
	}
}

// walkRow commits the root of a kept row and what is in it, as commitElement
// does a row built.
func (rt *engine) walkRow(e *node, clip Rect, hidden bool) {
	saved := rt.rowWalk
	rt.rowWalk = e
	rt.commitElement(e, clip, hidden)
	rt.rowWalk = saved
}

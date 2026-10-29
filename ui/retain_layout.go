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
	// A kept row without layPaint was compared (sameRow) with a layout that
	// painted differently, and settle keeps it from then on without comparing
	// it again: it would paint anew for good. It is laid out again, and so
	// compared, up to paintTries layouts running, until it has layPaint or
	// is volatile (which never has). retryAt is the last layout that counted.
	tries   uint8
	retryAt uint64
	rdir    uint8 // the root's direction, which an ancestor's Direction sets again after the link
	// What commit left of the row, which it appends again while commitEnv is
	// the same (commitRow).
	// scrolled are the offsets of the row's scroll containers (rowKeep.scrollers)
	// in the layout at: the offset of one changes with no build of the row (a
	// wheel over it while the pointer is pressed elsewhere, a drag of its
	// bar), and place applies it, which a kept layout skips.
	scrolled []scrollAt
	// relOK tells that rowKeep.rel holds the relative boxes of the layout the
	// nodes hold (P4a).
	relOK  bool
	cok    bool
	env    commitEnv
	hits   []hit
	labels []labelNode
}

// paintTries is the number of layouts a kept row without layPaint is laid out
// in before it is kept as it is: the first compares what it paints with the
// layout before, and finds it different; the second finds it the same.
const paintTries = 2

// scrollAt is the offset of a scroll container.
type scrollAt struct{ x, y float64 }

// noteScrolls records the offsets of the row's scroll containers.
func (k *rowKeep) noteScrolls() {
	k.lay.scrolled = k.lay.scrolled[:0]
	for _, j := range k.scrollers {
		st := k.nodes[j].st
		k.lay.scrolled = append(k.lay.scrolled, scrollAt{st.scrollX, st.scrollY})
	}
}

// scrollsSame reports whether the scroll containers of the row are at the
// offsets the layout kept was placed with.
func (k *rowKeep) scrollsSame() bool {
	for i, j := range k.scrollers {
		st := k.nodes[j].st
		if i >= len(k.lay.scrolled) || k.lay.scrolled[i] != (scrollAt{st.scrollX, st.scrollY}) {
			return false
		}
	}
	return true
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
	k.lay.at, k.lay.cok, k.lay.relOK = 0, false, false
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
	// A row without layPaint is laid out again (paintTries layouts), so that it
	// may get it; a second settle in the same layout does not count.
	painted := k.rlay&layPaint != 0 || k.tries >= paintTries
	if !painted && k.retryAt != rt.layouts && k.at != 0 {
		k.tries++
		k.retryAt = rt.layouts
	}
	if painted && k.at != 0 && k.at+1 == rt.layouts && k.lw == cw && list.lay&layRows != 0 && !rt.noLayoutCache &&
		k.rlay&layReuse != 0 && uint8(r.e.dir) == k.rdir && inheritsPaint != nil && inheritsPaint(list) &&
		!rt.revealsIn(rec) && r.e.st.krec == rec && rec.scrollsSame() {
		r.e.lay |= layKept
		r.e.lay &^= layFresh // laid out by the pass before, kept by this one (relayoutList)
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
		e.lay &^= layKept | layFresh
		return false
	}
	rt := e.c.rt
	if e.lay&layFresh != 0 {
		// Laid out in this layout: the boxes of the elements inside are
		// relative to their parents, and final. Save them before place.
		e.lay &^= layFresh
		rec.saveRel()
		return false
	}
	if e.x+x == rec.lay.ax && e.y+y == rec.lay.ay {
		if rt.layoutCheck {
			return checkKeptPlace(e, rec, x, y)
		}
		e.x, e.y = rec.lay.ax, rec.lay.ay
		e.lay |= layPlaced
		rt.rowsLaidKept++
		return true
	}
	if rec.lay.relOK {
		// Moved: the layout is kept, and place puts the elements where they
		// go from their relative boxes, as it did for the layout it kept.
		if rt.layoutCheck {
			return checkShift(e, rec, x, y)
		}
		rec.moveRow(e, x, y)
		return false
	}
	// Moved, and the relative boxes are not known: laid out now, as if
	// layoutList had.
	e.lay &^= layKept
	rec.layOutAgain(e)
	rec.lay.ax, rec.lay.ay = e.x+x, e.y+y
	return false
}

// moveRow gives a kept row that moved its relative boxes again, and notes
// where place is putting it: the next place of the same layout (an element that
// lays the list out and places it again, as a transition of a Match does) finds
// the row where this one left it, or moves it again.
func (k *rowKeep) moveRow(e *node, x, y float32) {
	k.restoreRel()
	e.lay &^= layKept
	k.lay.cok = false // the hits and labels are where the row was
	k.lay.ax, k.lay.ay = e.x+x, e.y+y
	e.c.rt.rowsShifted++
}

// layOutAgain lays the row out as layoutList does a row built, and saves its
// relative boxes. Rows reach it with relOK false, which no row laid out by
// layoutList does (the place hook saved them): the save here keeps a row that
// somehow missed the hook from being laid out at every move.
func (k *rowKeep) layOutAgain(e *node) {
	rt := e.c.rt
	k.rewind(rt.layouts)
	rx, ry := e.x, e.y // relative to the parent, which place adds
	k.zero()
	e.x, e.y = rx, ry
	k.lay.fullAt = rt.layouts
	sameRow(e.parent, e)
	layoutBox(e, k.lay.lw, k.lay.hm)
	k.saveRel()
}

// saveRel keeps the boxes of the elements inside the row, which layout left
// relative to their parents (the root's is set by layoutList each frame).
func (k *rowKeep) saveRel() {
	k.rel = k.rel[:0]
	for j := 1; j < len(k.nodes); j++ {
		k.rel = append(k.rel, relXY{k.nodes[j].x, k.nodes[j].y})
	}
	k.lay.relOK = true
}

// restoreRel gives the elements inside the row their relative boxes again.
func (k *rowKeep) restoreRel() {
	for j := 1; j < len(k.nodes); j++ {
		k.nodes[j].x, k.nodes[j].y = k.rel[j-1].x, k.rel[j-1].y
	}
}

// relXY is the box of an element relative to its parent, before place.
type relXY struct{ x, y float32 }

// forgetRowMeasures forgets the measures of the text in the rows held of the
// List list, whose text styles (its own and those it inherits) are not as they
// were when the rows were measured: the box around a List sized by its content
// is as wide as the rows' max-content width, which is measured before the list
// is laid out, and a row held keeps the measures of the last frame (the
// elements of a row linked are not zeroed, settle decides). A row not held was
// zeroed when it was linked.
func forgetRowMeasures(list *node) {
	var clear func(n *node)
	clear = func(n *node) {
		n.nmeasure, n.leafOK, n.intrOK = 0, false, 0
		for c := n.first; c != nil; c = c.next {
			clear(c)
		}
	}
	for c := list.first; c != nil; c = c.next {
		if c.listRow && heldRow(c) {
			clear(c)
		}
	}
}

// forgetKeptRows makes the kept rows in the subtree of e lay out again, and
// commit again, the next time: a transition moved their nodes after place
// (shift), and the record still says where layout put them, so a later place
// of the same layout (the element of an outer Match lays the list out and
// places it again) would find a row where it was and leave its elements where
// shift put them. shift is shared with drags and effects, which move
// nodes and move them back, so the record is left to the one that moves them
// for good.
func forgetKeptRows(e *node) {
	if e.listRow {
		if rec := keptRec(e); rec != nil {
			rec.lay.at, rec.lay.cok = 0, false
		}
	}
	for c := e.first; c != nil; c = c.next {
		forgetKeptRows(c)
	}
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
		if e.lay&layPaint != 0 {
			k.tries = 0
		}
		rec.noteScrolls()
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

// rowSnap is what laying a row out writes besides its boxes: the nodes, the
// layout records of the states, the record of the row, and the engine's
// counters. layoutCheck lays a row out for real after the code it checks
// has done its part, then puts all of it back, so that what happens next is
// what happens with the check off (a check that left the record zeroed let
// the next place of the same layout take a path the real code does not).
type rowSnap struct {
	nodes  []node
	lays   []layoutRecord
	rel    []relXY
	lay    rowLay
	laySeq uint64
	comp   uint32
	same   int
}

func snapRow(rt *engine, rec *rowKeep) rowSnap {
	s := rowSnap{nodes: slices.Clone(rec.nodes), rel: slices.Clone(rec.rel), lay: rec.lay, laySeq: rt.laySeq, comp: rt.computed, same: rt.paintSameRows}
	s.lay.hits, s.lay.labels, s.lay.scrolled = slices.Clone(rec.lay.hits), slices.Clone(rec.lay.labels), slices.Clone(rec.lay.scrolled)
	for _, st := range rec.states {
		s.lays = append(s.lays, st.lay)
	}
	return s
}

func (s rowSnap) restore(rt *engine, rec *rowKeep) {
	copy(rec.nodes, s.nodes)
	rec.rel, rec.lay = s.rel, s.lay
	for i, st := range rec.states {
		st.lay = s.lays[i]
	}
	rt.laySeq, rt.computed, rt.paintSameRows = s.laySeq, s.comp, s.same
}

// checkKeptPlace is the stay path of keptPlace, and checkShift the moved path,
// with the layout done anyway after the real work: the row is laid out and
// placed as a row built, and the boxes are compared with ==. The state the
// real path leaves is put back, nodes included, which equal the fresh ones
// when nothing differs.
func checkKeptPlace(e *node, rec *rowKeep, x, y float32) bool {
	rt := e.c.rt
	rx, ry := e.x, e.y // the root's box is relative to its parent until place
	e.x, e.y = rec.lay.ax, rec.lay.ay
	e.lay |= layPlaced
	rt.rowsLaidKept++
	checkAgainstLayout(e, rec, x, y, rx, ry)
	return true
}

func checkShift(e *node, rec *rowKeep, x, y float32) bool {
	rx, ry := e.x, e.y
	rec.moveRow(e, x, y)
	place(e, x, y)
	checkAgainstLayout(e, rec, x, y, rx, ry)
	return true
}

// checkAgainstLayout compares the nodes of the row as the real path left them
// with a layout of the row, and puts the real state back.
func checkAgainstLayout(e *node, rec *rowKeep, x, y, rx, ry float32) {
	rt := e.c.rt
	want := boxesOf(rec)
	snap := snapRow(rt, rec)
	e.x, e.y = rx, ry
	e.lay &^= layKept | layPlaced
	rec.layOutAgain(e)
	place(e, x, y)
	if !slices.Equal(want, boxesOf(rec)) {
		rt.layoutMismatch++
	}
	snap.restore(rt, rec)
}

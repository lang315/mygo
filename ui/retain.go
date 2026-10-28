package ui

import (
	"math"
	"slices"

	"github.com/egoist/mygo/internal/scene"
)

// A List or Table whose app sets ListState.RowVersion keeps the rows it built
// in the last frame, and links them again in the next instead of building
// them, as long as nothing that a build of them reads has changed: the row's
// key and version, the choice and checks of it and its neighbours, the theme,
// and input and requests of the states inside it (state.pending). A kept row
// is a copy of the row's subtree on the heap, taken after the last pass of a
// frame was committed, in storage the row's record reuses. Layout, commit and
// painting run over it as over a built row.
//
// Stale UI is what can go wrong here, so a row is kept only when every check
// of rowKeep.ok passes, and each check has a test (retain_test.go).

// rowKeep is the record of a row kept: its copy, and what it was built with.
type rowKeep struct {
	i        int
	version  uint64
	themeGen uint64
	theme    *Theme // the list's, when the row was built
	parent   uint64 // the id of the element holding the rows: theirs derive from it
	sig      uint64 // what a Table builds the rows with besides their data (rowSignature)
	dir      uint8  // the direction the holding element passes on, which an element copies as it is built
	// lastFrame is the last frame the record was captured or linked in, and
	// nothing else sets it: a row built in a frame without being captured (it
	// was hovered, focused, pending) does not renew an older record, which
	// capture releases instead, and a record that skipped a frame is not
	// linked (its states were pruned and recycled). Otherwise a record older
	// than the last frame the row was built in could match again once the
	// inputs settled, and show a state the row has not had since.
	lastFrame uint64
	mark      uint64 // the capture of the list that last saw the row built or linked
	depth     int
	chosen    [3]bool // of the rows above, at and below it
	checked   bool
	root      *node
	nodes     []node   // the copy, root first, in the order of a walk
	states    []*state // those of the nodes, which each pass stamps as seen
}

// actingIDs gathers, once for a pass, the ids of the elements the engine is
// acting on or waiting for: the pointer's chain, the press, the focus, a
// menu, a drag, a tooltip. A row holding one of them is built.
func (rt *engine) actingIDs() {
	if rt.actFrame == rt.frame && rt.actPass == rt.pass+1 {
		return
	}
	rt.actFrame, rt.actPass = rt.frame, rt.pass+1
	b := append(rt.acting[:0], rt.hover...)
	b = append(b, rt.focused, rt.lostFocus, rt.pressDragger, rt.dropOver,
		rt.menu.asked, rt.menu.chosen, rt.menu.text, rt.menu.chosenText, rt.menu.release, rt.menu.releaseText,
		rt.tips.shown, rt.tips.hover, rt.tips.focus)
	if rt.pressed != nil {
		b = append(b, rt.pressed.id)
	}
	if d := rt.drag; d != nil {
		b = append(b, d.src, d.over)
	}
	b = append(b, rt.clickLater...)
	rt.acting = b
	rt.actBloom = [4]uint64{}
	for _, id := range b {
		rt.actBloom[id>>6&3] |= 1 << (id & 63)
	}
}

// isActing reports whether the engine acts on or waits for the element id.
func (rt *engine) isActing(id uint64) bool {
	if rt.actBloom[id>>6&3]&(1<<(id&63)) == 0 {
		return false
	}
	return id != 0 && slices.Contains(rt.acting, id)
}

// ok reports whether row i, of key and states, may be kept (captured) or
// linked again: no input or request is pending in it, the engine acts on none
// of its elements, and it is an ordinary row. It does not compare what the row
// was built with (see keep).
func (f *listFrame) ok(i int, states []*state) bool {
	rt, s := f.c.rt, f.s
	if rt.noRetain || rt.insp.open || f.isHeader(i) || s.dragSel.on {
		return false
	}
	if sel := s.cursor(); sel != nil && s.Selection != nil && *sel == i {
		return false // the row the keys are on has a focus ring of its own (DrawOver)
	}
	if s.editAsked && s.editKey == f.key(i) {
		return false // the keys asked this row to edit its text: its build starts it
	}
	rt.actingIDs()
	for _, st := range states {
		if st.pending() || rt.isActing(st.id) {
			return false
		}
	}
	return true
}

// chosenAround is the choice of rows i-1, i and i+1: rounded corners of a
// block of chosen rows read the neighbours.
func (f *listFrame) chosenAround(i int) (c [3]bool) {
	if f.s.cursor() == nil {
		return
	}
	if i > 0 && !f.isHeader(i-1) {
		c[0] = f.chosen(i-1, f.key(i-1))
	}
	c[1] = f.chosen(i, f.key(i))
	if i < f.n-1 && !f.isHeader(i+1) {
		c[2] = f.chosen(i+1, f.key(i+1))
	}
	return
}

// keep links the copy of row i, of key, which the last frames built, in place
// of building it, and returns its holder, or nil when the row must be built.
func (f *listFrame) keep(i int, key any) *node {
	v := f.s.RowVersion
	if v == nil || f.noKeep {
		return nil
	}
	rec := f.kept[key]
	if rec == nil {
		return nil
	}
	c, rt := f.c, f.c.rt
	if rec.i != i || rec.version != v(i) || rec.themeGen != rt.themeGen || rec.theme != f.theme || rec.parent != c.parent.id || rec.sig != f.sig || rec.dir != c.parent.dir&^dirSet || rec.lastFrame+1 < rt.frame {
		// A row out of view for a frame had its states pruned and recycled.
		return nil
	}
	if rec.chosen != f.chosenAround(i) || (f.s.Checked != nil && rec.checked != f.s.Checked.has(key)) || !f.ok(i, rec.states) {
		return nil
	}
	root := rec.root
	root.next = nil
	c.parent.add(root)
	if root.depth != rec.depth {
		rec.depth = root.depth
		setDepth(root)
	}
	for _, st := range rec.states {
		// Stamped as built in every pass, as lookState would: not through it,
		// which clears what a build sets again (drags, drops).
		st.seen, st.pass = rt.frame, rt.pass
	}
	for j := range rec.nodes {
		n := &rec.nodes[j]
		// What layout and commit write is zero, as in an element just built: a
		// row laid out writes it, and one that is not (below the view) holds
		// nothing of the frame it was copied in. The measures of text and
		// heights (nmeasure, leafOK, intrOK) are of one frame, and depend on
		// the text style the row inherits, which may have changed; sameRow
		// compares the row by the record in its state (lay), which stays.
		n.lay, n.tl, n.nmeasure, n.leafOK, n.intrOK = 0, nil, 0, false, 0
		n.x, n.y, n.w, n.h, n.contentW, n.contentH, n.ink, n.world = 0, 0, 0, 0, 0, 0, [4]float32{}, scene.Affine{}
	}
	rec.lastFrame = rt.frame
	f.rows = append(f.rows, listRow{i: i, key: key, e: root})
	rt.rowsKept++
	return root
}

// setDepth gives the elements under e the depth they have below it.
func setDepth(e *node) {
	for c := e.first; c != nil; c = c.next {
		c.depth = e.depth + 1
		setDepth(c)
	}
}

// retainable reports whether e, and the elements in it, hold nothing of the
// pass that a copy would keep wrong: pointers to other elements, closures
// that capture them, a focus, a tooltip, an effect, an animation.
func retainable(e *node) bool {
	if e.popover != nil || e.placeOn != nil || e.nameFrom != nil || e.focusTo != nil || e.activeDescendant != nil ||
		e.field != nil || e.form != nil || e.followX != nil || e.colFit != nil || e.gridFit != nil || e.overflow != nil ||
		e.list != nil || e.rowsOf != nil || e.track != nil || e.webView != nil || e.inputFn != nil || e.objectFn != nil ||
		e.paintAfterFn != nil || e.paintFn != nil || e.textClient != nil || e.accFill != nil || e.copyData != nil || e.tip ||
		e.flags&flagFocusable != 0 || e.effect != nil || e.hidden || e.leaving != 0 || e.ghosts {
		return false
	}
	for c := e.first; c != nil; c = c.next {
		if !retainable(c) {
			return false
		}
	}
	return true
}

// count returns how many elements e has with those under it.
func count(e *node) int {
	n := 1
	for c := e.first; c != nil; c = c.next {
		n += count(c)
	}
	return n
}

// copyIn writes e and the elements under it into k's storage, depth first,
// linking the copies to each other, and notes their states.
func (k *rowKeep) copyIn(e *node) {
	n := count(e)
	if cap(k.nodes) < n {
		k.nodes = make([]node, n)
	} else {
		k.nodes = k.nodes[:n]
	}
	k.states = k.states[:0]
	at := 0
	k.root = k.fill(e, &at, nil)
}

func (k *rowKeep) fill(e *node, at *int, parent *node) *node {
	d := &k.nodes[*at]
	*at++
	// The slices the storage had keep their arrays: a steady row allocates
	// nothing, and a copy never shares an array with the pass's element.
	shadows, cols, rows, frags, spans := d.shadows, d.cols, d.rows, d.frags, d.spans
	*d = *e
	d.shadows, d.cols, d.rows, d.frags, d.spans = append(shadows[:0], e.shadows...), append(cols[:0], e.cols...),
		append(rows[:0], e.rows...), append(frags[:0], e.frags...), append(spans[:0], e.spans...)
	d.parent, d.first, d.last, d.next, d.nchild = parent, nil, nil, nil, 0
	if e.st != nil {
		k.states = append(k.states, e.st)
	}
	for c := e.first; c != nil; c = c.next {
		cd := k.fill(c, at, d)
		if d.last == nil {
			d.first = cd
		} else {
			d.last.next = cd
		}
		d.last = cd
		d.nchild++
	}
	return d
}

// captureRows keeps copies of the rows the frame built, after its last pass
// was committed, and forgets those that were not in it.
func (rt *engine) captureRows() {
	for i, f := range rt.keepLists {
		if slices.Contains(rt.keepLists[:i], f) || f.frame != rt.frame {
			continue
		}
		if f.pass == rt.pass {
			f.capture()
		} else {
			// Built in a pass before the last, and not in it: prune recycled
			// the states of its rows, and the arena slots of them were written
			// again.
			f.releaseAll()
		}
	}
	clear(rt.keepLists)
	rt.keepLists = rt.keepLists[:0]
}

// capture keeps the rows f built in its last pass.
func (f *listFrame) capture() {
	rt, v := f.c.rt, f.s.RowVersion
	if v == nil || f.noKeep {
		f.kept = nil
		return
	}
	if f.kept == nil {
		f.kept = map[any]*rowKeep{}
	}
	f.capSeq++
	for _, r := range f.rows {
		rec := f.kept[r.key]
		if rec != nil {
			rec.mark = f.capSeq
		}
		if rec != nil && rec.lastFrame == rt.frame {
			// Linked, or kept, in this frame: its storage is a row in the tree.
			// Another row with the key (an app that repeats a key) leaves it be.
			continue
		}
		if rt.noRetain || slices.Contains(f.handles, r.key) || !retainable(r.e) {
			f.release(r.key)
			continue
		}
		fresh := rec == nil
		if fresh {
			rec = f.spareKeep()
		}
		rec.copyIn(r.e)
		if !f.ok(r.i, rec.states) {
			if fresh {
				f.spare = append(f.spare, rec)
			} else {
				f.release(r.key)
			}
			continue
		}
		rec.i, rec.version, rec.themeGen, rec.theme, rec.lastFrame, rec.depth = r.i, v(r.i), rt.themeGen, f.theme, rt.frame, r.e.depth
		rec.parent, rec.sig, rec.dir = r.e.parent.id, f.sig, r.e.parent.dir&^dirSet
		rec.chosen = f.chosenAround(r.i)
		rec.checked = f.s.Checked != nil && f.s.Checked.has(r.key)
		rec.mark = f.capSeq
		f.kept[r.key] = rec
	}
	// Those the last pass did not link or build: a row linked in an earlier
	// pass had its states pruned with the pass.
	for k, rec := range f.kept {
		if rec.mark != f.capSeq {
			f.release(k)
		}
	}
}

// releaseAll forgets every record of f.
func (f *listFrame) releaseAll() {
	for k := range f.kept {
		f.release(k)
	}
}

// settingsSig hashes what a list does with its rows besides their data: the
// choice, the menus, the drags. A row kept was built with the settings the
// list had.
func (s *ListState) settingsSig() (h uint64) {
	for i, on := range [...]bool{s.cursor() != nil, s.Selection != nil, s.DragSelect, s.CellSelect, s.Reorder != nil,
		s.RowMenu != nil, s.CellMenu != nil, s.Label != nil, s.Checked != nil} {
		if on {
			h |= 1 << i
		}
	}
	return h<<48 ^ math.Float64bits(s.gap)>>16
}

// spareKeep returns a record for a row, from those released.
func (f *listFrame) spareKeep() *rowKeep {
	if n := len(f.spare); n > 0 {
		rec := f.spare[n-1]
		f.spare[n-1] = nil
		f.spare = f.spare[:n-1]
		return rec
	}
	return new(rowKeep)
}

// release forgets the record of key, keeping its storage for another row.
func (f *listFrame) release(key any) {
	if rec := f.kept[key]; rec != nil {
		delete(f.kept, key)
		clear(rec.states)
		rec.root = nil
		f.spare = append(f.spare, rec)
	}
}

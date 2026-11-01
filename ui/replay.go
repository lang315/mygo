package ui

import (
	"slices"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

// Replaying rows. A list row that lays out and paints with the same inputs as
// in the last frame (layPaint), painted in the same clip and the same
// surroundings (rowKey), in the same place or moved by whole device pixels in Y
// (retain_translate.go), appends the operations it painted then
// instead of painting again: the frames of a table that only hovers, ticks a
// clock elsewhere or does nothing paint most of their rows this way. A row
// records what it painted when everything it painted is in the scene's
// operations and glyphs alone. The code is reached only through rowPainter, which
// linkPaintCompare sets, so that an app without a list links none of it.

// rowKey is what painting a row read around it.
type rowKey struct {
	box, clip      Rect
	scale, opacity float32
	opaque         bool
	atlases        [2]uint64
	text           scene.TextParams
	themeGen       uint64
	focusVisible   bool
	windowFocus    bool
}

// rowPaint is what a row painted (see the comment above).
type rowPaint struct {
	key rowKey
	// under is the stack of opaque boxes around the row.
	under []opaqueBox
	// ids are the elements of the row, depth first, and pre and post their
	// flags before and after the row painted.
	ids       []uint64
	pre, post []uint32
	// boxes are the elements' boxes, which the layout of the row's elements
	// gives them: the same elements laid out as before may be placed by what
	// is around them (a grid's tracks, a table's fitted columns).
	boxes  []Rect
	ops    []scene.Op
	glyphs []scene.Glyph
	// g0 is len(Scene.Glyphs) when the row recorded.
	g0   int32
	uses text.Uses
	ok   bool
	// at is the layout the row last painted in, recorded or replayed: a row
	// that painted anew since, without recording, left a record older than
	// the frame before, which the row no longer looks like.
	at uint64
	// shiftable and inside tell that the record may be replayed shifted
	// (retain_translate.go): its coordinates are integers, and the row was
	// whole inside the clip when it recorded. frac is the layout whose shift
	// from the record was not a whole device pixel (at fracY): the row paints
	// anew and the record stays where it was, unless the row stands still
	// there in the next layout, when it records where it stands.
	shiftable, inside bool
	frac              uint64
	fracY             float32
	// covered: an opaque background around the row held its box (cut by the
	// clip) when it recorded, which decides how text on it is drawn (subpixel);
	// a row that moved is replayed only where that is the same.
	covered bool
}

// rowPainter paints a child of a list, replaying it where it can and
// recording it where it may. Set by linkPaintCompare.
var rowPainter func(p *Painter, c *node)

// rowKeyOf returns the key of painting c now. The window's focus is read
// without windowFocus, which asks for a frame when it changes: a row with a
// look that reads it is hovered, pressed or focused, and paints anew.
func rowKeyOf(p *Painter, c *node) rowKey {
	rt := p.rt
	return rowKey{
		box: Rect{c.x, c.y, c.w, c.h}, clip: p.clip, scale: p.scale, opacity: p.opacity, opaque: p.opaque,
		atlases: atlasGenerations(p.s), text: p.s.Text, themeGen: rt.themeGen, focusVisible: rt.focusVisible, windowFocus: rt.windowFocused,
	}
}

// recordable reports whether painting c now may be recorded, or replayed: into
// the window's scene, with culling, untransformed, and without the inspector, a
// drag or a web view's hole. A row cut by the clip records too: what culling
// drops of it depends on its ink and the clip, which are the row's own and in
// its key.
func recordable(p *Painter, c *node) bool {
	rt := p.rt
	if rt.noReplay || p.s != &rt.scene || p.drawing || !p.cull || p.transform.Set || c.noCull ||
		rt.insp.open || rt.drag != nil || len(p.holes) > 0 || c.st == nil {
		return false
	}
	return true
}

// rowInteractive reports whether one of the elements ids is hovered, pressed or
// focused: how they paint then is not in the row's key.
func rowInteractive(rt *engine, ids []uint64) bool {
	for _, id := range ids {
		if slices.Contains(rt.hover, id) || rt.pressed != nil && rt.pressed.id == id || rt.focused == id {
			return true
		}
	}
	return false
}

// paintRow is rowPainter.
func paintRow(p *Painter, c *node) {
	if !c.listRow {
		p.element(c)
		return
	}
	if replayRow(p, c) {
		if p.rt.replayCheck {
			checkReplay(p, c)
		}
		return
	}
	rt, s := p.rt, p.s
	rt.freshRows++
	if c.lay&layPaint == 0 || !recordable(p, c) {
		p.element(c)
		return
	}
	if r := c.st.replay; r != nil && r.ok && r.frac == rt.layouts {
		// Not a whole pixel from the record: painted anew and not recorded,
		// so that the next frame, a whole pixel from it, may replay.
		r.at = rt.layouts
		p.element(c)
		return
	}
	key := rowKeyOf(p, c)
	ops, glyphs, stops, wide := len(s.Ops), len(s.Glyphs), len(s.Stops), len(s.Wide)
	effects, canvases, drawn, transient := len(s.Effects), len(s.Canvases), len(rt.drawn), rt.text.TransientUses()
	r := c.st.replay
	if r == nil {
		r = rt.newRowPaint()
		c.st.replay = r
	}
	r.ok = false
	r.ids, r.pre, r.boxes = r.ids[:0], r.pre[:0], r.boxes[:0]
	walkRow(c, func(e *node) {
		r.ids = append(r.ids, e.id)
		r.pre = append(r.pre, e.flags)
		r.boxes = append(r.boxes, Rect{e.x, e.y, e.w, e.h})
	})
	// Recording does not nest: a row holding a list is never paint-same (P1-A).
	rt.text.Record(&r.uses)
	p.element(c)
	rt.text.StopRecord()
	if len(s.Stops) != stops || len(s.Wide) != wide || len(s.Effects) != effects || len(s.Canvases) != canvases ||
		len(rt.drawn) != drawn || rt.text.TransientUses() != transient || rt.untracked || rowInteractive(rt, r.ids) {
		return
	}
	for i := ops; i < len(s.Ops); i++ {
		o := &s.Ops[i]
		if o.Wide != 0 || o.Transform.Set || o.Image != nil || o.Kind == scene.OpEffect || o.Kind == scene.OpHole {
			return
		}
	}
	for i := glyphs; i < len(s.Glyphs); i++ {
		if g := &s.Glyphs[i]; g.Wide != 0 || g.Transform.Set {
			return
		}
	}
	r.key = key
	r.under = append(r.under[:0], p.under...)
	r.ops = append(r.ops[:0], s.Ops[ops:]...)
	r.glyphs = append(r.glyphs[:0], s.Glyphs[glyphs:]...)
	r.g0 = int32(glyphs)
	r.post = r.post[:0]
	walkRow(c, func(e *node) { r.post = append(r.post, e.flags) })
	r.ok = true
	r.at = rt.layouts
	r.shiftable, r.inside = shiftable(r.ops, r.glyphs), insideClip(c, p.clip)
	r.covered = p.covered(intersect(Rect{c.x, c.y, c.w, c.h}, p.clip))
	rt.recorded++
}

// walkRow calls fn on e and every element inside it, depth first.
func walkRow(e *node, fn func(e *node)) {
	fn(e)
	for c := e.first; c != nil; c = c.next {
		walkRow(c, fn)
	}
}

// collectRow appends e and every element inside it to nodes, depth first.
func collectRow(e *node, nodes []*node) []*node {
	nodes = append(nodes, e)
	for c := e.first; c != nil; c = c.next {
		nodes = collectRow(c, nodes)
	}
	return nodes
}

// replayRow appends what c, a list row, painted the last time, and reports
// whether it could: c lays out and paints as then (layPaint), in the same
// place, clip and surroundings, with the same elements in the same flags, and
// none of them is hovered, pressed or focused.
func replayRow(p *Painter, c *node) bool {
	rt, s := p.rt, p.s
	r := c.st.replay
	if r == nil || !r.ok || r.at+1 < rt.layouts || c.lay&layPaint == 0 || !recordable(p, c) {
		return false
	}
	// The same key but for the row's y: a row that moved by dy is replayed
	// shifted, if it may be.
	key := rowKeyOf(p, c)
	newY := key.box.Y
	dy := newY - r.key.box.Y
	key.box.Y = r.key.box.Y
	if r.key != key || !slices.Equal(r.under, p.under) || rowInteractive(rt, r.ids) {
		return false
	}
	// float32() rounds the product: without it arm64 may fuse it into the add
	// that translate does, and leave a coordinate off by an ulp of the product.
	d := float32(dy * p.scale)
	if dy != 0 {
		if d != float32(int32(d)) {
			// Not a whole device pixel: paint anew and keep the record, which
			// a scroll on may be a whole pixel from; a row that stands still
			// here records where it stands.
			if still := r.frac != 0 && r.frac+1 == rt.layouts && r.fracY == newY; still {
				r.frac = 0
			} else {
				r.frac, r.fracY = rt.layouts, newY
			}
			return false
		}
		if !r.shiftable || !r.inside || !insideClip(c, p.clip) || r.covered != p.covered(intersect(Rect{c.x, c.y, c.w, c.h}, p.clip)) {
			return false
		}
	}
	// The same elements, in the same order, with the flags they were built
	// with, in the same boxes (shifted by dy exactly, if the row moved). One
	// walk finds them; the rest goes over the list it makes.
	rt.replayWalks++
	nodes := collectRow(c, rt.rowNodes[:0])
	rt.rowNodes = nodes[:0]
	if len(nodes) != len(r.ids) {
		return false
	}
	for i, e := range nodes {
		b := r.boxes[i]
		b.Y += dy
		if r.ids[i] != e.id || r.pre[i] != e.flags || b != (Rect{e.x, e.y, e.w, e.h}) {
			return false
		}
	}
	if dy != 0 {
		translate(r.ops, r.glyphs, d)
		r.key.box.Y = newY
		for i, e := range nodes {
			r.boxes[i].Y = e.y
		}
		rt.rowsShiftReplayed++
	}
	delta := int32(len(s.Glyphs)) - r.g0
	base := len(s.Ops)
	s.Ops = append(s.Ops, r.ops...)
	for k := base; k < len(s.Ops); k++ {
		if o := &s.Ops[k]; o.Kind == scene.OpGlyphs {
			o.Start += delta
			o.End += delta
		}
	}
	s.Glyphs = append(s.Glyphs, r.glyphs...)
	rt.text.Touch(&r.uses)
	for i, e := range nodes {
		e.flags = r.post[i]
	}
	r.at = rt.layouts
	rt.replayed++
	return true
}

// checkReplay paints c as well, after its replay, and counts in
// replayMismatch a difference between the two, then drops the painting.
func checkReplay(p *Painter, c *node) {
	rt, s := p.rt, p.s
	r := c.st.replay
	// A record that may be shifted still holds whole pixels after a shift:
	// the next shift is added to it.
	if r.shiftable && !shiftable(r.ops, r.glyphs) {
		rt.replayMismatch++
	}
	ops, glyphs := len(s.Ops), len(s.Glyphs)
	from := ops - len(r.ops)
	gfrom := glyphs - len(r.glyphs)
	saved := make([]uint32, 0, len(r.ids))
	walkRow(c, func(e *node) { saved = append(saved, e.flags) })
	p.element(c)
	fresh, freshGlyphs := s.Ops[ops:], s.Glyphs[glyphs:]
	replayed, replayedGlyphs := s.Ops[from:ops], s.Glyphs[gfrom:glyphs]
	if len(fresh) != len(replayed) || !slices.Equal(freshGlyphs, replayedGlyphs) {
		rt.replayMismatch++
	} else {
		shift := int32(glyphs - gfrom)
		for k := range fresh {
			a, b := fresh[k], replayed[k]
			if a.Kind == scene.OpGlyphs {
				a.Start, a.End = a.Start-shift, a.End-shift
			}
			if a != b {
				rt.replayMismatch++
				break
			}
		}
	}
	i := 0
	walkRow(c, func(e *node) {
		if e.flags != saved[i] {
			rt.replayMismatch++
		}
		i++
	})
	s.Ops, s.Glyphs = s.Ops[:ops], s.Glyphs[:glyphs]
}

// newRowPaint returns an empty record, from those of the states that went.
func (rt *engine) newRowPaint() *rowPaint {
	n := len(rt.rowPool)
	if n == 0 {
		return &rowPaint{}
	}
	r := rt.rowPool[n-1]
	rt.rowPool = rt.rowPool[:n-1]
	*r = rowPaint{under: r.under[:0], ids: r.ids[:0], pre: r.pre[:0], post: r.post[:0], boxes: r.boxes[:0],
		ops: r.ops[:0], glyphs: r.glyphs[:0], uses: r.uses}
	r.uses.Reset()
	return r
}

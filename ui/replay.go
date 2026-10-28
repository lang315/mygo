package ui

import (
	"slices"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

// Replaying rows. A list row that lays out and paints with the same inputs as
// in the last frame (layPaint), painted in the same place, the same clip and
// the same surroundings (rowKey), appends the operations it painted then
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
	// last is the key of the row's last painting, recorded or not: a row records
	// when it paints twice in a row in the same place, not while it moves.
	last rowKey
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
	key := rowKeyOf(p, c)
	ops, glyphs, stops, wide := len(s.Ops), len(s.Glyphs), len(s.Stops), len(s.Wide)
	effects, canvases, drawn, transient := len(s.Effects), len(s.Canvases), len(rt.drawn), rt.text.TransientUses()
	r := c.st.replay
	stable := r != nil && r.last == key
	if r == nil {
		r = &rowPaint{}
		c.st.replay = r
		stable = true
	}
	r.last = key
	if !stable {
		// Scrolling, resizing: recording would copy what the next frame cannot
		// replay.
		r.ok = false
		p.element(c)
		return
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
	rt.recorded++
}

// walkRow calls fn on e and every element inside it, depth first.
func walkRow(e *node, fn func(e *node)) {
	fn(e)
	for c := e.first; c != nil; c = c.next {
		walkRow(c, fn)
	}
}

// replayRow appends what c, a list row, painted the last time, and reports
// whether it could: c lays out and paints as then (layPaint), in the same
// place, clip and surroundings, with the same elements in the same flags, and
// none of them is hovered, pressed or focused.
func replayRow(p *Painter, c *node) bool {
	rt, s := p.rt, p.s
	r := c.st.replay
	if r == nil || !r.ok || c.lay&layPaint == 0 || !recordable(p, c) || r.key != rowKeyOf(p, c) ||
		!slices.Equal(r.under, p.under) || rowInteractive(rt, r.ids) {
		return false
	}
	// The same elements, in the same order, with the flags they were built
	// with, in the same boxes.
	i := 0
	same := true
	walkRow(c, func(e *node) {
		if i >= len(r.ids) || r.ids[i] != e.id || r.pre[i] != e.flags || r.boxes[i] != (Rect{e.x, e.y, e.w, e.h}) {
			same = false
		}
		i++
	})
	if !same || i != len(r.ids) {
		return false
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
	i = 0
	walkRow(c, func(e *node) { e.flags = r.post[i]; i++ })
	rt.replayed++
	return true
}

// checkReplay paints c as well, after its replay, and counts in
// replayMismatch a difference between the two, then drops the painting.
func checkReplay(p *Painter, c *node) {
	rt, s := p.rt, p.s
	r := c.st.replay
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

package ui

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

// pcExtras style the random trees' elements further for painting: moved
// far, out of their clips and back, turned, scaled, blurred, with wide
// shadows, drawings beyond their boxes, dividers and the debug overlay.
var pcExtras = []func(e *node){
	nil,
	func(e *node) { e.Translate(0, -260) },
	func(e *node) { e.Translate(0, 340) },
	func(e *node) { e.Translate(-420, 15) },
	func(e *node) { e.Rotate(30) },
	func(e *node) { e.Scale(0.5, 3) },
	func(e *node) { e.Blur(2) },
	func(e *node) { e.Shadow(0, 60, 30, 10, RGBA(0, 0, 0, 0.4)) },
	func(e *node) {
		e.Draw(func(p *Painter, r Rect) { p.Fill(Rect{r.X - 30, r.Y - 30, r.W + 60, 10}, RGB(0, 120, 0), 0) })
	},
	func(e *node) {
		e.DrawOver(func(p *Painter, r Rect) { p.Fill(Rect{r.X, r.Y + r.H, 20, 40}, RGB(0, 0, 160), 2) })
	},
	func(e *node) { e.Dividers(1, RGB(90, 90, 90)) },
	func(e *node) { e.Debug() },
	func(e *node) { e.Focusable().Background(RGB(250, 250, 200)) },
	func(e *node) { e.Disabled(true).Background(RGB(30, 30, 30)) },
}

// Painting skips what its ink tells it shows nowhere (Painter.paintable),
// and paints what painting everything does: the same operations, but
// for clips pushed and popped around nothing, as random trees change and
// move frame after frame. Each frame is painted again with and without
// skipping, over the same images (effects' layers), which then hold the
// same pixels.
func TestPaintCullMatches(t *testing.T) {
	seeds, frames := 20, 60
	if testing.Short() {
		seeds = 6
	}
	if raceEnabled {
		// As in TestLayoutCacheMatches: fewer seeds under the race detector.
		seeds = 4
	}
	var culled uint64
	for seed := range uint64(seeds) {
		m := newLCModel(seed + 1000)
		extras := map[*lcNode]int{}
		m.extra = func(n *lcNode, e *node) {
			if f := pcExtras[extras[n]]; f != nil {
				f(e)
			}
		}
		restyle := func() {
			nodes, _ := m.all()
			for _, n := range nodes {
				if m.rng.IntN(5) == 0 {
					extras[n] = m.rng.IntN(len(pcExtras))
				}
			}
		}
		restyle()
		tt := lcTester(func(c *context) {
			coreBox(c).Fill().Children(func() { m.build(c, 0) })
		}, m.w, m.h, time.Unix(1e9, 0))
		var log []string
		for f := range frames {
			m.dup = nil
			if m.rng.IntN(3) > 0 {
				for range 1 + m.rng.IntN(3) {
					log = append(log, m.mutate())
				}
				restyle()
			} else {
				log = append(log, "unchanged")
			}
			tt.h.w, tt.h.h = float32(m.w), float32(m.h)
			before := tt.rt.culled
			if m.rng.IntN(4) == 0 {
				tt.Advance(30 * time.Millisecond)
			} else {
				tt.Frame()
			}
			culled += tt.rt.culled - before
			// Painted again, as what the frame's painting did first, as
			// images taking their size, is done.
			w, h, scale := tt.h.size()
			tt.rt.paint(tt.rt.c.root, w, h, scale)
			got := snapScene(&tt.rt.scene)
			tt.rt.noCull = true
			tt.rt.paint(tt.rt.c.root, w, h, scale)
			tt.rt.noCull = false
			if d := sameScenes(&got, &tt.rt.scene); d != "" {
				if len(log) > 6 {
					log = log[len(log)-6:]
				}
				t.Fatalf("seed %d, frame %d, after %s: %s", seed, f, strings.Join(log, ", "), d)
			}
		}
	}
	if culled == 0 {
		t.Error("painting skipped no element")
	}
}

// withoutEmptyClips returns ops without the clips pushed and popped
// around nothing.
func withoutEmptyClips(ops []scene.Op) []scene.Op {
	var out []scene.Op
	var open []int // where the clips pushed start in out
	for _, op := range ops {
		switch op.Kind {
		case scene.OpPushClip:
			open = append(open, len(out))
		case scene.OpPopClip:
			if at := open[len(open)-1]; at == len(out)-1 && out[at].Kind == scene.OpPushClip {
				out, open = out[:at], open[:len(open)-1]
				continue
			}
			open = open[:len(open)-1]
		}
		out = append(out, op)
	}
	return out
}

func sameOps(got, want []scene.Op) string {
	for i := range min(len(got), len(want)) {
		if !reflect.DeepEqual(got[i], want[i]) {
			return fmt.Sprintf("op %d of %d: %+v, want %+v", i, len(want), got[i], want[i])
		}
	}
	if len(got) != len(want) {
		return fmt.Sprintf("%d ops, want %d", len(got), len(want))
	}
	return ""
}

// sameScenes compares two frames' ops (without the clips around nothing)
// together with everything they point at: the glyphs (where text is), the
// stops, the wide colors and the effects, and the frame's own fields. sameOps alone misses text that
// moved, since its positions are in Scene.Glyphs. The ops are compared by
// what they point at, not by where in the scene's slices it is.
//
// Words (the effects' uniforms) is no op's index: the ops hold slices of it,
// which DeepEqual compares by content, and the arena itself is compared as
// it is. Canvases are the canvases drawn anew: the same pointers in one
// tester (a replay must not drop or add a canvas), the same number between
// two (each tester has canvases of its own); the ops point to their images
// by pointer, not by index.
//
// Where a glyph's bitmap is in the atlas (U, V) is compared by sameScenes
// only for frames of one tester: two testers place the same glyphs in
// different places, which sameScenesAcross ignores.
func sameScenes(got, want *scene.Scene) string { return sameScenesIn(got, want, true) }

func sameScenesAcross(got, want *scene.Scene) string { return sameScenesIn(got, want, false) }

func sameScenesIn(got, want *scene.Scene, shared bool) string {
	sameGlyphs := func(a, b []scene.Glyph) bool {
		return slices.EqualFunc(a, b, func(x, y scene.Glyph) bool {
			if !shared {
				x.U, x.V, y.U, y.V = 0, 0, 0, 0
			}
			return x == y
		})
	}
	if got.Width != want.Width || got.Height != want.Height || got.Scale != want.Scale || got.Zoom != want.Zoom ||
		got.Clear != want.Clear || got.Background != want.Background || got.Text != want.Text || got.Drawn != want.Drawn {
		return fmt.Sprintf("the frames differ: %dx%d scale %v zoom %v clear %v background %v text %v drawn %v, want %dx%d scale %v zoom %v clear %v background %v text %v drawn %v",
			got.Width, got.Height, got.Scale, got.Zoom, got.Clear, got.Background, got.Text, got.Drawn,
			want.Width, want.Height, want.Scale, want.Zoom, want.Clear, want.Background, want.Text, want.Drawn)
	}
	if !slices.Equal(got.Words, want.Words) {
		return fmt.Sprintf("effect words %v, want %v", got.Words, want.Words)
	}
	if len(got.Canvases) != len(want.Canvases) || shared && !slices.Equal(got.Canvases, want.Canvases) {
		return fmt.Sprintf("canvases %v, want %v", got.Canvases, want.Canvases)
	}
	g, w := withoutEmptyClips(got.Ops), withoutEmptyClips(want.Ops)
	for i := range min(len(g), len(w)) {
		if d := sameOpIn(g[i], got, w[i], want, sameGlyphs); d != "" {
			return fmt.Sprintf("op %d of %d: %s", i, len(w), d)
		}
	}
	if len(g) != len(w) {
		return fmt.Sprintf("%d ops, want %d", len(g), len(w))
	}
	return ""
}

// snapScene copies what sameScenes compares, for a scene its renderer
// reuses for the next frame.
func snapScene(s *scene.Scene) scene.Scene {
	return scene.Scene{
		Ops:     slices.Clone(s.Ops),
		Glyphs:  slices.Clone(s.Glyphs),
		Stops:   slices.Clone(s.Stops),
		Wide:    slices.Clone(s.Wide),
		Effects: slices.Clone(s.Effects),
		Words:   slices.Clone(s.Words), Canvases: slices.Clone(s.Canvases),
		Width: s.Width, Height: s.Height, Scale: s.Scale, Zoom: s.Zoom,
		Clear: s.Clear, Background: s.Background, Text: s.Text, Drawn: s.Drawn,
	}
}

func sameOpIn(x scene.Op, as *scene.Scene, y scene.Op, bs *scene.Scene, sameGlyphs func(a, b []scene.Glyph) bool) string {
	// The slices the ops index are compared through them.
	switch x.Kind {
	case scene.OpGlyphs:
		if !sameGlyphs(as.Glyphs[x.Start:x.End], bs.Glyphs[y.Start:y.End]) {
			return fmt.Sprintf("glyphs %+v, want %+v", as.Glyphs[x.Start:x.End], bs.Glyphs[y.Start:y.End])
		}
		x.Start, x.End, y.Start, y.End = 0, 0, 0, 0
	case scene.OpEffect:
		ea, eb := as.Effects[x.Start], bs.Effects[y.Start]
		ga, gb := ea.Glyphs, eb.Glyphs
		var ra, rb []scene.Glyph
		if ga != ([2]int32{}) {
			ra = as.Glyphs[ga[0]:ga[1]]
		}
		if gb != ([2]int32{}) {
			rb = bs.Glyphs[gb[0]:gb[1]]
		}
		ea.Glyphs, eb.Glyphs = [2]int32{}, [2]int32{}
		if !sameGlyphs(ra, rb) || !reflect.DeepEqual(ea, eb) {
			return fmt.Sprintf("effect %+v, want %+v (glyphs %+v, want %+v)", ea, eb, ra, rb)
		}
		x.Start, x.End, y.Start, y.End = 0, 0, 0, 0
	}
	if x.Stops != ([2]int32{}) || y.Stops != ([2]int32{}) {
		if !slices.Equal(as.Stops[x.Stops[0]:x.Stops[1]], bs.Stops[y.Stops[0]:y.Stops[1]]) {
			return "stops differ"
		}
		x.Stops, y.Stops = [2]int32{}, [2]int32{}
	}
	if x.Wide != 0 || y.Wide != 0 {
		if x.Wide == 0 || y.Wide == 0 || !reflect.DeepEqual(as.Wide[x.Wide-1], bs.Wide[y.Wide-1]) {
			return fmt.Sprintf("wide colors differ (%d, %d)", x.Wide, y.Wide)
		}
		x.Wide, y.Wide = 0, 0
	}
	if !reflect.DeepEqual(x, y) {
		return fmt.Sprintf("%+v, want %+v", x, y)
	}
	return ""
}

// The rows of a long column scrolled by a translation in a clip are
// painted only where they show.
func TestPaintCullsScrolledRows(t *testing.T) {
	var dy float32
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Clip().Children(func() {
			coreColumn(c).Translate(0, dy).Children(func() {
				for i := range 200 {
					coreRow(c).Height(28).Gap(8).Children(func() {
						coreText(c, "Row "+itoa(i))
						coreSpacer(c)
						coreText(c, itoa(i*7%100))
					})
				}
			})
		})
	}, 400, 300)
	for _, d := range []float32{0, -1000, -4000} {
		dy = d
		before := tt.rt.culled
		tt.Frame()
		// 11 rows show at most; the others are skipped whole.
		if n := tt.rt.culled - before; n < 188 || n > 190 {
			t.Errorf("at %v, %d rows skipped", d, n)
		}
	}
}

// sameScenes sees a change in each channel an op points into or the frame
// holds, and none in where in the slices the same content is.
func TestSameScenesSeesEachChannel(t *testing.T) {
	cv := &scene.Image{}
	base := func() *scene.Scene {
		return &scene.Scene{
			Width: 10, Height: 10, Scale: 1,
			Ops: []scene.Op{
				{Kind: scene.OpGlyphs, Start: 0, End: 1},
				{Kind: scene.OpFill, Paint: scene.PaintLinearStops, Stops: [2]int32{0, 2}},
				{Kind: scene.OpFill, Wide: 1},
				{Kind: scene.OpEffect, Start: 0},
			},
			Glyphs:   []scene.Glyph{{X: 1, Y: 2, W: 3, H: 4}},
			Stops:    []scene.Stop{{At: 0}, {At: 1}},
			Wide:     []scene.WideColors{{Color: [4]float32{2, 0, 0, 1}}},
			Effects:  []scene.EffectOp{{Uniforms: []uint32{7}}},
			Words:    []uint32{7},
			Canvases: []*scene.Image{cv},
		}
	}
	if d := sameScenes(base(), base()); d != "" {
		t.Fatalf("the same scene differs from itself: %s", d)
	}
	for name, change := range map[string]func(s *scene.Scene){
		"glyph":      func(s *scene.Scene) { s.Glyphs[0].Y++ },
		"stop":       func(s *scene.Scene) { s.Stops[1].At = 0.5 },
		"wide":       func(s *scene.Scene) { s.Wide[0].Color[0] = 3 },
		"effect":     func(s *scene.Scene) { s.Effects[0].Blur = 2 },
		"words":      func(s *scene.Scene) { s.Words[0] = 8 },
		"canvas":     func(s *scene.Scene) { s.Canvases[0] = &scene.Image{} },
		"no canvas":  func(s *scene.Scene) { s.Canvases = nil },
		"clear":      func(s *scene.Scene) { s.Clear = scene.Color{R: 1} },
		"background": func(s *scene.Scene) { s.Background = scene.Color{R: 1} },
		"text":       func(s *scene.Scene) { s.Text.Contrast = 1 },
		"size":       func(s *scene.Scene) { s.Width++ },
	} {
		s := base()
		change(s)
		if sameScenes(s, base()) == "" {
			t.Errorf("a change of the %s is not seen", name)
		}
	}
	// The same content at other indexes is the same frame.
	moved := base()
	moved.Glyphs = append([]scene.Glyph{{X: 9}}, moved.Glyphs...)
	moved.Ops[0].Start, moved.Ops[0].End = 1, 2
	moved.Stops = append([]scene.Stop{{At: 9}}, moved.Stops...)
	moved.Ops[1].Stops = [2]int32{1, 3}
	moved.Wide = append([]scene.WideColors{{}}, moved.Wide...)
	moved.Ops[2].Wide = 2
	moved.Effects = append([]scene.EffectOp{{}}, moved.Effects...)
	moved.Ops[3].Start = 1
	if d := sameScenes(moved, base()); d != "" {
		t.Errorf("content at other indexes differs: %s", d)
	}
	// Two testers' canvases are told apart by their number.
	other := base()
	other.Canvases[0] = &scene.Image{}
	if d := sameScenesAcross(other, base()); d != "" {
		t.Errorf("another tester's canvas differs: %s", d)
	}
}

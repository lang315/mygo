package raster

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// matching returns the index of the PopClip that closes the PushClip at p, or -1.
func matching(ops []scene.Op, p int) int {
	depth := 0
	for q := p; q < len(ops); q++ {
		switch ops[q].Kind {
		case scene.OpPushClip:
			depth++
		case scene.OpPopClip:
			if depth--; depth == 0 {
				return q
			}
		}
	}
	return -1
}

// changeMany changes s in one to three places at once, as a frame that shows
// a pressed row and a scrollbar does, and says how: the changes of change,
// ops inserted anywhere and at the end, a hole, a clip put around some ops
// (sometimes under a transform) and a clip taken off them.
func (m *sceneMaker) changeMany(s *scene.Scene) []string {
	var what []string
	for range 1 + m.rnd.IntN(3) {
		at := m.rnd.IntN(len(s.Ops) + 1)
		switch m.rnd.IntN(7) {
		case 0, 1:
			what = append(what, m.change(s))
		case 2:
			s.Ops = append(s.Ops[:at], append([]scene.Op{m.op(s)}, s.Ops[at:]...)...)
			what = append(what, "inserted")
		case 3:
			s.Ops = append(s.Ops, m.op(s))
			what = append(what, "appended")
		case 4:
			hole := scene.Op{Kind: scene.OpHole, Rect: m.rect(), Radii: [4]float32{3, 3, 3, 3}, Opacity: float32(m.rnd.IntN(2)) * 0.5}
			s.Ops = append(s.Ops[:at], append([]scene.Op{hole}, s.Ops[at:]...)...)
			what = append(what, "hole")
		case 5:
			// A clip around ops that hold their clips in pairs.
			end := at
			depth := 0
			for end < len(s.Ops) && end < at+1+m.rnd.IntN(6) {
				switch s.Ops[end].Kind {
				case scene.OpPushClip:
					depth++
				case scene.OpPopClip:
					depth--
				}
				if depth < 0 {
					break
				}
				end++
			}
			for depth > 0 && end < len(s.Ops) { // close what was opened
				if s.Ops[end].Kind == scene.OpPopClip {
					depth--
				}
				if s.Ops[end].Kind == scene.OpPushClip {
					depth++
				}
				end++
			}
			if depth != 0 || end > len(s.Ops) || at > end {
				what = append(what, "nothing")
				continue
			}
			push := scene.Op{Kind: scene.OpPushClip, Rect: m.rect(), Radii: [4]float32{4, 4, 4, 4}}
			if m.rnd.IntN(3) == 0 {
				push.Transform = scene.Translation(m.rnd.Float32()*10, m.rnd.Float32()*10)
			}
			ops := append([]scene.Op(nil), s.Ops[:at]...)
			ops = append(ops, push)
			ops = append(ops, s.Ops[at:end]...)
			ops = append(ops, scene.Op{Kind: scene.OpPopClip})
			s.Ops = append(ops, s.Ops[end:]...)
			what = append(what, "clip put around")
		default:
			for p, op := range s.Ops {
				if op.Kind != scene.OpPushClip {
					continue
				}
				if q := matching(s.Ops, p); q > p {
					s.Ops = append(s.Ops[:q], s.Ops[q+1:]...)
					s.Ops = append(s.Ops[:p], s.Ops[p+1:]...)
					what = append(what, "clip taken off")
				}
				break
			}
		}
	}
	return what
}

// TestRendererRedrawsSeveralChanges checks that frames that change in two or
// three places, among them ops inserted or deleted far from each other, as
// the lines of a table pressed with a scrollbar shown do, show what a whole
// drawing does, with and without effects that read their backdrops.
func TestRendererRedrawsSeveralChanges(t *testing.T) {
	for _, effects := range []bool{false, true} {
		partial, frames := 0, 0
		for seed := range uint64(120) {
			m := newSceneMaker(seed)
			m.effects = effects
			s := m.scene()
			var r Renderer
			full := NewImage(s.Width, s.Height)
			for step := range 30 {
				what := []string{"first"}
				if step > 0 {
					what = m.changeMany(s)
				}
				damage := r.Render(s)
				Render(full, s)
				if !bytes.Equal(r.Image.Pix, full.Pix) {
					writePNG(t, fmt.Sprintf("multi-%d-%d", seed, step), &r.Image)
					writePNG(t, fmt.Sprintf("multi-%d-%d-want", seed, step), full)
					t.Fatalf("effects %v, seed %d, step %d (%v): the redrawn %v differs from a whole drawing", effects, seed, step, what, damage)
				}
				if step > 0 {
					frames++
					if len(damage) != 1 || damage[0].Dx() != s.Width || damage[0].Dy() != s.Height {
						partial++
					}
				}
			}
		}
		t.Logf("effects %v: seeds 0..119, 30 steps each; %d of %d frames redrew part of the window", effects, partial, frames)
		if partial == 0 {
			t.Errorf("effects %v: every frame was drawn whole", effects)
		}
	}
}

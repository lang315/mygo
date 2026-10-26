package raster

import (
	"bytes"
	"fmt"
	"image"
	"math/rand/v2"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// advMove is how far, in ops, the adversarial changes move a clip's push or
// pop: past the 8 ops Diff looks ahead for a resync (alignWindow), so that
// the scenes with as many ops are compared in pairs.
const (
	advMove  = 24
	advSeeds = 200
)

// redrawDamage draws old whole, then redraws only the damage Diff (and
// AddBackdrops) finds for next, skipping the "past half the window" shortcut,
// and reports whether that equals next drawn whole.
func redrawDamage(r *Renderer, next *scene.Scene) (bool, []image.Rectangle, int) {
	r.diff.Reach = 1
	d, ok := r.diff.Diff(next)
	if !ok {
		r.Render(next)
		return true, nil, -1
	}
	d = r.diff.AddBackdrops(next, append([]image.Rectangle(nil), d...))
	for _, rc := range d {
		r.d.draw(&r.Image, next, rc, r.diff.Bounds())
	}
	r.diff.Remember(next)
	full := NewImage(next.Width, next.Height)
	Render(full, next)
	diffPx := 0
	for i := 0; i < len(full.Pix); i += 4 {
		if !bytes.Equal(full.Pix[i:i+4], r.Image.Pix[i:i+4]) {
			diffPx++
		}
	}
	return diffPx == 0, d, diffPx
}

func clone(s *scene.Scene) *scene.Scene {
	c := *s
	c.Ops = append([]scene.Op(nil), s.Ops...)
	c.Glyphs = append([]scene.Glyph(nil), s.Glyphs...)
	c.Effects = append([]scene.EffectOp(nil), s.Effects...)
	return &c
}

func fill(x, y, w, h float32, c uint8) scene.Op {
	return scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: x, Y: y, W: w, H: h}, Color: scene.Color{R: c, G: 40, B: 200 - c, A: 255}}
}

// TestAdvPopShiftEqualCount: a clip's pop moves past more ops than align looks
// ahead, while elsewhere one op becomes another, so both scenes have as many
// ops and align gives up. S is the same op in both, but inside the clip in one
// scene and outside it in the other: the pairwise comparison must not match it
// by its index (and the damage must not be empty, which the GPU renderers take
// for a frame that changed nothing).
func TestAdvPopShiftEqualCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		clip scene.Op
		s    scene.Op
	}{
		{"sharp fractional clip", scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{X: 20.5, Y: 20.5, W: 40, H: 40}}, fill(20, 30, 20, 10, 90)},
		{"rounded clip", scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{X: 20, Y: 20, W: 40, H: 40}, Radii: [4]float32{12, 12, 12, 12}}, fill(20, 20, 20, 20, 90)},
		{"rotated clip", scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{X: 30, Y: 30, W: 30, H: 30}, Transform: scene.Rotation(20)}, fill(30, 30, 20, 20, 90)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mk := func(newer bool) *scene.Scene {
				s := &scene.Scene{Width: 160, Height: 120, Clear: scene.Color{R: 250, G: 250, B: 250, A: 255}}
				s.Ops = append(s.Ops, tc.clip, fill(5, 100, 4, 4, 10))
				if newer {
					s.Ops = append(s.Ops, fill(100, 100, 4, 4, 30)) // Y: takes the place of the Pop
				} else {
					s.Ops = append(s.Ops, scene.Op{Kind: scene.OpPopClip})
				}
				s.Ops = append(s.Ops, tc.s)
				for k := range 8 { // same in both, far from the change, so align finds no resync within 8
					s.Ops = append(s.Ops, fill(float32(120+k*4), 5, 2, 2, uint8(k*20)))
				}
				if newer {
					s.Ops = append(s.Ops, scene.Op{Kind: scene.OpPopClip})
				} else {
					s.Ops = append(s.Ops, fill(140, 100, 4, 4, 50)) // W
				}
				return s
			}
			old, nw := mk(false), mk(true)
			var r Renderer
			r.Render(old)
			ok, d, px := redrawDamage(&r, nw)
			if !ok {
				t.Errorf("damaged draw differs from whole draw in %d pixels; damage %v; S bounds %v", px, d, r.diff.Bounds()[3])
			}
		})
	}
}

// advMaker builds scenes of small ops on a larger window, with nested,
// rounded, fractional, translated and rotated clips, holes, backdrop effects
// and runs of identical ops, and changes them adversarially.
type advMaker struct {
	rnd     *rand.Rand
	effects bool
}

func (m *advMaker) rect() scene.Rect {
	return scene.Rect{X: m.rnd.Float32()*220 - 10, Y: m.rnd.Float32()*170 - 10, W: 1 + m.rnd.Float32()*30, H: 1 + m.rnd.Float32()*30}
}

func (m *advMaker) color() scene.Color {
	return scene.Color{R: uint8(m.rnd.IntN(256)), G: uint8(m.rnd.IntN(256)), B: uint8(m.rnd.IntN(256)), A: uint8(64 + m.rnd.IntN(192))}
}

func (m *advMaker) clip() scene.Op {
	r := m.rect()
	r.W, r.H = r.W*3, r.H*3
	op := scene.Op{Kind: scene.OpPushClip, Rect: r}
	switch m.rnd.IntN(5) {
	case 0:
		k := m.rnd.Float32() * 15
		op.Radii = [4]float32{k, k, k, k}
	case 1:
		op.Transform = scene.Translation(m.rnd.Float32()*10, m.rnd.Float32()*10)
	case 2:
		op.Transform = scene.Rotation(m.rnd.Float32() * 40)
	}
	return op
}

func (m *advMaker) op(s *scene.Scene) scene.Op {
	if m.effects && m.rnd.IntN(5) == 0 {
		fx := scene.EffectOp{Effect: lensEffect, Blur: float32(m.rnd.IntN(4)) * 6}
		if m.rnd.IntN(3) == 0 {
			fx.Effect = tintEffect
		}
		fx.Params[0] = [4]float32{m.rnd.Float32() * 4}
		fx.Params[1] = [4]float32{0.2, 0.5, 0.8, 0.6}
		s.Effects = append(s.Effects, fx)
		return scene.Op{Kind: scene.OpEffect, Rect: m.rect(), Start: int32(len(s.Effects) - 1)}
	}
	switch m.rnd.IntN(5) {
	case 0:
		return scene.Op{Kind: scene.OpHole, Rect: m.rect(), Radii: [4]float32{2, 2, 2, 2}, Opacity: float32(m.rnd.IntN(2)) * 0.5}
	case 1:
		return scene.Op{Kind: scene.OpShadow, Rect: m.rect(), Color: m.color(), Blur: m.rnd.Float32() * 10}
	default:
		k := m.rnd.Float32() * 8
		return scene.Op{Kind: scene.OpFill, Rect: m.rect(), Radii: [4]float32{k, k, k, k}, Color: m.color()}
	}
}

func (m *advMaker) scene() *scene.Scene {
	s := &scene.Scene{Width: 240, Height: 180, Clear: scene.Color{R: 250, G: 250, B: 250, A: 255}}
	depth := 0
	for range 20 + m.rnd.IntN(20) {
		switch x := m.rnd.IntN(10); {
		case x == 0 && depth < 3:
			s.Ops = append(s.Ops, m.clip())
			depth++
		case x == 1 && depth > 0:
			s.Ops = append(s.Ops, scene.Op{Kind: scene.OpPopClip})
			depth--
		case x == 2 && len(s.Ops) > 0 && s.Ops[len(s.Ops)-1].Kind != scene.OpPushClip && s.Ops[len(s.Ops)-1].Kind != scene.OpPopClip:
			s.Ops = append(s.Ops, s.Ops[len(s.Ops)-1]) // repeated identical op
		default:
			s.Ops = append(s.Ops, m.op(s))
		}
	}
	for ; depth > 0; depth-- {
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpPopClip})
	}
	return s
}

func isClip(k scene.Kind) bool { return k == scene.OpPushClip || k == scene.OpPopClip }

// mutate changes s in one place and says how; scenes stay balanced.
func (m *advMaker) mutate(s *scene.Scene) string {
	n := len(s.Ops)
	at := m.rnd.IntN(n + 1)
	switch m.rnd.IntN(11) {
	case 0: // color
		for range 10 {
			i := m.rnd.IntN(n)
			if !isClip(s.Ops[i].Kind) && s.Ops[i].Kind != scene.OpEffect {
				s.Ops[i].Color = m.color()
				return "color"
			}
		}
	case 1: // insert
		s.Ops = append(s.Ops[:at], append([]scene.Op{m.op(s)}, s.Ops[at:]...)...)
		return "insert"
	case 2: // delete a drawing op
		for range 10 {
			i := m.rnd.IntN(n)
			if !isClip(s.Ops[i].Kind) && n > 4 {
				s.Ops = append(s.Ops[:i], s.Ops[i+1:]...)
				return "delete"
			}
		}
	case 3: // duplicate a neighbor (identical ops in a row)
		if at > 0 && !isClip(s.Ops[at-1].Kind) {
			s.Ops = append(s.Ops[:at], append([]scene.Op{s.Ops[at-1]}, s.Ops[at:]...)...)
			return "dup"
		}
	case 4, 5: // move a Pop past drawing ops, forward or back
		for range 10 {
			q := m.rnd.IntN(n)
			if s.Ops[q].Kind != scene.OpPopClip {
				continue
			}
			k := 1 + m.rnd.IntN(advMove)
			if m.rnd.IntN(2) == 0 {
				t := q
				for t+1 < n && t-q < k && !isClip(s.Ops[t+1].Kind) {
					s.Ops[t], s.Ops[t+1] = s.Ops[t+1], s.Ops[t]
					t++
				}
				return fmt.Sprintf("pop later %d", t-q)
			}
			t := q
			for t-1 >= 0 && q-t < k && !isClip(s.Ops[t-1].Kind) {
				s.Ops[t], s.Ops[t-1] = s.Ops[t-1], s.Ops[t]
				t--
			}
			return fmt.Sprintf("pop earlier %d", q-t)
		}
	case 6: // move a Push past drawing ops
		for range 10 {
			p := m.rnd.IntN(n)
			if s.Ops[p].Kind != scene.OpPushClip {
				continue
			}
			k := 1 + m.rnd.IntN(advMove)
			if m.rnd.IntN(2) == 0 {
				t := p
				for t+1 < n && t-p < k && !isClip(s.Ops[t+1].Kind) {
					s.Ops[t], s.Ops[t+1] = s.Ops[t+1], s.Ops[t]
					t++
				}
				return "push later"
			}
			t := p
			for t-1 >= 0 && p-t < k && !isClip(s.Ops[t-1].Kind) {
				s.Ops[t], s.Ops[t-1] = s.Ops[t-1], s.Ops[t]
				t--
			}
			return "push earlier"
		}
	case 7: // swap two adjacent drawing ops
		if at+1 < n && !isClip(s.Ops[at].Kind) && !isClip(s.Ops[at+1].Kind) {
			s.Ops[at], s.Ops[at+1] = s.Ops[at+1], s.Ops[at]
			return "swap"
		}
	case 8: // change a clip
		for range 10 {
			i := m.rnd.IntN(n)
			if s.Ops[i].Kind == scene.OpPushClip {
				c := m.clip()
				switch m.rnd.IntN(3) {
				case 0:
					s.Ops[i].Radii = c.Radii
				case 1:
					s.Ops[i].Transform = c.Transform
				default:
					s.Ops[i].Rect.X += m.rnd.Float32()*2 - 1
				}
				return "clip changed"
			}
		}
	case 9: // wrap a run of drawing ops in a clip
		end := at
		for end < n && end-at < 1+m.rnd.IntN(5) && !isClip(s.Ops[end].Kind) {
			end++
		}
		ops := append([]scene.Op(nil), s.Ops[:at]...)
		ops = append(ops, m.clip())
		ops = append(ops, s.Ops[at:end]...)
		ops = append(ops, scene.Op{Kind: scene.OpPopClip})
		s.Ops = append(ops, s.Ops[end:]...)
		return "wrap"
	default: // unwrap a clip
		for p := range n {
			if s.Ops[p].Kind == scene.OpPushClip && m.rnd.IntN(2) == 0 {
				if q := matching(s.Ops, p); q > p {
					s.Ops = append(s.Ops[:q], s.Ops[q+1:]...)
					s.Ops = append(s.Ops[:p], s.Ops[p+1:]...)
					return "unwrap"
				}
			}
		}
	}
	return "nothing"
}

// TestAdvFuzzDamage changes scenes of nested, rounded, fractional, translated
// and rotated clips, holes, backdrop effects and runs of identical ops, in one
// to three places each frame, and checks the damage redraws what a whole drawing
// does (skipping the shortcut that draws all of a window past half of it).
func TestAdvFuzzDamage(t *testing.T) {
	seeds := advSeeds
	for _, effects := range []bool{false, true} {
		fails := 0
		for seed := range uint64(seeds) {
			m := &advMaker{rnd: rand.New(rand.NewPCG(seed, 11)), effects: effects}
			s := m.scene()
			var r Renderer
			r.Render(s)
			for step := range 30 {
				prev := clone(s)
				var what []string
				for range 1 + m.rnd.IntN(3) {
					what = append(what, m.mutate(s))
				}
				ok, d, px := redrawDamage(&r, s)
				if !ok {
					fails++
					if fails <= 5 {
						t.Errorf("effects %v seed %d step %d %v: %d px differ; damage %v\n old %v\n new %v", effects, seed, step, what, px, d, kinds(prev), kinds(s))
					}
					r = Renderer{}
					r.Render(s)
				}
			}
		}
		t.Logf("effects %v: %d seeds x 30 steps, %d failing frames", effects, seeds, fails)
	}
}

func kinds(s *scene.Scene) string {
	b := []byte{}
	for _, op := range s.Ops {
		b = append(b, "FSGIP)EH?"[min(int(op.Kind), 8)])
	}
	return string(b)
}

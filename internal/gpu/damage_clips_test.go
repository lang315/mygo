package gpu

import (
	"fmt"
	"image"
	"math/rand/v2"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// clipMaker makes scenes of nested clips, rounded or not, whole or
// fractional, plain, moved or turned, holding fills of the same kinds.
type clipMaker struct{ rnd *rand.Rand }

func (m *clipMaker) transform() scene.Affine {
	switch m.rnd.IntN(4) {
	case 1:
		return scene.Translation(float32(m.rnd.IntN(40)), float32(m.rnd.IntN(40)))
	case 2:
		return scene.Translation(float32(m.rnd.IntN(40))+0.5, float32(m.rnd.IntN(40))+0.25)
	case 3:
		return scene.Rotation(float32(5 + m.rnd.IntN(30)))
	}
	return scene.Affine{}
}

func (m *clipMaker) fill() scene.Op {
	return scene.Op{Kind: scene.OpFill, Transform: m.transform(), Color: scene.Color{R: uint8(m.rnd.IntN(256)), A: 255},
		Rect: scene.Rect{X: float32(m.rnd.IntN(140)) + 0.5*float32(m.rnd.IntN(2)), Y: float32(m.rnd.IntN(140)), W: float32(10 + m.rnd.IntN(30)), H: float32(10 + m.rnd.IntN(30))}}
}

func (m *clipMaker) clip() scene.Op {
	op := scene.Op{Kind: scene.OpPushClip, Transform: m.transform(),
		Rect: scene.Rect{X: float32(m.rnd.IntN(60)) + 0.5*float32(m.rnd.IntN(2)), Y: float32(m.rnd.IntN(60)), W: float32(60 + m.rnd.IntN(60)), H: float32(60 + m.rnd.IntN(60))}}
	if m.rnd.IntN(2) == 0 {
		op.Radii = scene.Uniform(8)
	}
	return op
}

func (m *clipMaker) nest(ops []scene.Op, depth int) []scene.Op {
	for range 2 + m.rnd.IntN(4) {
		if depth < 3 && m.rnd.IntN(3) == 0 {
			ops = append(m.nest(append(ops, m.clip()), depth+1), scene.Op{Kind: scene.OpPopClip})
		} else {
			ops = append(ops, m.fill())
		}
	}
	return ops
}

// change deletes a fill, inserts one or gives one another transform: the
// clips stay, in the same order.
func (m *clipMaker) change(s *scene.Scene) string {
	var fills []int
	for i, op := range s.Ops {
		if op.Kind == scene.OpFill {
			fills = append(fills, i)
		}
	}
	switch m.rnd.IntN(3) {
	case 0:
		if len(fills) > 1 {
			i := fills[m.rnd.IntN(len(fills))]
			s.Ops = append(s.Ops[:i], s.Ops[i+1:]...)
			return "deleted"
		}
	case 1:
		if len(fills) > 0 {
			s.Ops[fills[m.rnd.IntN(len(fills))]].Transform = m.transform()
			return "moved"
		}
	}
	i := m.rnd.IntN(len(s.Ops) + 1)
	s.Ops = append(s.Ops[:i], append([]scene.Op{m.fill()}, s.Ops[i:]...)...)
	return "inserted"
}

// holders returns, for each clip of s in order, the ops it holds and
// whether the builder, folding translations or not, cuts it with a mask
// once content is drawn under a transform in it (encloses).
func holders(s *scene.Scene, fold bool) (held [][]int, encloses []bool, cut map[int]image.Rectangle) {
	var b Builder
	b.NoFold = !fold
	defer b.Release()
	b.Build(s, func(*scene.Image) uintptr { return 1 })
	var stack []int
	cut = map[int]image.Rectangle{} // what the clips around each op leave of it
	var cuts []image.Rectangle
	for i, op := range s.Ops {
		switch op.Kind {
		case scene.OpPushClip:
			stack = append(stack, len(held))
			held = append(held, nil)
			encloses = append(encloses, b.encloses[i])
			r := op.Transform.Bounds(op.Rect)
			c := image.Rect(int(r.X)-1, int(r.Y)-1, int(r.X+r.W)+2, int(r.Y+r.H)+2)
			if n := len(cuts); n > 0 {
				c = c.Intersect(cuts[n-1])
			}
			cuts = append(cuts, c)
		case scene.OpPopClip:
			stack = stack[:len(stack)-1]
			cuts = cuts[:len(cuts)-1]
		default:
			for _, k := range stack {
				held[k] = append(held[k], i)
			}
			if n := len(cuts); n > 0 {
				cut[i] = cuts[n-1]
			}
		}
	}
	return held, encloses, cut
}

// TestDamageFollowsTheClipsTheBuilderCutsWithAMask checks that when a
// change makes the builder cut a clip with a mask, or stop, which differs
// from the shader at its edge by a unit in the last place, the damage
// holds everything the clip holds: for clips rounded or not, whole or
// fractional, plain, only moved or turned, with translations folded or not.
func TestDamageFollowsTheClipsTheBuilderCutsWithAMask(t *testing.T) {
	for _, fold := range []bool{false, true} {
		t.Run(fmt.Sprint("fold=", fold), func(t *testing.T) {
			defer scene.FoldingForTest(fold)()
			fails, flips := 0, 0
			for seed := range uint64(400) {
				m := &clipMaker{rnd: rand.New(rand.NewPCG(seed, 5))}
				s := &scene.Scene{Width: 200, Height: 200, Ops: m.nest(nil, 0)}
				for step := range 20 {
					prev := &scene.Scene{Width: 200, Height: 200, Ops: append([]scene.Op(nil), s.Ops...)}
					what := m.change(s)
					_, encA, _ := holders(prev, fold)
					heldB, encB, cut := holders(s, fold)
					d := scene.Differ{Reach: 2}
					d.Diff(prev)
					d.Remember(prev)
					damage, ok := d.Diff(s)
					if !ok {
						continue
					}
					for k := range encA {
						if encA[k] == encB[k] {
							continue
						}
						flips++
						for _, i := range heldB[k] {
							op := s.Ops[i]
							r := op.Transform.Bounds(op.Rect)
							box := image.Rect(int(r.X), int(r.Y), int(r.X+r.W)+1, int(r.Y+r.H)+1).Intersect(image.Rect(0, 0, 200, 200)).Intersect(cut[i])
							for y := box.Min.Y; y < box.Max.Y; y++ {
								for x := box.Min.X; x < box.Max.X; x++ {
									in := false
									for _, dr := range damage {
										in = in || image.Pt(x, y).In(dr)
									}
									if !in {
										fails++
										if fails <= 3 {
											t.Errorf("seed %d step %d (%s): clip %d changed its cut, damage %v misses (%d,%d) of op %d", seed, step, what, k, damage, x, y, i)
										}
										x, y = box.Max.X, box.Max.Y
									}
								}
							}
						}
					}
				}
			}
			if flips == 0 {
				t.Error("no clip changed its cut: the scenes test nothing")
			}
			t.Logf("%d clips changed their cut, %d ops outside the damage", flips, fails)
		})
	}
}

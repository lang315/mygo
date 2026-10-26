package scene

import (
	"image"
	"math/rand/v2"
	"testing"
)

func damaged(r image.Rectangle, dmg []image.Rectangle) bool {
	if r.Empty() {
		return true
	}
	for _, d := range dmg {
		if r.In(d) {
			return true
		}
	}
	return false
}

// TestMatchIsConsistentWithTheDamage: Match agrees with the damage: a matched pair is the same op with the same
// bounds, or both its bounds are damaged; every op of either scene not
// matched is damaged; the mapping is strictly increasing.
func TestMatchIsConsistentWithTheDamage(t *testing.T) {
	multi := 0
	for seed := range uint64(3000) {
		rnd := rand.New(rand.NewPCG(seed, 3))
		rect := func() Rect {
			return Rect{X: rnd.Float32() * 200, Y: rnd.Float32() * 150, W: 1 + rnd.Float32()*30, H: 1 + rnd.Float32()*30}
		}
		var ops []Op
		depth := 0
		for range 30 + rnd.IntN(30) {
			switch x := rnd.IntN(8); {
			case x == 0 && depth < 3:
				ops = append(ops, Op{Kind: OpPushClip, Rect: Rect{X: rnd.Float32() * 100, Y: rnd.Float32() * 80, W: 100, H: 80}})
				depth++
			case x == 1 && depth > 0:
				ops = append(ops, Op{Kind: OpPopClip})
				depth--
			case x == 2 && len(ops) > 0:
				ops = append(ops, ops[len(ops)-1])
			default:
				ops = append(ops, Op{Kind: OpFill, Rect: rect(), Color: Color{R: uint8(rnd.IntN(4)), A: 255}})
			}
		}
		for ; depth > 0; depth-- {
			ops = append(ops, Op{Kind: OpPopClip})
		}
		old := &Scene{Width: 240, Height: 180, Ops: ops}
		nw := &Scene{Width: 240, Height: 180, Ops: append([]Op(nil), ops...)}
		for range 1 + rnd.IntN(4) {
			at := rnd.IntN(len(nw.Ops) + 1)
			switch rnd.IntN(4) {
			case 0:
				nw.Ops = append(nw.Ops[:at], append([]Op{{Kind: OpFill, Rect: rect(), Color: Color{G: 9, A: 255}}}, nw.Ops[at:]...)...)
			case 1:
				if at < len(nw.Ops) && nw.Ops[at].Kind == OpFill {
					nw.Ops = append(nw.Ops[:at], nw.Ops[at+1:]...)
				}
			case 2:
				if at < len(nw.Ops) && nw.Ops[at].Kind == OpFill {
					nw.Ops[at].Color.R++
				}
			default:
				if at < len(nw.Ops) && nw.Ops[at].Kind == OpPushClip {
					nw.Ops[at].Rect.X += 3
				}
			}
		}
		var d Differ
		d.Reach = 1
		d.Diff(old)
		d.Remember(old)
		dmg, ok := d.Diff(nw)
		if !ok {
			continue
		}
		if d.multi {
			multi++
		}
		usedOld := make([]bool, len(old.Ops))
		last := -1
		for j := range nw.Ops {
			i, ok := d.Match(j)
			if !ok {
				if !damaged(d.next[j], dmg) {
					t.Fatalf("seed %d: unmatched new op %d %v not damaged %v", seed, j, d.next[j], dmg)
				}
				continue
			}
			if i <= last || i >= len(old.Ops) {
				t.Fatalf("seed %d: Match(%d)=%d after %d", seed, j, i, last)
			}
			last = i
			usedOld[i] = true
			if old.Ops[i] != nw.Ops[j] || d.bounds[i] != d.next[j] {
				if !damaged(d.bounds[i], dmg) || !damaged(d.next[j], dmg) {
					t.Fatalf("seed %d: Match(%d)=%d differs and is not damaged", seed, j, i)
				}
			}
		}
		for i, u := range usedOld {
			if !u && !damaged(d.bounds[i], dmg) {
				t.Fatalf("seed %d: old op %d unmatched, not damaged", seed, i)
			}
		}
	}
	t.Logf("%d of 3000 diffs in multi-range mode", multi)
}

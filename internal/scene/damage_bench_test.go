package scene

import "testing"

// benchScene makes a scene of n fills in rows of a page, every tenth wrapped
// in a clip: about what a table of text and boxes draws.
func benchScene(n int) *Scene {
	s := &Scene{Width: 1280, Height: 800}
	for i := 0; i < n; i++ {
		r := Rect{X: float32(i % 60 * 20), Y: float32(i / 60 % 40 * 20), W: 18, H: 18}
		if i%10 == 0 {
			s.Ops = append(s.Ops, Op{Kind: OpPushClip, Rect: r})
		}
		s.Ops = append(s.Ops, Op{Kind: OpFill, Rect: r, Color: Color{R: uint8(i), G: uint8(i >> 8), A: 255}})
		if i%10 == 0 {
			s.Ops = append(s.Ops, Op{Kind: OpPopClip})
		}
	}
	return s
}

// withInserted returns a copy of s with fills inserted before the ops at
// each of the places, in descending order.
func withInserted(s *Scene, places ...int) *Scene {
	c := *s
	c.Ops = append([]Op(nil), s.Ops...)
	for _, at := range places {
		c.Ops = append(c.Ops[:at], append([]Op{{Kind: OpFill, Rect: Rect{X: 5, Y: 5, W: 50, H: 9}, Color: Color{B: 200, A: 255}}}, c.Ops[at:]...)...)
	}
	return &c
}

// BenchmarkDiff10k compares scenes of 10,000 fills: the same twice, one op
// changed, one op inserted, and two inserted in two places (a pressed row
// and an overlay scrollbar).
func BenchmarkDiff10k(b *testing.B) {
	base := benchScene(10_000)
	changed := withInserted(base)
	changed.Ops[len(changed.Ops)/2].Color = Color{R: 1, G: 2, B: 3, A: 255}
	n := len(base.Ops)
	for _, tc := range []struct {
		name string
		next *Scene
	}{
		{"identical", withInserted(base)},
		{"one-changed", changed},
		{"one-inserted", withInserted(base, n/3)},
		{"two-inserted", withInserted(base, n, n/3)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			d := Differ{Reach: 1}
			d.Diff(base)
			d.Remember(base)
			b.ReportAllocs()
			for b.Loop() {
				d.Diff(tc.next)
			}
		})
	}
}

// Worst cases for align: 17 changes spread over 10k ops with as many ops
// (align walks all the middle, fails on the 17th, the pairwise path walks it
// again), and many inserted runs that align does resync.
func BenchmarkDiffAdv10k(b *testing.B) {
	base := benchScene(10_000)
	n := len(base.Ops)
	spread := withInserted(base)
	for k := 1; k <= 17; k++ {
		spread.Ops[k*n/18+1].Color = Color{R: 9, A: 255}
	}
	var places []int
	for k := 60; k >= 1; k-- {
		places = append(places, k*n/61)
	}
	for _, tc := range []struct {
		name string
		next *Scene
	}{
		{"17-changed-spread", spread},
		{"60-inserted-spread", withInserted(base, places...)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			d := Differ{Reach: 1}
			d.Diff(base)
			d.Remember(base)
			b.ReportAllocs()
			for b.Loop() {
				d.Diff(tc.next)
			}
		})
	}
}

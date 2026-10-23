package ui

import (
	"testing"
	"time"
)

// BenchmarkTransitionAtRest builds, lays out and paints the frames of 1000
// boxes in a scroll view, with and without a Colors Transition on each,
// once the pointer moved and every fade is over: nothing animates, so the
// Transition should cost next to nothing.
func BenchmarkTransitionAtRest(b *testing.B) {
	for _, c := range []struct {
		name string
		tr   *ElementTransition
	}{
		{"none", nil},
		{"colors", &ElementTransition{Colors: true}},
	} {
		b.Run(c.name, func(b *testing.B) {
			view := func(ctx *context) {
				coreScroll(ctx).Size(400, 600).Children(func() {
					for i := range 1000 {
						e := coreBox(ctx).Key(i).Size(380, 24).Background(Color{R: 200, G: 200, B: 220, A: 255})
						if c.tr != nil {
							e.Transition(*c.tr)
						}
					}
				})
			}
			tt := coreNewTester(view, 400, 600)
			tt.Move(10, 10)
			tt.Advance(time.Second) // the Tester's clock does not run inside b.Loop
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				tt.h.img.Invalidate()
				tt.Frame()
			}
		})
	}
}

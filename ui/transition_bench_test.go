package ui

import (
	"testing"
	"time"
)

// BenchmarkTransitionAtRest builds, lays out and paints the frames of 1000
// boxes in a scroll view, with and without a Colors Transition on each,
// once the pointer moved and every fade is over: nothing animates, so the
// Transition should cost next to nothing. In the visible- cases every box
// shows; in the others few do, which measures culling as well.
func BenchmarkTransitionAtRest(b *testing.B) {
	// The boxes of the rows are 24 DIPs high, so that about 25 show; of the
	// visible ones, half a DIP, so that they all do.
	for _, c := range []struct {
		name   string
		tr     *ElementTransition
		height float32
	}{
		{"none", nil, 24},
		{"colors", &ElementTransition{Colors: true}, 24},
		{"visible-none", nil, 0.5},
		{"visible-colors", &ElementTransition{Colors: true}, 0.5},
	} {
		b.Run(c.name, func(b *testing.B) {
			view := func(ctx *context) {
				coreScroll(ctx).Size(400, 600).Children(func() {
					for i := range 1000 {
						e := coreBox(ctx).Key(i).Size(380, c.height).Background(Color{R: 200, G: 200, B: 220, A: 255})
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

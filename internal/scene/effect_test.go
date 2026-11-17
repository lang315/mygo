package scene

import (
	"math"
	"testing"
)

// A blur that is infinite, not a number or huge gives a finite backdrop
// within the cap, so that renderers' loops over its radius end.
func TestBackdropOfBadBlur(t *testing.T) {
	for _, blur := range []float32{float32(math.Inf(1)), float32(math.NaN()), 1e9, -1} {
		b := BackdropOf(Rect{X: 10, Y: 10, W: 20, H: 20}, blur, 100, 100)
		if b.Radius < 0 || b.Radius > 4096 {
			t.Errorf("blur %v: Radius %d, want within [0, 4096]", blur, b.Radius)
		}
		if math.IsNaN(float64(b.Sigma)) || math.IsInf(float64(b.Sigma), 0) || b.Sigma < 0 {
			t.Errorf("blur %v: Sigma %v, want finite", blur, b.Sigma)
		}
	}
	// The cap itself is not lowered: 4096 blurs as the formula gives it,
	// at an eighth of the size, the average's variance left out.
	r := Rect{X: 10, Y: 10, W: 20, H: 20}
	b := BackdropOf(r, 4096, 100000, 100000)
	if want := float32(math.Sqrt(4096*4096-64.0/12)) / 8; b.Down != 8 || math.Abs(float64(b.Sigma-want)) > 1e-3 {
		t.Errorf("blur 4096: Down %d, Sigma %v, want 8 and %v", b.Down, b.Sigma, want)
	}
	// Above the cap, a blur is the cap's.
	for _, blur := range []float32{float32(math.Inf(1)), 1e9, 4097} {
		if got, want := BackdropOf(r, blur, 100000, 100000), BackdropOf(r, 4096, 100000, 100000); got != want {
			t.Errorf("blur %v: %+v, want the cap's %+v", blur, got, want)
		}
	}
	// NaN and below zero blur nothing.
	for _, blur := range []float32{float32(math.NaN()), -1} {
		if b := BackdropOf(Rect{W: 10, H: 10}, blur, 100, 100); b.Radius != 0 || b.Sigma != 0 || b.Down != 1 {
			t.Errorf("blur %v: %+v, want no blur", blur, b)
		}
	}
}

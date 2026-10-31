package ui

import (
	"math"
	"testing"
)

// fraction places a value of a Slider, a Progress-like indicator or a color channel
// from 0 to 1, and its result becomes the width of a rect the painter fills: a
// value that is not a number (a division of the app's that went wrong) puts it at
// the start, as a range with nothing in it does, and never makes a NaN size.
func TestFractionIsAlwaysBetweenZeroAndOne(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	for _, tc := range []struct {
		name      string
		v, lo, hi float64
		want      float32
	}{
		{"inside", 25, 0, 100, 0.25},
		{"below", -5, 0, 100, 0},
		{"above", 500, 0, 100, 1},
		{"empty range", 5, 10, 10, 0},
		{"reversed range", 5, 10, 0, 0},
		{"NaN value", nan, 0, 100, 0},
		{"NaN lo", 5, nan, 100, 0},
		{"NaN hi", 5, 0, nan, 0},
		{"infinite value", inf, 0, 100, 1},
		{"minus infinite value", -inf, 0, 100, 0},
		{"infinite range", 5, 0, inf, 0},
		{"infinite value over infinite range", inf, 0, inf, 0},
	} {
		if got := fraction(tc.v, tc.lo, tc.hi); got != tc.want {
			t.Errorf("%s: fraction(%v, %v, %v) = %v, want %v", tc.name, tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}

package ui

import (
	"image"
	"testing"
)

// A row whose path spills into a layer of paths drawn on the CPU must not
// replay: the layer holds only what its frame painted. Each painting of an
// element's paths starts a new run of layers (Painter.draw), so a row's
// path opens a layer of its own, whose image operation keeps the row from
// recording, rather than going into the layer of the row above.
func TestRowsWithSpilledPathsPaintAgain(t *testing.T) {
	defer func(was bool) { spillEveryPath = was }(spillEveryPath)
	spillEveryPath = true
	var s OutlineState[int]
	roots := make([]int, 40)
	for i := range roots {
		roots[i] = i
	}
	tt := coreNewTester(func(c *context) {
		// Rows of an arrow alone: each row's first paint is its path.
		coreOutline(c, &s, roots, func(int) []int { return []int{} }, func(int) {}).Grow(1)
	}, 200, 400)
	frame := func(replay bool) *image.RGBA {
		tt.rt.noReplay = !replay
		for range 3 {
			tt.Frame()
		}
		return tt.Image()
	}
	painted := frame(false)
	replayed := frame(true)
	if tt.rt.paths.spill.opened == 0 {
		t.Fatal("no path spilled")
	}
	if tt.rt.replayed != 0 {
		t.Errorf("%d rows replayed with their paths in a layer", tt.rt.replayed)
	}
	if b := painted.Bounds(); b != replayed.Bounds() {
		t.Fatalf("bounds %v and %v", b, replayed.Bounds())
	}
	for i := range painted.Pix {
		if painted.Pix[i] != replayed.Pix[i] {
			x, y := i/4%painted.Bounds().Dx(), i/4/painted.Bounds().Dx()
			t.Fatalf("pixel (%d, %d) painted %v, replayed %v", x, y, painted.RGBAAt(x, y), replayed.RGBAAt(x, y))
		}
	}
}

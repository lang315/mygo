package ui

import (
	"bytes"
	"testing"
)

func skipView(clicked *int) func(c *context) {
	return func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			if coreButton(c, "Go").Clicked() {
				*clicked++
			}
			coreText(c, "hello")
		})
	}
}

// A tester that skips the rasterizer builds, lays out and paints every frame
// and handles input as before, but draws no pixels until Image is asked for,
// and then the last frame, the same as one that drew every frame.
func TestTesterSkipRasterDrawsOnlyWhenAsked(t *testing.T) {
	var n1, n2 int
	drawn := coreNewTester(skipView(&n1), 300, 200)
	skipped := coreNewTester(skipView(&n2), 300, 200)
	skipped.SkipRaster(true)
	for _, tt := range []*Tester{drawn, skipped} {
		tt.Frame()
		if err := tt.Click("Go"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	if n1 != 1 || n2 != 1 {
		t.Fatalf("clicks seen %d and %d, want 1 and 1: input is handled as before", n1, n2)
	}
	if !skipped.h.stale {
		t.Fatal("the skipping tester drew its frames")
	}
	if drawn.h.stale {
		t.Fatal("the other tester did not draw its frames")
	}
	want, got := drawn.Image(), skipped.Image()
	if !bytes.Equal(want.Pix, got.Pix) {
		t.Fatal("Image of the skipping tester is not the last frame as the other draws it")
	}
	if skipped.h.stale {
		t.Fatal("Image left the frame to be drawn")
	}
}

// Skipping can be turned off: the frames after it draw as usual.
func TestTesterSkipRasterCanBeTurnedOff(t *testing.T) {
	var n int
	tt := coreNewTester(skipView(&n), 300, 200)
	tt.SkipRaster(true)
	tt.Frame()
	tt.SkipRaster(false)
	tt.Frame()
	if tt.h.stale {
		t.Fatal("a frame after SkipRaster(false) was not drawn")
	}
}

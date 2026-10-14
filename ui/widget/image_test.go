package widget

import (
	"crypto/sha256"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// testPNG is in helpers_test.go (Task 8).

func TestImageTakesItsSize(t *testing.T) {
	for name, data := range map[string][]byte{
		"png": testPNG(t, color.RGBA{255, 0, 0, 255}),
		"bad": []byte("not an image"),
	} {
		tt := ui.NewTester(View(HBox(0).Add(Image(data, 40, 20), Label("after"))), 800, 600)
		tt.Frame()
		if a := find(t, tt, "after"); a.X < 39.5 || a.X > 40.5 {
			t.Errorf("%s: after at %v, want x 40", name, a)
		}
	}
}

func TestImageCacheDecodesOnce(t *testing.T) {
	data := testPNG(t, color.RGBA{0, 255, 0, 255})
	if a, b := decodeImage(data), decodeImage(data); a == nil || a != b {
		t.Fatalf("decoded %p then %p, want one bitmap", a, b)
	}
}

func TestImageCacheIsBounded(t *testing.T) {
	first := testPNG(t, color.RGBA{1, 2, 3, 255})
	decodeImage(first)
	for i := range imageCacheSize {
		decodeImage(testPNG(t, color.RGBA{uint8(i), uint8(i >> 8), 200, 255}))
	}
	imageCache.Lock()
	n := imageCache.order.Len()
	_, kept := imageCache.m[sha256.Sum256(first)]
	imageCache.Unlock()
	if n != imageCacheSize || kept {
		t.Fatalf("%d entries, oldest kept %v: want %d, evicted", n, kept, imageCacheSize)
	}
}

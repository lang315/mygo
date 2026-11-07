package gifvideo

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"io"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

var (
	videoRed   = color.RGBA{255, 0, 0, 255}
	videoGreen = color.RGBA{0, 255, 0, 255}
	videoBlue  = color.RGBA{0, 0, 255, 255}
	videoWhite = color.RGBA{255, 255, 255, 255}
)

// Importing this package is enough for ui to read a GIF: its test binary
// imports no other GIF decoder, and DecodeBitmap and DecodeBitmapConfig read
// a small GIF.
func TestImportingGifvideoLetsUIReadGIFs(t *testing.T) {
	g := &gif.GIF{
		Config: image.Config{Width: 3, Height: 2},
		Image:  []*image.Paletted{image.NewPaletted(image.Rect(0, 0, 3, 2), color.Palette{videoRed})},
		Delay:  []int{10},
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	if cfg, err := ui.DecodeBitmapConfig(b.Bytes()); err != nil || cfg.Width != 3 || cfg.Height != 2 {
		t.Errorf("DecodeBitmapConfig: %v, %v", cfg, err)
	}
	bm, err := ui.DecodeBitmap(b.Bytes())
	if err != nil || bm == nil {
		t.Fatalf("DecodeBitmap: %v, %v", bm, err)
	}
	if w, h := bm.Size(); w != 3 || h != 2 {
		t.Errorf("the bitmap is %d by %d, want 3 by 2", w, h)
	}
}

// TestDecoderFrames checks that a GIF's frames draw over what the ones
// before left as their disposal says, with their transparent pixels
// showing what is under them, for as long as their delays say, and that
// the decoder starts over after Rewind.
func TestDecoderFrames(t *testing.T) {
	transparent := color.RGBA{}
	full := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{videoRed})
	// Green over the top left, but for a transparent pixel at 1,1.
	corner := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{videoGreen, transparent})
	corner.SetColorIndex(1, 1, 1)
	low := image.NewPaletted(image.Rect(2, 2, 4, 4), color.Palette{videoBlue})
	dot := image.NewPaletted(image.Rect(3, 0, 4, 1), color.Palette{videoWhite})
	g := &gif.GIF{
		Config:   image.Config{Width: 4, Height: 4},
		Image:    []*image.Paletted{full, corner, low, dot},
		Delay:    []int{0, 5, 20, 1},
		Disposal: []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious, gif.DisposalNone},
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	d, err := NewDecoder(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	type px struct {
		x, y int
		c    color.RGBA
	}
	frames := []struct {
		d    time.Duration
		want []px
	}{
		{100 * time.Millisecond, []px{{0, 0, videoRed}, {3, 3, videoRed}}},
		{50 * time.Millisecond, []px{{0, 0, videoGreen}, {1, 1, videoRed}, {3, 3, videoRed}}},
		// The green corner went, leaving nothing; blue covers the bottom
		// right.
		{200 * time.Millisecond, []px{{0, 0, transparent}, {1, 1, transparent}, {3, 3, videoBlue}, {3, 0, videoRed}}},
		// The blue went back to what was under it.
		{100 * time.Millisecond, []px{{3, 0, videoWhite}, {3, 3, videoRed}, {0, 0, transparent}}},
	}
	check := func(round int) {
		for i, want := range frames {
			f, err := d.Next()
			if err != nil {
				t.Fatalf("round %d, frame %d: %v", round, i, err)
			}
			if f.Duration != want.d {
				t.Errorf("round %d, frame %d shows for %v, want %v", round, i, f.Duration, want.d)
			}
			img := f.Image.(*image.RGBA)
			for _, p := range want.want {
				if got := img.RGBAAt(p.x, p.y); got != p.c {
					t.Errorf("round %d, frame %d at %d,%d: %v, want %v", round, i, p.x, p.y, got, p.c)
				}
			}
		}
		if _, err := d.Next(); err != io.EOF {
			t.Errorf("round %d: after the last frame, %v", round, err)
		}
	}
	check(1)
	if err := d.Rewind(); err != nil {
		t.Fatal(err)
	}
	check(2)

	if _, err := NewDecoder([]byte("GIF89a, not quite")); err == nil {
		t.Error("data that is no GIF decoded")
	}
}

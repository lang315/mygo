package gifvideo

import (
	"bytes"
	"compress/lzw"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"io"
	"runtime"
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

// gifOf returns a GIF of a logical screen of sw×sh and one frame of fw×fh,
// whose data is lzw: a few bytes however large they are.
func gifOf(sw, sh, fw, fh int, lzw []byte) []byte {
	b := []byte("GIF89a")
	b = binary.LittleEndian.AppendUint16(b, uint16(sw))
	b = binary.LittleEndian.AppendUint16(b, uint16(sh))
	b = append(b, 0x80, 0, 0, 0, 0, 0, 255, 255, 255) // a table of two colors
	b = append(b, 0x2C, 0, 0, 0, 0)
	b = binary.LittleEndian.AppendUint16(b, uint16(fw))
	b = binary.LittleEndian.AppendUint16(b, uint16(fh))
	b = append(b, 0)
	b = append(b, lzw...)
	return append(b, 0x3B)
}

// A GIF whose logical screen or whose frame has more than ui.MaxImagePixels
// is refused from its headers, before the memory for it is allocated.
func TestDecoderRefusesHugeGIF(t *testing.T) {
	oneByOne := []byte{2, 2, 0x44, 0x01, 0}
	allocated := func(data []byte) (err error, n uint64) {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		_, err = NewDecoder(data)
		runtime.ReadMemStats(&after)
		return err, after.TotalAlloc - before.TotalAlloc
	}
	// image/gif refuses a frame that does not fit the logical screen, so the
	// screen's size bounds the frames' too.
	err, n := allocated(gifOf(8193, 8192, 1, 1, oneByOne))
	if err == nil {
		t.Error("a GIF with a screen past MaxImagePixels decoded")
	}
	if n > 16<<20 {
		t.Errorf("refusing a GIF with a screen past MaxImagePixels allocated %d MB", n>>20)
	}
	if _, err := NewDecoder(gifOf(0, 0, 16384, 8192, []byte{2, 0})); err == nil {
		t.Error("a GIF with a frame past MaxImagePixels decoded")
	}
	// At the cap, and a small one.
	if _, err := NewDecoder(gifOf(8192, 8192, 1, 1, oneByOne)); err != nil {
		t.Errorf("a GIF of MaxImagePixels: %v", err)
	}
	if _, err := NewDecoder(gifOf(2, 2, 1, 1, oneByOne)); err != nil {
		t.Errorf("a small GIF: %v", err)
	}
}

// solidGIF returns a GIF of 4×4 frames of solid colors, each showing for
// delay hundredths of a second.
func solidGIF(t *testing.T, delay int, colors ...color.RGBA) []byte {
	t.Helper()
	g := &gif.GIF{Config: image.Config{Width: 4, Height: 4}}
	for _, c := range colors {
		f := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{c})
		g.Image = append(g.Image, f)
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalNone)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestGIFDecoderEndsEarly checks that a GIF that ends early, with no
// trailer or cut off in a frame, plays the frames it has whole, as
// browsers do, and that one with no whole frame is an error.
func TestGIFDecoderEndsEarly(t *testing.T) {
	data := solidGIF(t, 10, videoRed, videoGreen, videoBlue)
	for _, tc := range []struct {
		name string
		data []byte
		want []color.RGBA
	}{
		{"no trailer", data[:len(data)-1], []color.RGBA{videoRed, videoGreen, videoBlue}},
		// Without the last frame's block terminator and a byte of its
		// pixels.
		{"cut off in the last frame", data[:len(data)-3], []color.RGBA{videoRed, videoGreen}},
	} {
		d, err := NewDecoder(tc.data)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		for i, want := range tc.want {
			f, err := d.Next()
			if err != nil {
				t.Fatalf("%s, frame %d: %v", tc.name, i, err)
			}
			if got := f.Image.(*image.RGBA).RGBAAt(1, 1); got != want {
				t.Errorf("%s, frame %d: %v, want %v", tc.name, i, got, want)
			}
		}
		if _, err := d.Next(); err != io.EOF {
			t.Errorf("%s: after %d frames, %v", tc.name, len(tc.want), err)
		}
	}
	if _, err := NewDecoder(data[:40]); err == nil {
		t.Error("a GIF cut off in its first frame decoded")
	}
}

// TestGIFDecoderPixelsPastColors checks that a GIF whose pixels point past
// its color table plays, those pixels black, as Chrome draws them, where
// image/gif decodes none of it.
func TestGIFDecoderPixelsPastColors(t *testing.T) {
	// A 2×2 frame of pixels 0 to 3, its color table red and green.
	var px bytes.Buffer
	w := lzw.NewWriter(&px, lzw.LSB, 2)
	w.Write([]byte{0, 1, 2, 3})
	w.Close()
	frame := func(table []byte) []byte {
		b := []byte{0x2c, 0, 0, 0, 0, 2, 0, 2, 0, 0}
		if table != nil {
			b[9] = 0x80
			b = append(b, table...)
		}
		b = append(b, 2, byte(px.Len()))
		b = append(b, px.Bytes()...)
		return append(b, 0)
	}
	colors := []byte{255, 0, 0, 0, 255, 0}
	screen := []byte("GIF89a\x02\x00\x02\x00\x00\x00\x00")
	global := append(append([]byte{}, screen...), colors...)
	global[10] = 0x80
	black := color.RGBA{0, 0, 0, 255}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"global color table", append(append(global, frame(nil)...), 0x3b)},
		{"local color table", append(append(append([]byte{}, screen...), frame(colors)...), 0x3b)},
	} {
		d, err := NewDecoder(tc.data)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		f, err := d.Next()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		pic := f.Image.(*image.RGBA)
		for i, want := range []color.RGBA{videoRed, videoGreen, black, black} {
			if got := pic.RGBAAt(i%2, i/2); got != want {
				t.Errorf("%s: pixel %d is %v, want %v", tc.name, i, got, want)
			}
		}
	}
}

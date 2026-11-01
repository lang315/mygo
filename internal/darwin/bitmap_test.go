//go:build darwin

package darwin

import (
	"testing"
	"unsafe"
)

// TestNSImageFromBGRA reads the pixels back out of the image's bitmap: the
// preview arrives as premultiplied BGRA and AppKit's default bitmap is
// premultiplied RGBA, so B and R swap and nothing else changes.
func TestNSImageFromBGRA(t *testing.T) {
	load()
	// 3×2, rows of distinct colours with B≠R and partial alpha.
	pix := []byte{
		10, 20, 30, 255, 0, 0, 0, 0, 40, 50, 60, 128,
		1, 2, 3, 4, 200, 100, 50, 255, 9, 8, 7, 9,
	}
	withPool(func() {
		img := nsImageFromBGRA(pix, 3, 2)
		if img == 0 {
			t.Fatal("no image")
		}
		size := msgSize(img, sel("size"))
		if size.Width != 3 || size.Height != 2 {
			t.Fatalf("size %v, want 3×2", size)
		}
		reps := send(img, "representations")
		if n := sendInt(reps, "count"); n != 1 {
			t.Fatalf("%d representations, want 1", n)
		}
		rep := send(reps, "objectAtIndex:", 0)
		if w, h := sendInt(rep, "pixelsWide"), sendInt(rep, "pixelsHigh"); w != 3 || h != 2 {
			t.Fatalf("rep %d×%d, want 3×2", w, h)
		}
		stride := sendInt(rep, "bytesPerRow")
		data := send(rep, "bitmapData")
		got := unsafe.Slice(*(**byte)(unsafe.Pointer(&data)), stride*2)
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				s := pix[y*12+x*4:]
				d := got[y*stride+x*4:]
				if d[0] != s[2] || d[1] != s[1] || d[2] != s[0] || d[3] != s[3] {
					t.Errorf("pixel %d,%d = %v, want RGBA of BGRA %v", x, y, d[:4], s[:4])
				}
			}
		}
	})
}

func TestNSImageFromBGRAWrongLength(t *testing.T) {
	load()
	withPool(func() {
		for _, c := range []struct {
			pix  []byte
			w, h int
		}{{make([]byte, 3), 1, 1}, {make([]byte, 8), 1, 1}, {nil, 0, 0}, {make([]byte, 4), 0, 1}} {
			if img := nsImageFromBGRA(c.pix, c.w, c.h); img != 0 {
				t.Errorf("%d bytes for %d×%d: got an image, want 0", len(c.pix), c.w, c.h)
			}
		}
	})
}

// The cursor that hides the pointer is an image of one transparent pixel.
func TestNoCursorIsTransparent(t *testing.T) {
	load()
	c := noCursor()
	if c == 0 || c == send(class("NSCursor"), "arrowCursor") {
		t.Fatalf("cursor %v", c)
	}
	if noCursor() != c {
		t.Error("the cursor is made again")
	}
	withPool(func() {
		rep := send(send(send(c, "image"), "representations"), "objectAtIndex:", 0)
		stride := sendInt(rep, "bytesPerRow")
		data := send(rep, "bitmapData")
		got := unsafe.Slice(*(**byte)(unsafe.Pointer(&data)), stride)[:4]
		if w, h := sendInt(rep, "pixelsWide"), sendInt(rep, "pixelsHigh"); w != 1 || h != 1 || got[3] != 0 {
			t.Errorf("the cursor's image is %d×%d, pixel %v", w, h, got)
		}
	})
}

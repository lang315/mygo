//go:build linux && (amd64 || arm64)

package linux

import (
	"image/color"
	"testing"
)

// TestStraightRGBAMatchesNRGBAModel checks every premultiplied value against
// image/color, which is what png.Encode wrote for the drag icon before.
func TestStraightRGBAMatchesNRGBAModel(t *testing.T) {
	src := make([]byte, 4)
	dst := make([]byte, 4)
	for a := 0; a < 256; a++ {
		for c := 0; c <= a; c++ {
			src[0], src[1], src[2], src[3] = byte(c), byte(c/2), byte(a-c), byte(a) // B, G, R, A
			straightRGBA(dst, src)
			want := color.NRGBAModel.Convert(color.RGBA{R: src[2], G: src[1], B: src[0], A: src[3]}).(color.NRGBA)
			if dst[0] != want.R || dst[1] != want.G || dst[2] != want.B || dst[3] != want.A {
				t.Fatalf("BGRA %v: got %v, want %v", src, dst, want)
			}
		}
	}
}

func TestCopyRowsHonoursTheStride(t *testing.T) {
	const w, h, stride = 3, 2, 4*3 + 12
	src := make([]byte, 4*w*h)
	for i := range src {
		src[i] = byte(i + 1)
	}
	dst := make([]byte, stride*h)
	copyRows(dst, stride, src, w, h)
	for y := 0; y < h; y++ {
		for i := 0; i < 4*w; i++ {
			if dst[y*stride+i] != src[y*4*w+i] {
				t.Fatalf("row %d byte %d = %d, want %d", y, i, dst[y*stride+i], src[y*4*w+i])
			}
		}
		for i := 4 * w; i < stride; i++ {
			if dst[y*stride+i] != 0 {
				t.Fatalf("row %d padding byte %d written", y, i)
			}
		}
	}
}

func TestPixbufFromBGRARejectsABadLength(t *testing.T) {
	for _, c := range []struct {
		n, w, h int
	}{{3, 1, 1}, {8, 1, 1}, {0, 0, 0}, {4, 0, 1}} {
		if pb := pixbufFromBGRA(make([]byte, c.n), c.w, c.h); pb != 0 {
			t.Errorf("%d bytes for %d×%d: got a pixbuf", c.n, c.w, c.h)
		}
	}
}

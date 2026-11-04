package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

// halves is a side×side bitmap, red on its left half and blue on its right.
func halves(side int) *Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := range side {
		for x := range side {
			c := color.RGBA{R: 255, A: 255}
			if x >= side/2 {
				c = color.RGBA{B: 255, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return NewBitmap(img)
}

// Rotate turns a bitmap icon as it turns an SVG one: a quarter turn puts the
// left half on top.
func TestIconImageRotate(t *testing.T) {
	bmp := halves(32)
	var deg float32
	tt := NewTester(func(c *Context) {
		IconImage(c, bmp).FontSize(40).Rotate(deg)
	}, 100, 100)
	red := func(x, y int) bool { c := tt.Image().RGBAAt(x, y); return c.R > 200 && c.B < 50 }
	blue := func(x, y int) bool { c := tt.Image().RGBAAt(x, y); return c.B > 200 && c.R < 50 }
	if !red(10, 20) || !blue(30, 20) {
		t.Fatalf("unturned: %v left, %v right; want red and blue", tt.Image().RGBAAt(10, 20), tt.Image().RGBAAt(30, 20))
	}
	deg = 90
	tt.Frame()
	if !red(20, 10) || !blue(20, 30) {
		t.Fatalf("a quarter turn: %v top, %v bottom; want red and blue", tt.Image().RGBAAt(20, 10), tt.Image().RGBAAt(20, 30))
	}
}

// A content transition from a bitmap icon to an SVG one fades the bitmap out
// as the SVG comes in.
func TestIconImageContentTransitionToIcon(t *testing.T) {
	svg := MustParseSVG([]byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`))
	bmp := halves(32)
	useSVG := false
	tt, now := clockTester(func(c *context) {
		var e *node
		if useSVG {
			e = coreIcon(c, svg)
		} else {
			e = coreIconImage(c, bmp)
		}
		e.FontSize(40).ContentTransition(ContentTransition{Animation: &Animation{Duration: 100 * time.Millisecond, Ease: Linear}})
	}, 100, 100)
	ops := func() (images, glyphs int) {
		for _, op := range tt.h.last.Ops {
			switch op.Kind {
			case scene.OpImage:
				images++
			case scene.OpGlyphs:
				glyphs++
			}
		}
		return
	}
	if i, g := ops(); i != 1 || g != 0 {
		t.Fatalf("at rest: %d images, %d icons; want the bitmap only", i, g)
	}
	useSVG = true
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if i, g := ops(); i != 1 || g != 1 {
		t.Errorf("changing: %d images, %d icons; want the bitmap fading and the SVG", i, g)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if i, g := ops(); i != 0 || g != 1 {
		t.Errorf("changed: %d images, %d icons; want the SVG only", i, g)
	}
}

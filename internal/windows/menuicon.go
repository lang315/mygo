//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"image"
	"image/png"

	"golang.org/x/image/draw"
)

// menuBitmap returns a size×size 32-bit bitmap of a PNG for a menu item,
// premultiplied and top-down as menus draw them, or 0 for a PNG that
// does not decode. The caller deletes it.
func menuBitmap(b []byte, size int) uintptr {
	src, err := png.Decode(bytes.NewReader(b))
	if err != nil || size <= 0 {
		return 0
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(img, img.Bounds(), src, src.Bounds(), draw.Src, nil)
	bmp, px := newDIBSection(0, int32(size), int32(size))
	if bmp == 0 {
		return 0
	}
	for i := range px {
		p := img.Pix[i*4 : i*4+4] // RGBA, premultiplied
		px[i] = uint32(p[3])<<24 | uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
	}
	return bmp
}

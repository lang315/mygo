//go:build linux && (amd64 || arm64)

package linux

import (
	"image/color"
	"unsafe"
)

// straightRGBA converts premultiplied BGRA (platform.DragRequest.Preview) to
// the straight RGBA a GdkPixbuf holds, as image/color converts it, which is
// what the PNG of the drag icon carried before.
func straightRGBA(dst, src []byte) {
	for i := 0; i+3 < len(src); i += 4 {
		c := color.NRGBAModel.Convert(color.RGBA{R: src[i+2], G: src[i+1], B: src[i], A: src[i+3]}).(color.NRGBA)
		dst[i], dst[i+1], dst[i+2], dst[i+3] = c.R, c.G, c.B, c.A
	}
}

// copyRows copies h rows of w pixels into dst, whose rows are stride bytes
// apart (a GdkPixbuf may pad them).
func copyRows(dst []byte, stride int, src []byte, w, h int) {
	for y := 0; y < h; y++ {
		copy(dst[y*stride:y*stride+4*w], src[y*4*w:(y+1)*4*w])
	}
}

// pixbufFromBGRA returns a new GdkPixbuf, which the caller unrefs, of w×h
// pixels from premultiplied BGRA; 0 when the slice does not hold w×h pixels.
func pixbufFromBGRA(pix []byte, w, h int) ptr {
	if w <= 0 || h <= 0 || len(pix) != 4*w*h {
		return 0
	}
	const rgb = 0 // GDK_COLORSPACE_RGB
	pb := gdkPixbufNew(rgb, true, 8, int32(w), int32(h))
	if pb == 0 {
		return 0
	}
	stride := int(gdkPixbufGetRowstride(pb))
	// The last row of a GdkPixbuf is not padded: stride*(h-1) + 4*w bytes.
	dst := unsafe.Slice((*byte)(gdkPixbufGetPixels(pb)), stride*(h-1)+4*w)
	rgba := make([]byte, len(pix))
	straightRGBA(rgba, pix)
	copyRows(dst, stride, rgba, w, h)
	return pb
}

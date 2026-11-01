//go:build darwin

package darwin

import "unsafe"

// nsImageFromBGRA returns an autoreleased NSImage of w×h points holding the
// pixels, premultiplied BGRA as platform.DragRequest.Preview has them, in a
// bitmap of w×h pixels; 0 when the slice does not hold w×h pixels. It runs in
// a pool. AppKit takes the pixels as they are, with no PNG in between.
func nsImageFromBGRA(pix []byte, w, h int) id {
	if w <= 0 || h <= 0 || len(pix) != 4*w*h {
		return 0
	}
	rep := autorelease(msgInitBitmap(send(class("NSBitmapImageRep"), "alloc"),
		sel("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:"),
		0, w, h, 8, 4, true, false, nsString("NSCalibratedRGBColorSpace"), 4*w, 32))
	if rep == 0 {
		return 0
	}
	rep = send(rep, "bitmapImageRepByRetaggingWithColorSpace:", uintptr(send(class("NSColorSpace"), "sRGBColorSpace")))
	if rep == 0 {
		return 0
	}
	data := send(rep, "bitmapData")
	if data == 0 {
		return 0
	}
	dst := unsafe.Slice(*(**byte)(unsafe.Pointer(&data)), len(pix))
	for i := 0; i < len(pix); i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = pix[i+2], pix[i+1], pix[i], pix[i+3]
	}
	img := autorelease(send(send(class("NSImage"), "alloc"), "init"))
	send(img, "addRepresentation:", uintptr(rep))
	msgSetSize(img, sel("setSize:"), NSSize{Width: float64(w), Height: float64(h)})
	return img
}

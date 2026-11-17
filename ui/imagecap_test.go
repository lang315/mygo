package ui

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"runtime"
	"testing"
)

// grayPNG returns a black w×h PNG of one bit a pixel, a few kilobytes
// however large it is.
func grayPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var idat bytes.Buffer
	z, _ := zlib.NewWriterLevel(&idat, zlib.BestSpeed)
	row := make([]byte, 1+(w+7)/8) // filter byte, then the bits
	for range h {
		if _, err := z.Write(row); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return pngOf(w, h, idat.Bytes())
}

// pngOf returns a PNG of a gray header of one bit and idat.
func pngOf(w, h int, idat []byte) []byte {
	var b bytes.Buffer
	b.Write(pngSignature)
	hdr := make([]byte, 13)
	binary.BigEndian.PutUint32(hdr, uint32(w))
	binary.BigEndian.PutUint32(hdr[4:], uint32(h))
	hdr[8] = 1 // bit depth; color type 0, gray
	writeChunk(&b, "IHDR", hdr)
	writeChunk(&b, "IDAT", idat)
	writeChunk(&b, "IEND", nil)
	return b.Bytes()
}

// allocated returns the bytes f allocates.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func TestAPNGRefusesHugeHeader(t *testing.T) {
	data := pngOf(60000, 60000, []byte("x"))
	var err error
	n := allocated(func() { _, err = NewAPNGDecoder(data) })
	if err == nil {
		t.Error("a 60000×60000 PNG decoded")
	}
	if n > 64<<20 {
		t.Errorf("refusing a 60000×60000 PNG allocated %d MB", n>>20)
	}

	// At the cap, and a pixel past it (5 × 13421773 = 1<<26 + 1).
	if _, err := NewAPNGDecoder(pngOf(8192, 8192, []byte("x"))); err != nil {
		t.Errorf("a PNG of MaxImagePixels: %v", err)
	}
	if _, err := NewAPNGDecoder(pngOf(5, 13421773, []byte("x"))); err == nil {
		t.Error("a PNG of MaxImagePixels+1 decoded")
	}

	dec, err := NewAPNGDecoder(grayPNG(t, 4096, 4096))
	if err != nil {
		t.Fatal(err)
	}
	f, err := dec.Next()
	if err != nil {
		t.Fatal(err)
	}
	if b := f.Image.Bounds(); b.Dx() != 4096 || b.Dy() != 4096 {
		t.Errorf("a 4096×4096 PNG decoded to %v", b)
	}
}

func TestDecodeBitmapRefusesHugeImage(t *testing.T) {
	data := grayPNG(t, 20000, 20000)
	var err error
	n := allocated(func() { _, err = DecodeBitmap(data) })
	if err == nil {
		t.Error("a 20000×20000 PNG decoded")
	}
	if n > 64<<20 {
		t.Errorf("refusing a 20000×20000 PNG allocated %d MB", n>>20)
	}

	if _, err := DecodeBitmap(grayPNG(t, 8192, 8192)); err != nil {
		t.Errorf("a PNG of MaxImagePixels: %v", err)
	}
	if _, err := DecodeBitmap(grayPNG(t, 5, 13421773)); err == nil {
		t.Error("a PNG of MaxImagePixels+1 decoded")
	}
	// A phone's 48-megapixel photo.
	if bm, err := DecodeBitmap(grayPNG(t, 8064, 6048)); err != nil {
		t.Errorf("an 8064×6048 PNG: %v", err)
	} else if w, h := bm.Size(); w != 8064 || h != 6048 {
		t.Errorf("an 8064×6048 PNG decoded to %d×%d", w, h)
	}

	bm, err := DecodeBitmap(grayPNG(t, 4096, 4096))
	if err != nil {
		t.Fatal(err)
	}
	if w, h := bm.Size(); w != 4096 || h != 4096 {
		t.Errorf("a 4096×4096 PNG decoded to %d×%d", w, h)
	}
}

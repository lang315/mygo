package widget

import (
	"container/list"
	"crypto/sha256"
	"sync"

	"github.com/egoist/mygo/ui"
)

const (
	imageCacheSize = 256
	// imageCacheMax is the largest input cached: icons repeat across rows,
	// photos rarely do, and 256 decoded photos would take gigabytes.
	imageCacheMax = 64 << 10
)

// imageCache keeps the small bitmaps decoded last, by the SHA-256 of their
// bytes, as gota's pixbuf cache does: the same icon in many rows decodes
// once. Bad bytes are kept too (as nil), so they fail once.
var imageCache = struct {
	sync.Mutex
	order *list.List // of cachedImage, most recent first
	m     map[[32]byte]*list.Element
}{order: list.New(), m: map[[32]byte]*list.Element{}}

type cachedImage struct {
	key [32]byte
	bmp *ui.Bitmap
}

// decodeImage returns the bitmap of a PNG, JPEG, GIF, WebP or BMP, nil for
// bytes that are none.
func decodeImage(data []byte) *ui.Bitmap {
	if len(data) == 0 {
		return nil
	}
	if len(data) > imageCacheMax {
		bmp, _ := ui.DecodeBitmap(data)
		return bmp
	}
	key := sha256.Sum256(data)
	imageCache.Lock()
	if e, ok := imageCache.m[key]; ok {
		imageCache.order.MoveToFront(e)
		imageCache.Unlock()
		return e.Value.(cachedImage).bmp
	}
	imageCache.Unlock()
	// Decode outside the lock, so a slow decode holds up no other image;
	// two goroutines decoding the same bytes at once just both do it.
	bmp, _ := ui.DecodeBitmap(data)
	imageCache.Lock()
	defer imageCache.Unlock()
	if e, ok := imageCache.m[key]; ok {
		return e.Value.(cachedImage).bmp
	}
	imageCache.m[key] = imageCache.order.PushFront(cachedImage{key, bmp})
	if imageCache.order.Len() > imageCacheSize {
		last := imageCache.order.Back()
		imageCache.order.Remove(last)
		delete(imageCache.m, last.Value.(cachedImage).key)
	}
	return bmp
}

// ImageWidget shows an image.
type ImageWidget struct {
	common[ImageWidget]
	bmp  *ui.Bitmap
	w, h float32
}

// Image shows the image in data (PNG, JPEG, GIF, WebP or BMP) w by h DIPs;
// 0 keeps the image's own size. Bytes that are no image leave the room
// empty.
func Image(data []byte, w, h float32) *ImageWidget {
	i := &ImageWidget{bmp: decodeImage(data), w: w, h: h}
	i.init(i)
	return i
}

// SetImage shows another image.
func (i *ImageWidget) SetImage(data []byte) *ImageWidget {
	bmp := decodeImage(data) // outside mu: decoding takes a while
	change(func() { i.bmp = bmp })
	return i
}

func (i *ImageWidget) build(c *ui.Context) ui.Element {
	var e ui.Element
	if i.bmp == nil {
		e = ui.Box(c)
	} else {
		e = ui.Image(c, i.bmp)
	}
	if i.w > 0 {
		e.Width(i.w)
	}
	if i.h > 0 {
		e.Height(i.h)
	}
	return e
}

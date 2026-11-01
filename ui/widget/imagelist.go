package widget

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// maxImageSide is the longest side, in pixels, an image of a list may have: a
// larger one is refused before it is decoded.
const maxImageSide = 4096

// ImageListWidget keeps images by key, as WinForms' ImageList does, to share
// between the controls that show images: a ListBox's and a TreeView's rows
// (SetImageList), a menu item's icon (MenuItem.IconPNG with PNG), a text box's
// icon. It is not a control: make one, add the images, and give the controls
// what it returns.
//
// An image comes in one size or several (the same picture at 16, 32 and 48
// pixels, say). Size is the side the controls draw it at, in DIPs, as
// WinForms' ImageSize; the bitmap they get is the largest, which ui draws
// smaller smoothly, or ImageAt picks the smallest one that is crisp at a
// display's scale.
//
// Every method, and the function ImagesBy returns, is safe from any goroutine.
// The list has a lock of its own, taken after the package's and never before
// it: controls call ImagesBy's function with the package's lock held, so
// nothing here takes the package's lock while holding this one.
type ImageListWidget struct {
	size int

	m    sync.RWMutex // keys and imgs
	keys []string
	imgs map[string]*listImage

	// gen changes whenever an image is set, removed or cleared: the views that
	// show the list's images fold it in the versions of their rows, so that a
	// row kept shows the image now.
	gen atomic.Uint64
}

// generation is gen, 0 for no list.
func (l *ImageListWidget) generation() uint64 {
	if l == nil {
		return 0
	}
	return l.gen.Load()
}

// listImage is the sizes given for a key, by width, smallest first.
type listImage struct {
	bitmaps []*ui.Bitmap
	widths  []int
	raw     [][]byte // the data each was added from, nil for an image.Image
	png     [][]byte // the PNG data of each, made when asked for
}

// ImageList makes an empty list of images shown size DIPs square.
func ImageList(size int) *ImageListWidget {
	return &ImageListWidget{size: size, imgs: map[string]*listImage{}}
}

// Size returns the side, in DIPs, the images are shown at.
func (l *ImageListWidget) Size() int { return l.size }

// Add decodes the images (PNG or GIF, or JPEG, WebP or BMP once the program
// imports their decoders, see ui.DecodeBitmap; the same picture at one size or
// several) and keeps them under key, in place of those it had. It
// fails, and changes nothing, if there is none, one does not decode, one is
// empty or longer than 4096 pixels on a side, or their proportions differ.
func (l *ImageListWidget) Add(key string, data ...[]byte) error {
	if len(data) == 0 {
		return errors.New("widget: ImageList.Add: no image")
	}
	e := &listImage{}
	for i, d := range data {
		cfg, err := ui.DecodeBitmapConfig(d)
		if err != nil {
			return fmt.Errorf("widget: ImageList.Add %q: image %d: %w", key, i, err)
		}
		if err := checkSide(cfg.Width, cfg.Height); err != nil {
			return fmt.Errorf("widget: ImageList.Add %q: image %d: %w", key, i, err)
		}
		bmp, err := ui.DecodeBitmap(d)
		if err != nil {
			return fmt.Errorf("widget: ImageList.Add %q: image %d: %w", key, i, err)
		}
		if err := e.add(bmp, bytes.Clone(d)); err != nil {
			return fmt.Errorf("widget: ImageList.Add %q: image %d: %w", key, i, err)
		}
	}
	l.set(key, e)
	return nil
}

// AddImage keeps the images (the same picture at one size or several) under
// key, in place of those it had; it refuses what Add refuses, and a nil image.
func (l *ImageListWidget) AddImage(key string, imgs ...image.Image) error {
	if len(imgs) == 0 {
		return errors.New("widget: ImageList.AddImage: no image")
	}
	e := &listImage{}
	for i, img := range imgs {
		if img == nil {
			return fmt.Errorf("widget: ImageList.AddImage %q: image %d is nil", key, i)
		}
		b := img.Bounds()
		if err := checkSide(b.Dx(), b.Dy()); err != nil {
			return fmt.Errorf("widget: ImageList.AddImage %q: image %d: %w", key, i, err)
		}
		if err := e.add(ui.NewBitmap(img), nil); err != nil {
			return fmt.Errorf("widget: ImageList.AddImage %q: image %d: %w", key, i, err)
		}
	}
	l.set(key, e)
	return nil
}

func checkSide(w, h int) error {
	if w <= 0 || h <= 0 {
		return errors.New("the image is empty")
	}
	if w > maxImageSide || h > maxImageSide {
		return fmt.Errorf("the image is %dx%d, no side may pass %d pixels", w, h, maxImageSide)
	}
	return nil
}

// add keeps bmp among the sizes, by width; a size of other proportions than
// the first is refused.
func (e *listImage) add(bmp *ui.Bitmap, raw []byte) error {
	w, h := bmp.Size()
	if len(e.bitmaps) > 0 {
		w0, h0 := e.bitmaps[0].Size()
		if d := w*h0 - w0*h; d > w0*h/50 || -d > w0*h/50 { // 2%
			return fmt.Errorf("a %dx%d image does not have the proportions of the %dx%d one", w, h, w0, h0)
		}
	}
	at, _ := slices.BinarySearch(e.widths, w)
	e.widths = slices.Insert(e.widths, at, w)
	e.bitmaps = slices.Insert(e.bitmaps, at, bmp)
	e.raw = slices.Insert(e.raw, at, raw)
	e.png = slices.Insert(e.png, at, nil)
	return nil
}

func (l *ImageListWidget) set(key string, e *listImage) {
	l.m.Lock()
	if _, ok := l.imgs[key]; !ok {
		l.keys = append(l.keys, key)
	}
	l.imgs[key] = e
	l.gen.Add(1)
	l.m.Unlock()
	l.changed()
}

// changed redraws the windows, which may show the list's images. Called with
// l.m not held.
func (l *ImageListWidget) changed() { Change(func() {}) }

// Has reports whether the list has an image under key.
func (l *ImageListWidget) Has(key string) bool {
	l.m.RLock()
	defer l.m.RUnlock()
	_, ok := l.imgs[key]
	return ok
}

// Len returns how many keys the list has.
func (l *ImageListWidget) Len() int {
	l.m.RLock()
	defer l.m.RUnlock()
	return len(l.keys)
}

// Keys returns the keys, in the order they were first added.
func (l *ImageListWidget) Keys() []string {
	l.m.RLock()
	defer l.m.RUnlock()
	return slices.Clone(l.keys)
}

// Remove takes the images of the keys out; a key the list lacks is ignored.
func (l *ImageListWidget) Remove(keys ...string) {
	l.m.Lock()
	for _, k := range keys {
		delete(l.imgs, k)
	}
	l.keys = slices.DeleteFunc(l.keys, func(k string) bool { _, ok := l.imgs[k]; return !ok })
	l.gen.Add(1)
	l.m.Unlock()
	l.changed()
}

// Clear takes every image out.
func (l *ImageListWidget) Clear() {
	l.m.Lock()
	clear(l.imgs)
	l.keys = nil
	l.gen.Add(1)
	l.m.Unlock()
	l.changed()
}

// Image returns the bitmap under key, the largest of its sizes, which ui
// shows smaller smoothly; nil for a key the list lacks.
func (l *ImageListWidget) Image(key string) *ui.Bitmap {
	l.m.RLock()
	defer l.m.RUnlock()
	if e := l.imgs[key]; e != nil {
		return e.bitmaps[len(e.bitmaps)-1]
	}
	return nil
}

// ImageAt returns the smallest size under key that has a pixel for each of
// the Size DIPs at the display's scale (2 for twice the pixels), else the
// largest; nil for a key the list lacks.
func (l *ImageListWidget) ImageAt(key string, scale float64) *ui.Bitmap {
	l.m.RLock()
	defer l.m.RUnlock()
	e := l.imgs[key]
	if e == nil {
		return nil
	}
	need := int(float64(l.size)*scale + 0.999)
	for i, w := range e.widths {
		if w >= need {
			return e.bitmaps[i]
		}
	}
	return e.bitmaps[len(e.bitmaps)-1]
}

// PNG returns the largest size under key as PNG data, for the fields that
// take PNG bytes (MenuItem.IconPNG): the data it was added from if that was a
// PNG, else the picture encoded once (a JPEG with an EXIF orientation comes
// upright). Do not change the bytes. It is nil for a key the list lacks.
func (l *ImageListWidget) PNG(key string) []byte {
	l.m.RLock()
	e := l.imgs[key]
	if e == nil {
		l.m.RUnlock()
		return nil
	}
	last := len(e.bitmaps) - 1
	cached, raw, bmp := e.png[last], e.raw[last], e.bitmaps[last]
	l.m.RUnlock()
	if cached != nil {
		return cached
	}
	data := raw
	if !isPNG(raw) {
		img := bmp.Image()
		if img == nil {
			return nil
		}
		var b bytes.Buffer // encoded with no lock held: a big image takes a while
		if err := png.Encode(&b, img); err != nil {
			return nil
		}
		data = b.Bytes()
	}
	l.m.Lock()
	if l.imgs[key] == e {
		e.png[last] = data
	}
	l.m.Unlock()
	return data
}

func isPNG(data []byte) bool {
	return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
}

// ImagesBy returns the function a control's SetImage takes: the image of the
// list under the key of each item, nil for an item whose key the list lacks.
// ListBox and TreeView draw it 16 DIPs square; SetImageList draws it at the
// list's Size.
func ImagesBy[T any](l *ImageListWidget, key func(*T) string) func(*T) *ui.Bitmap {
	return func(it *T) *ui.Bitmap {
		if k := key(it); k != "" {
			return l.Image(k)
		}
		return nil
	}
}

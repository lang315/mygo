package widget

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"slices"

	"github.com/egoist/mygo/ui"
)

// ImageListWidget keeps images by key, as WinForms' ImageList does, to share
// between the controls that show images: a ListBox's and a TreeView's rows
// (SetImage with ImagesBy), a menu item's icon (MenuItem.IconPNG with PNG), a
// text box's icon. It is not a control: make one, add the images, and give the
// controls what it returns.
//
// An image comes in one size or several (the same picture at 16, 32 and 48
// pixels, say). Size is the side the controls draw it at, in DIPs; the bitmap
// they get is the largest, which ui draws smaller smoothly, or ImageAt picks
// the smallest one that is crisp at a display's scale.
//
// Every method is safe from any goroutine.
type ImageListWidget struct {
	size int
	keys []string
	imgs map[string]*listImage
}

// listImage is the sizes given for a key, by width, smallest first.
type listImage struct {
	bitmaps []*ui.Bitmap
	png     [][]byte // the PNG data of each, made when asked for
	widths  []int
	srcs    []image.Image // the images given, for PNG data made later
}

// ImageList makes an empty list of images shown size DIPs square.
func ImageList(size int) *ImageListWidget {
	return &ImageListWidget{size: size, imgs: map[string]*listImage{}}
}

// Size returns the side, in DIPs, the images are shown at.
func (l *ImageListWidget) Size() int { return l.size }

// Add decodes the images (PNG, JPEG, GIF, WebP or BMP, the same picture at
// one size or several) and keeps them under key, in place of those it had. It
// fails, and changes nothing, if there is none or one does not decode.
func (l *ImageListWidget) Add(key string, data ...[]byte) error {
	if len(data) == 0 {
		return errors.New("widget: ImageList.Add: no image")
	}
	e := &listImage{}
	for i, d := range data {
		img, _, err := image.Decode(bytes.NewReader(d))
		if err != nil {
			return fmt.Errorf("widget: ImageList.Add %q: image %d: %w", key, i, err)
		}
		bmp, err := ui.DecodeBitmap(d)
		if err != nil {
			return fmt.Errorf("widget: ImageList.Add %q: image %d: %w", key, i, err)
		}
		e.add(bmp, img)
	}
	l.set(key, e)
	return nil
}

// AddImage keeps the images (the same picture at one size or several) under
// key, in place of those it had.
func (l *ImageListWidget) AddImage(key string, imgs ...image.Image) *ImageListWidget {
	if len(imgs) == 0 {
		return l
	}
	e := &listImage{}
	for _, img := range imgs {
		e.add(ui.NewBitmap(img), img)
	}
	l.set(key, e)
	return l
}

func (e *listImage) add(bmp *ui.Bitmap, img image.Image) {
	w, _ := bmp.Size()
	at, _ := slices.BinarySearch(e.widths, w)
	e.widths = slices.Insert(e.widths, at, w)
	e.bitmaps = slices.Insert(e.bitmaps, at, bmp)
	e.srcs = slices.Insert(e.srcs, at, img)
	e.png = slices.Insert(e.png, at, nil)
}

func (l *ImageListWidget) set(key string, e *listImage) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := l.imgs[key]; !ok {
		l.keys = append(l.keys, key)
	}
	l.imgs[key] = e
}

// Has reports whether the list has an image under key.
func (l *ImageListWidget) Has(key string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := l.imgs[key]
	return ok
}

// Len returns how many keys the list has.
func (l *ImageListWidget) Len() int {
	mu.Lock()
	defer mu.Unlock()
	return len(l.keys)
}

// Keys returns the keys, in the order they were first added.
func (l *ImageListWidget) Keys() []string {
	mu.Lock()
	defer mu.Unlock()
	return slices.Clone(l.keys)
}

// Remove takes the images of the keys out; a key the list lacks is ignored.
func (l *ImageListWidget) Remove(keys ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, k := range keys {
		delete(l.imgs, k)
	}
	l.keys = slices.DeleteFunc(l.keys, func(k string) bool { _, ok := l.imgs[k]; return !ok })
}

// Clear takes every image out.
func (l *ImageListWidget) Clear() {
	mu.Lock()
	defer mu.Unlock()
	clear(l.imgs)
	l.keys = nil
}

// Image returns the bitmap under key, the largest of its sizes, which ui
// shows smaller smoothly; nil for a key the list lacks.
func (l *ImageListWidget) Image(key string) *ui.Bitmap {
	mu.Lock()
	defer mu.Unlock()
	if e := l.imgs[key]; e != nil {
		return e.bitmaps[len(e.bitmaps)-1]
	}
	return nil
}

// ImageAt returns the smallest size under key that has a pixel for each of
// the Size DIPs at the display's scale (2 for twice the pixels), else the
// largest; nil for a key the list lacks.
func (l *ImageListWidget) ImageAt(key string, scale float64) *ui.Bitmap {
	mu.Lock()
	defer mu.Unlock()
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
// take PNG bytes (MenuItem.IconPNG); nil for a key the list lacks.
func (l *ImageListWidget) PNG(key string) []byte {
	mu.Lock()
	defer mu.Unlock()
	e := l.imgs[key]
	if e == nil {
		return nil
	}
	last := len(e.bitmaps) - 1
	if e.png[last] == nil {
		var b bytes.Buffer
		if err := png.Encode(&b, e.srcs[last]); err != nil {
			return nil
		}
		e.png[last] = b.Bytes()
	}
	return e.png[last]
}

// ImagesBy returns the function a control takes for its images (ListBox's and
// TreeView's SetImage): the image of the list under the key of each item, nil
// for an item whose key the list lacks.
func ImagesBy[T any](l *ImageListWidget, key func(*T) string) func(*T) *ui.Bitmap {
	return func(it *T) *ui.Bitmap {
		if k := key(it); k != "" {
			return l.imageLocked(k)
		}
		return nil
	}
}

// imageLocked is Image for the callbacks controls make with the package's
// lock held.
func (l *ImageListWidget) imageLocked(key string) *ui.Bitmap {
	if e := l.imgs[key]; e != nil {
		return e.bitmaps[len(e.bitmaps)-1]
	}
	return nil
}

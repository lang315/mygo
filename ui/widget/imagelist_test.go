package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// pngOf returns a PNG of side x side pixels of c.
func pngOf(t testing.TB, side int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func width(b *ui.Bitmap) int { w, _ := b.Size(); return w }

func TestImageListHoldsKeyedImagesInTheOrderAdded(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	il := ImageList(16)
	if err := il.Add("open", pngOf(t, 16, blue)); err != nil {
		t.Fatal(err)
	}
	if err := il.Add("save", pngOf(t, 16, blue)); err != nil {
		t.Fatal(err)
	}
	if il.Size() != 16 || il.Len() != 2 || !il.Has("open") || il.Has("close") || !slices.Equal(il.Keys(), []string{"open", "save"}) {
		t.Fatalf("size %d len %d keys %v", il.Size(), il.Len(), il.Keys())
	}
	if il.Image("close") != nil {
		t.Error("an unknown key has an image")
	}
	if err := il.Add("open", pngOf(t, 24, blue)); err != nil || width(il.Image("open")) != 24 || !slices.Equal(il.Keys(), []string{"open", "save"}) {
		t.Errorf("adding a key again replaces its image and keeps its place: %v %v", err, il.Keys())
	}
	il.Remove("open", "nothing")
	if il.Has("open") || il.Len() != 1 {
		t.Errorf("Remove: keys %v", il.Keys())
	}
	il.Clear()
	if il.Len() != 0 || il.Image("save") != nil {
		t.Error("Clear left images")
	}
}

func TestImageListAddRefusesWhatIsNotAnImage(t *testing.T) {
	il := ImageList(16)
	if err := il.Add("bad", []byte("not an image")); err == nil || il.Has("bad") {
		t.Errorf("a bad image was taken: %v", err)
	}
	if err := il.Add("none"); err == nil || il.Has("none") {
		t.Errorf("no image was taken: %v", err)
	}
}

// Several sizes of one image: the largest for Image, which ui draws smaller
// smoothly, and the smallest that is crisp for a display's scale.
func TestImageListPicksTheSizeForTheDisplay(t *testing.T) {
	c := color.RGBA{255, 0, 0, 255}
	il := ImageList(16)
	if err := il.Add("a", pngOf(t, 32, c), pngOf(t, 16, c), pngOf(t, 48, c)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		scale float64
		want  int
	}{{1, 16}, {1.25, 32}, {1.5, 32}, {2, 32}, {2.5, 48}, {3, 48}, {4, 48}} {
		if got := width(il.ImageAt("a", tc.scale)); got != tc.want {
			t.Errorf("scale %v: %d pixels, want %d", tc.scale, got, tc.want)
		}
	}
	if width(il.Image("a")) != 48 {
		t.Errorf("Image is %d pixels, want the largest, 48", width(il.Image("a")))
	}
	if il.ImageAt("zzz", 1) != nil {
		t.Error("an unknown key has an image at a scale")
	}
}

func TestImageListPNGForMenuIcons(t *testing.T) {
	c := color.RGBA{0, 255, 0, 255}
	il := ImageList(16)
	il.AddImage("g", image.NewRGBA(image.Rect(0, 0, 16, 16)), image.NewRGBA(image.Rect(0, 0, 32, 32)))
	if err := il.Add("p", pngOf(t, 16, c)); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]int{"g": 32, "p": 16} {
		data := il.PNG(k)
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width != want {
			t.Errorf("PNG(%q): %v, width %d, want %d", k, err, cfg.Width, want)
		}
	}
	if il.PNG("zzz") != nil {
		t.Error("an unknown key has PNG data")
	}
}

type pic struct{ id, name, icon string }

// A ListBox's and a TreeView's images come from the list by a key of the item.
func TestImagesByDrivesListBoxAndTreeViewImages(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	il := ImageList(16)
	if err := il.Add("doc", pngOf(t, 16, blue)); err != nil {
		t.Fatal(err)
	}
	items := []pic{{"1", "First", "doc"}, {"2", "Second", "nothing"}}
	lb := ListBox(func(p *pic) string { return p.id }, func(p *pic) string { return p.name }).
		SetItems(items).SetImage(ImagesBy(il, func(p *pic) string { return p.icon }))
	tt := tester(t, lb)
	first, second := find(t, tt, "First"), find(t, tt, "Second")
	if b, n := bluePixels2(tt, first), bluePixels2(tt, second); b == 0 || n != 0 {
		t.Errorf("blue pixels in the row with the image: %d, in the row whose key is unknown: %d", b, n)
	}
	if ImagesBy(il, func(p *pic) string { return p.icon })(&items[1]) != nil {
		t.Error("an unknown key gave an image")
	}
	tv := TreeView(func(p *pic) string { return p.id }, func(p *pic) string { return p.name }).
		SetRoots(items[:1]).SetImage(ImagesBy(il, func(p *pic) string { return p.icon }))
	tt = tester(t, tv)
	if bluePixels2(tt, find(t, tt, "First")) == 0 {
		t.Error("the tree node shows no image")
	}
}

func bluePixels2(tt *ui.Tester, r ui.Rect) int {
	img := tt.Image()
	n := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X) - 20; x < int(r.X+r.W); x++ {
			if x < 0 {
				continue
			}
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c.B > 200 && c.R < 60 && c.G < 60 {
				n++
			}
		}
	}
	return n
}

func TestImageListIsSafeFromManyGoroutines(t *testing.T) {
	c := color.RGBA{255, 255, 0, 255}
	data := pngOf(t, 16, c)
	il := ImageList(16)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k := string(rune('a' + i))
			for range 20 {
				_ = il.Add(k, data)
				il.Image(k)
				il.Keys()
				il.Remove(k)
			}
		}()
	}
	wg.Wait()
}

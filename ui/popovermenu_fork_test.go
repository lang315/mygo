package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// popoverStyleApp is a view with a drawn menu holding the items the
// fork's menus show with a look of their own: one that destroys, one with
// a PNG icon, one with an SVG icon and a plain one.
type popoverStyleApp struct {
	theme       *Theme
	icon        []byte
	notDestroys bool // Remove is a plain item
}

func (a *popoverStyleApp) view(c *context) {
	a.theme = c.Theme()
	svg, err := ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect width="24" height="24" fill="currentColor"/></svg>`))
	if err != nil {
		panic(err)
	}
	coreColumn(c).Fill().Padding(20).Children(func() {
		row := coreRow(c).Padding(8).Focusable().Label("Report").Children(func() {
			coreText(c, "Report.pdf")
		})
		row.PopoverMenu(func(m *Menu) {
			m.Item("Keep")
			m.Item("Remove").Destructive(!a.notDestroys)
			m.Item("Rename").IconPNG(a.icon)
			m.Item("Share").Icon(svg)
		})
	})
}

// solidPNG returns a 32×32 PNG of one color.
func solidPNG(t *testing.T, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pixelsOf counts the pixels of r in img that are within a few levels of c.
func pixelsOf(img *image.RGBA, r Rect, c Color) (n int) {
	near := func(a, b uint8) bool { return int(a)-int(b) < 6 && int(b)-int(a) < 6 }
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := img.RGBAAt(x, y)
			if near(p.R, c.R) && near(p.G, c.G) && near(p.B, c.B) {
				n++
			}
		}
	}
	return n
}

// inkOf counts the pixels of r in img that differ from its top left one.
func inkOf(img *image.RGBA, r Rect) (n int) {
	bg := img.RGBAAt(int(r.X)+1, int(r.Y)+1)
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X) + 30; x < int(r.X+r.W); x++ { // after the icon's room
			if img.RGBAAt(x, y) != bg {
				n++
			}
		}
	}
	return n
}

func openPopoverStyleApp(t *testing.T, notDestroys bool) (*Tester, *popoverStyleApp) {
	a := &popoverStyleApp{icon: solidPNG(t, color.NRGBA{255, 0, 255, 255}), notDestroys: notDestroys}
	tt := coreNewTester(a.view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	if !tt.HasText("Remove") {
		t.Fatalf("no menu after a right-click: %q", tt.Texts())
	}
	return tt, a
}

// A drawn menu shows an item that destroys in the danger color and
// semibold, as the menus the backends draw do, and the others as they are.
func TestPopoverMenuDrawsDestructiveItems(t *testing.T) {
	tt, a := openPopoverStyleApp(t, false)
	img := tt.Image()
	remove, _ := tt.Find("Remove")
	keep, _ := tt.Find("Keep")
	if n := pixelsOf(img, remove, a.theme.Danger); n == 0 {
		t.Errorf("Remove has no pixel in the danger color %+v", a.theme.Danger)
	}
	if n := pixelsOf(img, keep, a.theme.Danger); n != 0 {
		t.Errorf("Keep has %d pixels in the danger color", n)
	}
	// Semibold text has more ink than regular text of the same letters.
	pt, _ := openPopoverStyleApp(t, true)
	plain, _ := pt.Find("Remove")
	inkPlain := inkOf(pt.Image(), plain)
	if inkBold := inkOf(img, remove); inkPlain == 0 || float64(inkBold) < 1.1*float64(inkPlain) {
		t.Errorf("Remove has %d inked pixels, a plain item %d: want a semibold item's, more", inkBold, inkPlain)
	}
}

// A drawn menu shows an item's icon before its label: a PNG as it is, an
// SVG in the color of the text.
func TestPopoverMenuDrawsIcons(t *testing.T) {
	tt, a := openPopoverStyleApp(t, false)
	img := tt.Image()
	rename, _ := tt.Find("Rename")
	before := Rect{X: rename.X, Y: rename.Y, W: 30, H: rename.H}
	if n := pixelsOf(img, before, Color{R: 255, G: 0, B: 255, A: 255}); n < 100 {
		t.Errorf("%d pixels of the PNG icon before Rename, want its 16×16 DIPs", n)
	}
	share, _ := tt.Find("Share")
	before = Rect{X: share.X, Y: share.Y, W: 30, H: share.H}
	if n := pixelsOf(img, before, a.theme.Text); n < 100 {
		t.Errorf("%d pixels of the SVG icon in the text color %+v before Share, want its 16×16 DIPs", n, a.theme.Text)
	}
	keep, _ := tt.Find("Keep")
	before = Rect{X: keep.X, Y: keep.Y, W: 30, H: keep.H}
	if n := pixelsOf(img, before, Color{R: 255, G: 0, B: 255, A: 255}); n != 0 {
		t.Errorf("%d pixels of an icon before Keep, which has none", n)
	}
}

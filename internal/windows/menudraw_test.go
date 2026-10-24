//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// The highlights of menu items cover their pixels inside fully, those past
// the rounded corners not at all, and those the corners cross partly.
func TestRoundedCoverage(t *testing.T) {
	r := &rect{0, 0, 40, 20}
	for _, c := range []struct {
		x, y     float64
		min, max float64
	}{
		{20.5, 10.5, 1, 1}, // the middle
		{0.5, 10.5, 1, 1},  // the left edge, between the corners
		{20.5, 0.5, 1, 1},  // the top edge
		{0.5, 0.5, 0, 0},   // past the top-left corner's curve
		{39.5, 19.5, 0, 0}, // past the bottom-right one's
		{1.5, 1.5, 0.01, 0.99},
	} {
		if got := roundedCoverage(c.x, c.y, r, 4); got < c.min || got > c.max {
			t.Errorf("coverage at (%v, %v) = %v, want %v to %v", c.x, c.y, got, c.min, c.max)
		}
	}
}

func TestMenuColors(t *testing.T) {
	if got := dibColor(0x332211); got != 0x112233 {
		t.Errorf("dibColor(0x332211) = %#x, want 0x112233", got)
	}
	if got := blend(0xFFFFFF, 0x000000, 0.5); got != 0x808080 {
		t.Errorf("blend halfway = %#x, want 0x808080", got)
	}
	if got := blend(0x102030, 0xA0B0C0, 0); got != 0xA0B0C0 {
		t.Errorf("blend with none = %#x, want the background", got)
	}
}

// redPNG is a PNG of a solid red square.
func redPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// drawnMenu builds a popup menu of items, MyGo drawing them, and returns the
// looks of its items by label.
func drawnMenu(t *testing.T, b *Backend, dpi int, items ...*platform.MenuItem) map[string]*menuLook {
	t.Helper()
	owner := b.menus.newOwner()
	h := b.buildMenu(&platform.Menu{Items: items}, false, owner, nil, dpi)
	t.Cleanup(func() {
		b.menus.drop(owner)
		procDestroyMenu.Call(h)
	})
	looks := map[string]*menuLook{}
	for _, l := range b.menus.looks {
		if l.owner == owner {
			looks[l.label] = l
		}
	}
	return looks
}

// drawnItem measures an item and draws it in a bitmap, and returns its
// width and how much redder than green its pixels are, at most, in the
// place of an icon: the test icon is solid red, the menu's colors are not,
// and the antialiasing of text only a little.
func drawnItem(b *Backend, l *menuLook, dpi int, state uint32) (width int32, red int) {
	var mis measureItemStruct
	b.measureItem(0, l, &mis)
	width, height := int32(mis.ItemWidth), int32(mis.ItemHeight)
	img := newBitmap(0, width, height)
	defer img.free()
	d := drawItemStruct{CtlType: odtMenu, ItemState: state, DC: img.dc, Item: rect{0, 0, width, height}}
	b.drawItem(0, l, &d)
	left := toPx(menuInset+menuPadding, dpi)
	for y := range height {
		for x := left; x < left+toPx(16, dpi); x++ {
			p := img.px[y*width+x]
			red = max(red, int(p>>16&0xFF)-int(p>>8&0xFF))
		}
	}
	return width, red
}

// A popup item MyGo draws shows its icon in a column of its own, which the
// menu's other items leave room for, as Windows does for the bitmaps of the
// items it draws.
func TestDrawnItemShowsIcon(t *testing.T) {
	if !fluentMenus() {
		t.Skip("Windows draws the items' bitmaps before Windows 11")
	}
	b := &Backend{menus: newMenuTable()}
	dpi := dpiOf(0)
	withIcon := drawnMenu(t, b, dpi,
		&platform.MenuItem{ID: 1, Label: "Icon", Enabled: true, Visible: true, Icon: redPNG(t)},
		&platform.MenuItem{ID: 2, Label: "Plain", Enabled: true, Visible: true})
	without := drawnMenu(t, b, dpi, &platform.MenuItem{ID: 3, Label: "Plain", Enabled: true, Visible: true})

	_, red := drawnItem(b, withIcon["Icon"], dpi, 0)
	if red < 200 {
		t.Errorf("the item with an icon is %d redder than green in its icon's place, want 200 or more", red)
	}
	_, dimmed := drawnItem(b, withIcon["Icon"], dpi, odsGrayed)
	if dimmed <= 0 || dimmed >= red {
		t.Errorf("the disabled item's icon is %d redder than green, want it shown and fainter than the enabled one's %d", dimmed, red)
	}
	wide, plainRed := drawnItem(b, withIcon["Plain"], dpi, 0)
	narrow, _ := drawnItem(b, without["Plain"], dpi, 0)
	if plainRed > 0 {
		t.Errorf("the item without an icon has red in its icon's place (%d)", plainRed)
	}
	if want := toPx(16, dpi); wide-narrow < want {
		t.Errorf("an item is %d pixels wider in a menu with icons, want at least %d", wide-narrow, want)
	}
}

// Windows measures an item with a bitmap (MIIM_BITMAP) itself, to the
// bitmap's height, without asking the window: an item MyGo draws has none,
// so that it is measured as the others.
func TestDrawnItemHasNoMenuBitmap(t *testing.T) {
	if !fluentMenus() {
		t.Skip("Windows draws the items' bitmaps before Windows 11")
	}
	b := &Backend{menus: newMenuTable()}
	owner := b.menus.newOwner()
	items := []*platform.MenuItem{
		{ID: 1, Label: "Icon", Enabled: true, Visible: true, Icon: redPNG(t)},
		{ID: 2, Label: "More", Enabled: true, Visible: true, Icon: redPNG(t),
			Submenu: &platform.Menu{Items: []*platform.MenuItem{{ID: 3, Label: "Inner", Enabled: true, Visible: true}}}},
	}
	h := b.buildMenu(&platform.Menu{Items: items}, false, owner, nil, dpiOf(0))
	defer procDestroyMenu.Call(h)
	defer b.menus.drop(owner)
	for i, it := range items {
		mi := menuItemInfo{Mask: miimBitmap}
		mi.Size = uint32(unsafe.Sizeof(mi))
		procGetMenuItemInfoW.Call(h, uintptr(i), 1, uintptr(unsafe.Pointer(&mi)))
		if mi.BmpItem != 0 {
			t.Errorf("item %q has a menu bitmap, which Windows measures it by", it.Label)
		}
	}
}

// A destructive item's text is the danger color, readable on the highlight
// and greyed out when disabled; the system's colors win in high contrast,
// which has no danger color. Its label is the semibold font where there is
// one.
func TestDestructiveMenuItemLook(t *testing.T) {
	light := menuPalette{back: 0xF9F9F9, plate: 0xF0F0F0, text: 0x1A1A1A, plateText: 0x1A1A1A, disabled: 0x787878, danger: 0x1C2BC4}
	dark := menuPalette{back: 0x2B2B2B, plate: 0x383838, text: 0xFFFFFF, plateText: 0xFFFFFF, disabled: 0x787878, danger: 0xA499FF}
	highContrast := menuPalette{back: 0, plate: 0xFF0000, text: 0xFFFFFF, plateText: 0x000000, disabled: 0x808080}
	for _, c := range []struct {
		name                            string
		p                               menuPalette
		selected, disabled, destructive bool
		want                            uint32
	}{
		{"plain", light, false, false, false, 0x1A1A1A},
		{"plain highlighted", light, true, false, false, 0x1A1A1A},
		{"danger", light, false, false, true, 0x1C2BC4},
		{"danger highlighted stays danger", light, true, false, true, 0x1C2BC4},
		{"danger in the dark", dark, false, false, true, 0xA499FF},
		{"danger disabled is grey", light, false, true, true, 0x787878},
		{"danger disabled and highlighted is grey", light, true, true, true, 0x787878},
		{"plain disabled", light, false, true, false, 0x787878},
		{"high contrast keeps the system's text", highContrast, false, false, true, 0xFFFFFF},
		{"high contrast keeps the highlight's text", highContrast, true, false, true, 0x000000},
	} {
		if got := c.p.textColor(c.selected, c.disabled, c.destructive); got != c.want {
			t.Errorf("%s: color %#x, want %#x", c.name, got, c.want)
		}
	}
	f := menuFonts{text: 1, bold: 2}
	if got := (&menuLook{destructive: true}).labelFont(f); got != 2 {
		t.Errorf("a destructive label's font is %d, want the semibold one", got)
	}
	if got := (&menuLook{}).labelFont(f); got != 1 {
		t.Errorf("a plain label's font is %d, want the text font", got)
	}
	if got := (&menuLook{destructive: true}).labelFont(menuFonts{text: 1}); got != 1 {
		t.Errorf("without a semibold font a destructive label's font is %d, want the text font", got)
	}
}

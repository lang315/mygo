//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func testPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for i := range 32 {
		img.Set(i, i, color.RGBA{R: 255, A: 255})
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestMenuBitmap(t *testing.T) {
	if menuBitmap([]byte("not a png"), 16) != 0 {
		t.Error("a bad PNG made a bitmap")
	}
	bmp := menuBitmap(testPNG(t), 16)
	if bmp == 0 {
		t.Fatal("no bitmap")
	}
	procDeleteObject.Call(bmp)
}

func TestPopupMenuFreesBitmaps(t *testing.T) {
	b := &Backend{menus: newMenuTable()}
	m := &platform.Menu{Items: []*platform.MenuItem{{ID: 1, Label: "Icon", Enabled: true, Visible: true, Icon: testPNG(t)}}}
	owner := b.menus.newOwner()
	h := b.buildMenu(m, false, owner, nil, 96)
	if len(b.menus.bitmaps[owner]) != 1 {
		t.Fatalf("bitmaps %v", b.menus.bitmaps[owner])
	}
	b.menus.drop(owner)
	procDestroyMenu.Call(h)
	if len(b.menus.bitmaps) != 0 {
		t.Errorf("bitmaps left after drop: %v", b.menus.bitmaps)
	}
}

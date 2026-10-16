package ui

import (
	"bytes"
	"image/png"
	"testing"
)

const (
	checkIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 6 9 17l-5-5"/></svg>`
	logoIconSVG  = `<svg viewBox="0 0 30 10"><rect width="10" height="10" fill="#f00"/><rect x="20" width="10" height="10" fill="currentColor"/></svg>`
)

func TestMenuIconPNG(t *testing.T) {
	for _, tc := range []struct {
		src      string
		template bool
	}{{checkIconSVG, true}, {logoIconSVG, false}} {
		b, template := MenuIconPNG(MustParseSVG([]byte(tc.src)), Hex("#18181b"))
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if s := img.Bounds().Size(); s.X != 32 || s.Y != 32 {
			t.Errorf("size %v, want 32×32", s)
		}
		if template != tc.template {
			t.Errorf("template %v, want %v", template, tc.template)
		}
		opaque := 0
		for y := range 32 {
			for x := range 32 {
				if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
					opaque++
				}
			}
		}
		if opaque == 0 {
			t.Error("the icon drew nothing")
		}
	}
}

func TestMenuItemIcons(t *testing.T) {
	check := MustParseSVG([]byte(checkIconSVG))
	tt := coreNewTester(func(c *context) {
		coreText(c, "here").ContextMenu(func(m *Menu) {
			m.Item("Svg").Icon(check)
			m.Item("Png").IconPNG([]byte("png"))
			m.Item("Both").Icon(check).IconPNG([]byte("png"))
			m.Item("None")
			m.Item("NilPNG").Icon(check).IconPNG(nil)
		})
	}, 300, 200)
	if err := tt.RightClick("here"); err != nil {
		t.Fatal(err)
	}
	items := tt.h.menu.Items
	if len(items) != 5 {
		t.Fatalf("menu %q", tt.Menu())
	}
	if items[0].Icon == nil || !items[0].Template {
		t.Errorf("Svg: icon %d bytes, template %v", len(items[0].Icon), items[0].Template)
	}
	if string(items[1].Icon) != "png" || items[1].Template {
		t.Errorf("Png: icon %q, template %v", items[1].Icon, items[1].Template)
	}
	if string(items[2].Icon) != "png" || items[2].Template {
		t.Errorf("Both: the PNG must win, icon %q, template %v", items[2].Icon, items[2].Template)
	}
	if items[3].Icon != nil {
		t.Errorf("None has an icon")
	}
	if items[4].Icon == nil || !items[4].Template {
		t.Errorf("NilPNG: IconPNG(nil) cleared the SVG icon, icon %d bytes, template %v", len(items[4].Icon), items[4].Template)
	}
}

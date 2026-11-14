package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

// TestShots writes the studio's pages to PNG files in the directory named by
// MYGO_SKIN_SHOTS, for a look at the skins: each skin, light and dark, each
// tab, plus a hovered menu bar item and a selected grid row. It is skipped
// without the variable, so normal runs write nothing.
func TestShots(t *testing.T) {
	dir := os.Getenv("MYGO_SKIN_SHOTS")
	if dir == "" {
		t.Skip("MYGO_SKIN_SHOTS names the directory the PNG files go to")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	reset(t)

	write := func(tt *widget.Tester, name string) {
		t.Helper()
		tt.Frame()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	slug := func(s string) string { return strings.ReplaceAll(strings.ToLower(s), " ", "-") }

	skins := []struct {
		name string
		s    *skin.Skin
	}{
		{"none", nil},
		{"calm-blue", skin.Calm},
		{"calm-cham", skin.Calm.WithPalette(skin.Cham)},
	}
	// MYGO_SKIN_SCALE is the scale of the shots, 1 when unset.
	scale := float32(1)
	if v := os.Getenv("MYGO_SKIN_SCALE"); v != "" {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil || f <= 0 {
			t.Fatalf("MYGO_SKIN_SCALE=%q is not a positive number", v)
		}
		scale = float32(f)
	}
	// open builds the studio under a skin and an appearance, 1280 by 800 DIPs,
	// with the header pickers set to the skin and palette it shows.
	open := func(s *skin.Skin, dark bool) *widget.Tester {
		skin.Use(nil)
		tt := widget.TestForm(t, newStudio())
		tt.SetSize(1280, 800)
		tt.SetScale(scale)
		wantSkin, wantPalette := "none", "blue"
		if s != nil {
			wantSkin = "calm"
			for value, p := range palettes {
				if p == s.Palette() {
					wantPalette = value
				}
			}
			pick(tt, "Skin", "Calm")
			pick(tt, "Palette", s.Palette().Name())
		}
		if got := tt.Find(widget.ByLabel("Skin")).Value(); got != wantSkin {
			t.Errorf("the Skin picker holds %q, want %q", got, wantSkin)
		}
		if got := tt.Find(widget.ByLabel("Palette")).Value(); got != wantPalette {
			t.Errorf("the Palette picker holds %q, want %q", got, wantPalette)
		}
		tt.SetDark(dark)
		tt.Frame()
		return tt
	}
	for _, sk := range skins {
		for _, dark := range []bool{false, true} {
			mode := map[bool]string{false: "light", true: "dark"}[dark]
			tt := open(sk.s, dark)
			for _, page := range pages {
				tt.Find(widget.ByLabel(page)).Click()
				write(tt, fmt.Sprintf("%s-%s-%s", sk.name, mode, slug(page)))
			}
		}
	}

	// Under Calm Blue, light: a menu bar item hovered, a grid row selected.
	tt := open(skin.Calm, false)
	tt.Find(widget.ByLabel("Containers")).Click()
	file := tt.FindAll(widget.ByLabel("File"))
	if len(file) == 0 {
		t.Fatalf("no menu bar item File: %q", tt.Texts())
	}
	b := file[0].Bounds()
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	write(tt, "calm-blue-light-menu-hover")

	tt = open(skin.Calm, false)
	tt.Find(widget.ByLabel("Grid")).Click()
	tt.Frame()
	// The first row is the header: pick the data row 3, and check the grid
	// holds exactly it, drawn unlike the row 4 under it.
	tt.Find(widget.ByRole(ui.RoleRow).Label("3")).Click()
	tt.Frame()
	var picked, other ui.Rect
	selected := 0
	for _, n := range tt.Nodes() {
		if n.Role != ui.RoleRow {
			continue
		}
		if n.Checked { // a grid row reports its selection as checked
			selected++
			picked = n.Bounds
			if n.Label != "3" {
				t.Errorf("the selected row is %q, want 3", n.Label)
			}
		}
		if n.Label == "4" {
			other = n.Bounds
		}
	}
	if selected != 1 {
		t.Fatalf("the grid has %d selected rows, want 1", selected)
	}
	img := tt.Image() // a DIP is scale pixels; x = 800 DIPs is past the row's text
	x := int(800 * scale)
	in := img.RGBAAt(x, int((picked.Y+picked.H/2)*scale))
	out := img.RGBAAt(x, int((other.Y+other.H/2)*scale))
	t.Logf("pixel in the selected row %v, in an unselected row %v", in, out)
	if in == out {
		t.Errorf("the selected row is drawn as the unselected one: both %v", in)
	}
	write(tt, "calm-blue-light-grid-row-selected")
}

package main

import (
	"fmt"
	"image"
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
		{"classic-indigo", skin.Classic}, // Compact, Classic's own density
		{"classic-slate", skin.Classic.WithPalette(skin.ClassicSlate)},
		{"highcontrast", skin.HighContrast},
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
		wantSkin, wantPalette := "none", skin.Blue.Name()
		if s != nil {
			wantSkin, wantPalette = strings.ToLower(s.Name()), ""
			pick(tt, "Skin", s.Name())
			if s.Palette() != nil {
				wantPalette = s.Palette().Name()
				pick(tt, "Palette", wantPalette)
			}
			if a := skin.Active(); a == nil || a.Name() != s.Name() || a.Palette() != s.Palette() || a.Density() != s.Density() {
				t.Fatalf("the studio's skin is not %s", s.Name())
			}
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

	// The Edit tab, light and dark, under Calm Blue.
	for _, dark := range []bool{false, true} {
		tt := open(skin.Calm, dark)
		tt.Find(widget.ByRole(ui.RoleTab).Label("Edit")).Click()
		write(tt, "calm-blue-"+map[bool]string{false: "light", true: "dark"}[dark]+"-edit")
	}
}

// TestShotsMatchMain compares the shots of main (MYGO_SKIN_SHOTS_MAIN) with
// the branch's (MYGO_SKIN_SHOTS): outside the rectangle the Edit tab's header
// takes in the tab strip (from the right edge of the eighth tab to the
// window's, the strip's height) every shot is the same, but the chart page's,
// where the spinner turns, the dialogs page's, which shows the status
// controls under its message boxes, and but a box of 20 by 12 pixels of the numbers
// page, where the date picker shows the time. It is skipped without both variables, and reads
// shots taken at scale 1.
func TestShotsMatchMain(t *testing.T) {
	mainDir, dir := os.Getenv("MYGO_SKIN_SHOTS_MAIN"), os.Getenv("MYGO_SKIN_SHOTS")
	if mainDir == "" || dir == "" {
		t.Skip("MYGO_SKIN_SHOTS_MAIN and MYGO_SKIN_SHOTS name the directories of main's shots and the branch's")
	}
	reset(t)
	type choice struct{ skin, palette string }
	studios := map[string]choice{
		"none": {}, "calm-blue": {"Calm", ""}, "calm-cham": {"Calm", skin.Cham.Name()},
		"classic-indigo": {"Classic", ""}, "classic-slate": {"Classic", skin.ClassicSlate.Name()}, "highcontrast": {"HighContrast", ""},
	}
	// strip returns the rectangle of the Edit tab's header and what follows
	// it, in the studio under the skin named name.
	strip := func(name string) ui.Rect {
		skin.Use(nil)
		tt := widget.TestForm(t, newStudio())
		tt.SetSize(1280, 800)
		if c := studios[name]; c.skin != "" {
			pick(tt, "Skin", c.skin)
			if c.palette != "" {
				pick(tt, "Palette", c.palette)
			}
		}
		tt.Frame()
		last := tt.Find(widget.ByRole(ui.RoleTab).Label("Dialogs and messages")).Bounds()
		edit := tt.Find(widget.ByRole(ui.RoleTab).Label("Edit")).Bounds()
		return ui.Rect{X: last.X + last.W - 2, Y: edit.Y - 2, W: 1280, H: edit.H + 4}
	}
	files, err := filepath.Glob(filepath.Join(mainDir, "*.png"))
	// Main's eight pages in six skins, light and dark, and its four shots of
	// states: 100.
	if err != nil || len(files) != 100 {
		t.Fatalf("%d shots of main in %s (%v), want 100", len(files), mainDir, err)
	}
	load := func(path string) image.Image {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	rects := map[string]ui.Rect{}
	compared, inside := 0, 0
	for _, file := range files {
		base := filepath.Base(file)
		if strings.HasSuffix(base, "-chart-and-progress.png") {
			continue // the spinner turns between two runs
		}
		if strings.HasSuffix(base, "-dialogs-and-messages.png") {
			continue // the badges, info bars and empty state are new on this page
		}
		name := ""
		for k := range studios {
			if strings.HasPrefix(base, k+"-") && len(k) > len(name) {
				name = k
			}
		}
		if name == "" {
			t.Fatalf("shot %s is of no skin the test knows", base)
		}
		r, ok := rects[name]
		if !ok {
			r = strip(name)
			rects[name] = r
			t.Logf("%s: the Edit header's rectangle is %v", name, r)
		}
		a, b := load(file), load(filepath.Join(dir, base))
		if a.Bounds() != b.Bounds() {
			t.Errorf("%s: sizes differ: %v and %v", base, a.Bounds(), b.Bounds())
			continue
		}
		compared++
		outside := 0
		box := image.Rectangle{Min: a.Bounds().Max, Max: a.Bounds().Min} // of the pixels outside
		for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
			for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
				if a.At(x, y) == b.At(x, y) {
					continue
				}
				if float32(x) >= r.X && float32(y) >= r.Y && float32(y) < r.Y+r.H {
					inside++
				} else {
					outside++
					box = box.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
		// The picker's clock: the numbers page shows the time of day in a
		// box of a few digits, which differs between two runs a minute apart.
		if strings.HasSuffix(base, "-numbers-and-dates.png") && box.Dx() <= 20 && box.Dy() <= 12 {
			outside = 0
		}
		if outside != 0 {
			t.Errorf("%s: %d pixels differ outside the Edit header's rectangle %v, within %v", base, outside, r, box)
		}
	}
	// The Edit header is on main now, so the rectangles may match too.
	t.Logf("%d shots compared (the 12 of the chart page and the 12 of the dialogs page skipped); %d pixels differ inside the rectangles", compared, inside)
}

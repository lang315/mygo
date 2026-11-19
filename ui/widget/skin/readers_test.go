package skin

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// readerSentinel is Calm with a placeholder, a settings row, an icon and a
// drop-down shadow no plain theme or Calm has, in both appearances.
func readerSentinel() *Skin {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	for _, tk := range []*Tokens{&l, &d} {
		tk.TextPlaceholder = ui.Hex("#ff00aa")
		tk.SettingsRowHeight = 61
		tk.Icon = 23
		tk.PopupShadow.Color = ui.Hex("#00ff0080")
	}
	return New("Sentinel", l, d)
}

// showSkin shows w in a window of width by height, dark or light, with text
// at scale and the skin s on (nil for none).
func showSkin(t *testing.T, w widget.Control, width, height int, s *Skin, dark bool, scale float32) *widget.Tester {
	t.Helper()
	t.Cleanup(func() { Use(nil) })
	tt := widget.NewTester(t, w, width, height)
	Use(s)
	tt.SetDark(dark)
	tt.SetPreferences(ui.Preferences{TextScale: scale})
	tt.Frame()
	return tt
}

// nodeOf returns the bounds of the first node of role r (and label, unless
// ""), failing the test without one.
func nodeOf(t *testing.T, tt *widget.Tester, r ui.Role, label string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r && (label == "" || n.Label == label) {
			return n.Bounds
		}
	}
	t.Fatalf("no %v node %q", r, label)
	return ui.Rect{}
}

// nearest is the pixel of r nearest to c, and how far it is.
func nearest(tt *widget.Tester, r ui.Rect, c ui.Color) (ui.Color, float64) {
	img := tt.Image()
	best, most := ui.Color{}, 1e9
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := colorAt(img, x, y)
			if d := dist(p, c); d < most {
				best, most = p, d
			}
		}
	}
	return best, most
}

func appearance(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

// A TextBox's placeholder takes the skin's TextPlaceholder, light and dark,
// and its plain color again after Use(nil).
func TestPlaceholderTakesTheSkin(t *testing.T) {
	pink := ui.Hex("#ff00aa")
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			tt := showSkin(t, widget.TextBox().SetPlaceholder("x"), 300, 120, readerSentinel(), dark, 1)
			box := nodeOf(t, tt, ui.RoleTextField, "")
			if c, d := nearest(tt, box, pink); d > 8 {
				t.Errorf("with the sentinel skin the placeholder's nearest pixel to %v is %v (%.0f away)", pink, c, d)
			}
			Use(nil)
			tt.Frame()
			if c, d := nearest(tt, box, pink); d < 100 {
				t.Errorf("after Use(nil) the placeholder still paints %v (%.0f from %v)", c, d, pink)
			}
			// Calm's own, the dark one in the dark.
			Use(Calm)
			tt.Frame()
			want, other := ui.Hex("#6b7280"), ui.Hex("#8e96a1")
			if dark {
				want, other = other, want
			}
			// The ink of the placeholder: its pixel farthest from the field.
			field := px(tt, box.X+box.W-6, box.Y+box.H/2)
			ink, _ := strongest(tt, ui.Rect{X: box.X + 2, Y: box.Y + 2, W: box.W / 2, H: box.H - 4}, field)
			if dist(ink, want) > 8 || dist(ink, want) >= dist(ink, other) {
				t.Errorf("with Calm the placeholder's ink is %v, want %v (not %v)", ink, want, other)
			}
		})
	}
}

// A MultilineTextBox's placeholder takes the skin's TextPlaceholder too.
func TestMultilinePlaceholderTakesTheSkin(t *testing.T) {
	pink, whole := ui.Hex("#ff00aa"), ui.Rect{W: 300, H: 200}
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			tt := showSkin(t, widget.MultilineTextBox().SetPlaceholder("x"), 300, 200, readerSentinel(), dark, 1)
			if c, d := nearest(tt, whole, pink); d > 8 {
				t.Errorf("with the sentinel skin the placeholder's nearest pixel to %v is %v (%.0f away)", pink, c, d)
			}
			Use(nil)
			tt.Frame()
			if c, d := nearest(tt, whole, pink); d < 100 {
				t.Errorf("after Use(nil) the placeholder still paints %v (%.0f from %v)", c, d, pink)
			}
		})
	}
}

// The app's own text color stays on the text, and the placeholder takes the
// skin's.
func TestOwnStyleWinsOverPlaceholder(t *testing.T) {
	own, pink := ui.Hex("#123456"), ui.Hex("#ff00aa")
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			typed := widget.TextBox().SetText("x").Apply(widget.NewStyle().TextColor(widget.Hex("#123456")))
			empty := widget.TextBox().SetPlaceholder("x").Apply(widget.NewStyle().TextColor(widget.Hex("#123456")))
			tt := showSkin(t, widget.VBox(8).Add(typed, empty), 300, 240, readerSentinel(), dark, 1)
			var boxes []ui.Rect
			for _, n := range tt.Nodes() {
				if n.Role == ui.RoleTextField {
					boxes = append(boxes, n.Bounds)
				}
			}
			if len(boxes) != 2 {
				t.Fatalf("%d text fields, want 2", len(boxes))
			}
			if c, d := nearest(tt, boxes[0], own); d > 8 {
				t.Errorf("the text's nearest pixel to its own %v is %v (%.0f away)", own, c, d)
			}
			if c, d := nearest(tt, boxes[1], pink); d > 8 {
				t.Errorf("the placeholder's nearest pixel to the skin's %v is %v (%.0f away)", pink, c, d)
			}
		})
	}
}

// A settings row is at least the skin's SettingsRowHeight, at the text
// scale, and below it again after Use(nil).
func TestSettingsRowsTakeTheSkin(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, scale := range []float32{1, 1.5} {
			t.Run(fmt.Sprintf("%s-%g", appearance(dark), scale), func(t *testing.T) {
				f := widget.Fields().SettingsRows().Row("Sounds", widget.Switch())
				tt := showSkin(t, widget.VBox(0).Add(widget.Label("above"), f, widget.Label("below")), 600, 400, readerSentinel(), dark, scale)
				if h, want := rowHeight(t, tt), 61*scale; h < want-0.01 {
					t.Errorf("the row is %v high, want at least %v", h, want)
				}
				Use(nil)
				tt.Frame()
				if h := rowHeight(t, tt); h >= 61 {
					t.Errorf("after Use(nil) the row is %v high, want below 61", h)
				}
			})
		}
	}
}

// A form's rows in the plain layout keep their height under the skin: only
// settings rows take SettingsRowHeight. A settings row's label stays level
// with its control.
func TestSettingsRowHeightOnlyInSettingsRows(t *testing.T) {
	for _, f := range []*widget.FieldsWidget{
		widget.Fields().Row("Name", widget.Switch()),
		widget.Fields().Stacked().Row("Name", widget.Switch()),
	} {
		tt := showSkin(t, widget.VBox(0).Add(widget.Label("above"), f, widget.Label("below")), 600, 400, readerSentinel(), false, 1)
		if h := rowHeight(t, tt); h >= 61 {
			t.Errorf("a form not of settings rows is %v high under the sentinel skin, want its own height", h)
		}
	}
	// The label and the control at the middle of the row.
	s := widget.Fields().SettingsRows().Row("Sounds", widget.Button("Test"))
	tt := showSkin(t, widget.VBox(0).Add(widget.Label("above"), s, widget.Label("below")), 600, 400, readerSentinel(), false, 1)
	a, _ := tt.Tester.Find("above")
	middle := a.Y + a.H + rowHeight(t, tt)/2
	label, _ := tt.Tester.Find("Sounds")
	button := tt.Find(widget.ByLabel("Test")).Bounds()
	for _, r := range []ui.Rect{label, button} {
		if d := r.Y + r.H/2 - middle; d > 1 || d < -1 {
			t.Errorf("%v is %v from the row's middle", r, d)
		}
	}
}

// A heading, a divider and a group break keep their own height between
// settings rows of the skin's: only the rows of fields take it.
func TestSettingsRowsKeepHeadingsAndDividers(t *testing.T) {
	f := widget.Fields().SettingsRows().Section("General").Row("Alpha", widget.Switch()).Row("Beta", widget.Switch()).
		Section("").Row("Gamma", widget.Switch())
	tt := showSkin(t, f, 600, 400, readerSentinel(), false, 1)
	mid := func(s string) float32 {
		r, ok := tt.Tester.Find(s)
		if !ok {
			t.Fatalf("no %q", s)
		}
		return r.Y + r.H/2
	}
	head, _ := tt.Tester.Find("General")
	// Each step is half a row, the gaps, what is between, and half a row.
	for _, c := range []struct {
		name     string
		got      float32
		min, max float32
	}{
		{"General to Alpha", mid("Alpha") - (head.Y + head.H), 61 / 2, 61/2 + 12},
		{"Alpha to Beta, over a divider", mid("Beta") - mid("Alpha"), 61, 61 + 20},
		{"Beta to Gamma, over a break", mid("Gamma") - mid("Beta"), 61, 61 + 30},
	} {
		if c.got < c.min || c.got > c.max {
			t.Errorf("%s is %v, want %v to %v", c.name, c.got, c.min, c.max)
		}
	}
}

// rowHeight is the height of a form of one row between the labels "above"
// and "below", with no gap: the row's.
func rowHeight(t *testing.T, tt *widget.Tester) float32 {
	t.Helper()
	a, ok := tt.Tester.Find("above")
	b, ok2 := tt.Tester.Find("below")
	if !ok || !ok2 {
		t.Fatalf("no label above or below the form: %q", tt.Texts())
	}
	return b.Y - (a.Y + a.H)
}

// solid is a bitmap of one color, for an icon whose side the pixels show.
func solid(c color.RGBA) *ui.Bitmap { return ui.NewBitmap(solidImage(c)) }

func solidImage(c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}

// greenPNG is the icon's green as the bytes of a PNG.
func greenPNG(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, solidImage(iconGreen)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// iconRuns are the widths of every run of the icon's green across a row of
// pixels of r: one per row of each icon drawn.
func iconRuns(tt *widget.Tester, r ui.Rect) []int {
	img := tt.Image()
	var runs []int
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		run := 0
		for x := int(r.X); x <= int(r.X+r.W); x++ {
			p := img.RGBAAt(x, y)
			if x < int(r.X+r.W) && int(p.G)-int(p.R) > 100 && int(p.G)-int(p.B) > 100 {
				run++
			} else if run > 0 {
				runs = append(runs, run)
				run = 0
			}
		}
	}
	return runs
}

var iconGreen = color.RGBA{0, 200, 0, 255}

// iconRun is the longest run of the icon's green, across a row of pixels of
// r, counting a pixel at least half covered: the icon's drawn width.
func iconRun(tt *widget.Tester, r ui.Rect) float32 {
	img := tt.Image()
	most := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		run := 0
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := img.RGBAAt(x, y)
			if int(p.G)-int(p.R) > 100 && int(p.G)-int(p.B) > 100 {
				run++
				most = max(most, run)
			} else {
				run = 0
			}
		}
	}
	return float32(most)
}

// sideOf says whether a run of got whole pixels draws a side of want DIPs:
// within a pixel, so 16 is 16 and 34.5 is 34 or 35.
func sideOf(got, want float32) bool { return got > want-1 && got < want+1 }

type iconRow struct{ ID, Name string }

// The icons of a TextBox, a DataGridView, a ListBox, a TreeView and a small
// icon ListView are the skin's Icon at the text scale, and 16 without a
// skin; a ListView's large icons keep their SetIconSize.
func TestIconsTakeTheSkin(t *testing.T) {
	green := solid(iconGreen)
	img := func(*iconRow) *ui.Bitmap { return green }
	rows := []iconRow{{"1", "Ada"}}
	key := func(r *iconRow) string { return r.ID }
	name := func(r *iconRow) string { return r.Name }
	cases := []struct {
		name string
		make func() widget.Control
	}{
		{"TextBox", func() widget.Control { return widget.TextBox().Icon(green) }},
		{"DataGridView", func() widget.Control {
			return widget.DataGridView(key, widget.Column[iconRow]{Title: "Name", Text: name, Image: img}).SetRows(rows).SetSize(0, 200)
		}},
		{"ListBox", func() widget.Control { return widget.ListBox(key, name).SetImage(img).SetItems(rows).SetSize(0, 200) }},
		{"TreeView", func() widget.Control {
			return widget.TreeView(key, name).SetImage(img).SetRoots(rows).SetSize(0, 200)
		}},
		{"ListView", func() widget.Control {
			return widget.ListView(key, widget.Column[iconRow]{Title: "Name", Text: name}).SetImage(img).
				SetView(widget.ViewSmallIcon).SetRows(rows).SetSize(0, 200)
		}},
	}
	whole := ui.Rect{W: 600, H: 400}
	for _, c := range cases {
		for _, dark := range []bool{false, true} {
			for _, scale := range []float32{1, 1.5} {
				t.Run(fmt.Sprintf("%s-%s-%g", c.name, appearance(dark), scale), func(t *testing.T) {
					tt := showSkin(t, c.make(), 600, 400, readerSentinel(), dark, scale)
					if got, want := iconRun(tt, whole), 23*scale; !sideOf(got, want) {
						t.Errorf("with the sentinel skin the icon is %v wide, want %v", got, want)
					}
					Use(nil)
					tt.Frame()
					if got := iconRun(tt, whole); !sideOf(got, 16) {
						t.Errorf("after Use(nil) the icon is %v wide, want 16", got)
					}
				})
			}
		}
	}
	// A ComboBox section's icon, in its open drop-down.
	for _, scale := range []float32{1, 1.5} {
		combo := widget.ComboBox().AddSection("Fruits", greenPNG(t), widget.Option{Value: "a", Text: "Apple"}).SetValue("a")
		tt := showSkin(t, widget.VBox(8).Add(combo), 600, 400, readerSentinel(), false, scale)
		if err := tt.Click("Apple"); err != nil {
			t.Fatal(err)
		}
		tt.Settle()
		if got, want := iconRun(tt, whole), 23*scale; !sideOf(got, want) {
			t.Errorf("at %g, a ComboBox section's icon is %v wide, want %v", scale, got, want)
		}
		Use(nil)
		tt.Frame()
		if got := iconRun(tt, whole); !sideOf(got, 16) {
			t.Errorf("at %g after Use(nil), a ComboBox section's icon is %v wide, want 16", scale, got)
		}
	}
	// Large icons keep the size the app set.
	lv := widget.ListView(key, widget.Column[iconRow]{Title: "Name", Text: name}).SetImage(img).
		SetView(widget.ViewLargeIcon).SetIconSize(40).SetRows(rows).SetSize(0, 200)
	tt := showSkin(t, lv, 600, 400, readerSentinel(), false, 1.5)
	if got := iconRun(tt, whole); !sideOf(got, 40) {
		t.Errorf("a large icon of SetIconSize(40) is %v wide under the sentinel skin, want 40", got)
	}
}

// The rows a DataGridView keeps from frame to frame show the new icon side
// in the frame after a skin switch: none stays at the old one.
func TestIconsRepaintKeptRows(t *testing.T) {
	green := solid(iconGreen)
	var rows []iconRow
	for i := range 50 {
		rows = append(rows, iconRow{fmt.Sprint(i), fmt.Sprint("Row ", i)})
	}
	key := func(r *iconRow) string { return r.ID }
	col := widget.Column[iconRow]{Title: "Name", Text: func(r *iconRow) string { return r.Name },
		Image: func(*iconRow) *ui.Bitmap { return green }}
	for _, c := range []struct {
		name string
		w    widget.Control
	}{
		{"DataGridView", widget.DataGridView(key, col).SetRows(rows).SetSize(0, 360)},
		{"ListView", widget.ListView(key, col).SetView(widget.ViewSmallIcon).SetRows(rows).SetSize(0, 360)},
	} {
		t.Run(c.name, func(t *testing.T) {
			tt := showSkin(t, c.w, 600, 400, Calm, false, 1)
			tt.Settle()
			if runs := iconRuns(tt, ui.Rect{W: 600, H: 400}); len(runs) < 5*16 || slices.Max(runs) > 17 {
				t.Fatalf("with Calm, %d rows of icon pixels, the widest %d: want many, 16 wide", len(runs), slices.Max(runs))
			}
			Use(readerSentinel())
			tt.Settle()
			runs := iconRuns(tt, ui.Rect{W: 600, H: 400})
			if len(runs) < 5*23 {
				t.Fatalf("%d rows of icon pixels, want several icons", len(runs))
			}
			for _, w := range runs {
				if !sideOf(float32(w), 23) {
					t.Fatalf("a row of an icon's pixels is %d wide, want 23: a kept row is stale", w)
				}
			}
		})
	}
}

// The rows a ListView keeps take the new icon side when the text scale
// changes under a skin.
func TestListViewKeptRowsFollowTheTextScale(t *testing.T) {
	green := solid(iconGreen)
	var rows []iconRow
	for i := range 50 {
		rows = append(rows, iconRow{fmt.Sprint(i), fmt.Sprint("Row ", i)})
	}
	lv := widget.ListView(func(r *iconRow) string { return r.ID }, widget.Column[iconRow]{Title: "Name",
		Text: func(r *iconRow) string { return r.Name }, Image: func(*iconRow) *ui.Bitmap { return green }}).
		SetView(widget.ViewSmallIcon).SetRows(rows).SetSize(0, 360)
	tt := showSkin(t, lv, 600, 400, readerSentinel(), false, 1)
	tt.Settle()
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Settle()
	runs := iconRuns(tt, ui.Rect{W: 600, H: 400})
	if len(runs) < 3*34 {
		t.Fatalf("%d rows of icon pixels, want several icons", len(runs))
	}
	for _, w := range runs {
		if !sideOf(float32(w), 34.5) {
			t.Fatalf("a row of an icon's pixels is %d wide, want 34.5: a kept row is stale", w)
		}
	}
}

// A ListView in the list view gives an item room for the skin's icon and
// its margin, over the row height.
func TestListViewRowsFitTheIcon(t *testing.T) {
	green := solid(iconGreen)
	rows := []iconRow{{"1", "Ada"}, {"2", "Grace"}}
	lv := widget.ListView(func(r *iconRow) string { return r.ID }, widget.Column[iconRow]{Title: "Name",
		Text: func(r *iconRow) string { return r.Name }, Image: func(*iconRow) *ui.Bitmap { return green }}).
		SetView(widget.ViewList).SetRows(rows).SetSize(0, 300)
	tt := showSkin(t, lv, 600, 400, readerSentinel(), false, 1)
	// The item's text fills its height.
	for _, s := range []string{"Ada", "Grace"} {
		if r, ok := tt.Tester.Find(s); !ok || r.H < 31-0.01 {
			t.Errorf("the item %q is %v high, want at least the icon and its margin, 31", s, r.H)
		}
	}
}

// An open ComboBox's drop-down casts the skin's PopupShadow.
func TestDropDownShadowTakesTheSkin(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			combo := widget.ComboBox(widget.Option{Value: "a", Text: "Alpha"}, widget.Option{Value: "b", Text: "Beta"}).SetValue("a")
			tt := showSkin(t, widget.VBox(8).Add(combo), 400, 400, readerSentinel(), dark, 1)
			if err := tt.Click("Alpha"); err != nil {
				t.Fatal(err)
			}
			tt.Settle()
			beta, ok := tt.Tester.Find("Beta")
			if !ok {
				t.Fatalf("the drop-down did not open: %q", tt.Texts())
			}
			// From the last option down past the panel's edge: the shadow's band.
			band := ui.Rect{X: beta.X, Y: beta.Y + beta.H, W: 60, H: 30}
			if g := greenest(tt, band); g < 30 {
				t.Errorf("the shadow under the drop-down is %d green over red and blue, want the sentinel's green", g)
			}
			Use(Calm)
			tt.Frame()
			if g := greenest(tt, band); g > 8 {
				t.Errorf("with Calm the shadow under the drop-down is %d green: the test sees no change", g)
			}
		})
	}
}

// Calm's drop-downs keep the shadow they had before skins, in both
// appearances.
func TestCalmDropDownShadowUnchanged(t *testing.T) {
	want := Shadow{Y: 6, Blur: 20, Color: ui.RGBA(0, 0, 0, 0.18)}
	if l, d := Calm.Tokens(false).PopupShadow, Calm.Tokens(true).PopupShadow; l != want || d != want {
		t.Errorf("Calm's PopupShadow is %+v and %+v, want %+v", l, d, want)
	}
}

// greenest is how much more green than red and blue the greenest pixel of r is.
func greenest(tt *widget.Tester, r ui.Rect) int {
	img := tt.Image()
	most := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := img.RGBAAt(x, y)
			most = max(most, int(p.G)-max(int(p.R), int(p.B)))
		}
	}
	return most
}

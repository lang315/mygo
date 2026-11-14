package widget

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func domainTester(t *testing.T, d *DomainUpDownWidget) (*Tester, *Node) {
	t.Helper()
	tt := NewTester(t, d, 400, 200)
	return tt, tt.Find(ByRole(ui.RoleStepper))
}

var days = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

func TestDomainUpDownShowsTheSelectedItem(t *testing.T) {
	d := DomainUpDown(days...).SetSelectedItem("Wed")
	tt, _ := domainTester(t, d)
	if !tt.Has(ByLabel("Wed")) || tt.Has(ByLabel("Tue")) || d.SelectedIndex() != 2 || d.SelectedItem() != "Wed" {
		t.Fatalf("index %d item %q texts %q", d.SelectedIndex(), d.SelectedItem(), tt.Texts())
	}
}

// As in WinForms, Up goes to the item before, Down to the next one.
func TestDomainUpDownUpIsThePreviousItemAndDownTheNext(t *testing.T) {
	changed := make(chan string, 8)
	var d *DomainUpDownWidget
	d = DomainUpDown(days...).SetSelectedIndex(2).OnChange(func() { changed <- d.SelectedItem() })
	tt, s := domainTester(t, d)
	s.Key(0, ui.KeyUp)
	if got := received(t, changed); got != "Tue" || d.SelectedIndex() != 1 {
		t.Fatalf("Up: OnChange saw %q, index %d", got, d.SelectedIndex())
	}
	s.Key(0, ui.KeyDown)
	s.Key(0, ui.KeyDown)
	received(t, changed)
	if got := received(t, changed); got != "Thu" {
		t.Fatalf("Down twice from Tue: %q", got)
	}
	if !tt.Has(ByLabel("Thu")) {
		t.Fatalf("the field still shows %q", tt.Texts())
	}
}

func TestDomainUpDownStopsAtTheEndsUnlessItWraps(t *testing.T) {
	d := DomainUpDown(days...)
	_, s := domainTester(t, d)
	s.Key(0, ui.KeyUp)
	if d.SelectedIndex() != 0 {
		t.Fatalf("Up from the first item went to %d", d.SelectedIndex())
	}
	d.SetWrap(true)
	s.Key(0, ui.KeyUp)
	if d.SelectedItem() != "Fri" {
		t.Fatalf("Up from the first item with Wrap gave %q", d.SelectedItem())
	}
	s.Key(0, ui.KeyDown)
	if d.SelectedItem() != "Mon" {
		t.Fatalf("Down from the last item with Wrap gave %q", d.SelectedItem())
	}
}

func TestDomainUpDownSettersDoNotReportAndIgnoreWhatIsNotThere(t *testing.T) {
	changed := make(chan struct{}, 2)
	d := DomainUpDown(days...).OnChange(func() { changed <- struct{}{} })
	tt, _ := domainTester(t, d)
	d.SetSelectedItem("Fri")
	d.SetSelectedItem("Sun") // not an item
	tt.Settle()
	if d.SelectedItem() != "Fri" || !tt.Has(ByLabel("Fri")) {
		t.Fatalf("item %q", d.SelectedItem())
	}
	d.SetSelectedIndex(99) // clamps
	if d.SelectedIndex() != 4 {
		t.Fatalf("index %d", d.SelectedIndex())
	}
	select {
	case <-changed:
		t.Fatal("a setter ran OnChange")
	default:
	}
}

func TestDomainUpDownSetItemsKeepsTheSelectedItemIfItStays(t *testing.T) {
	d := DomainUpDown(days...).SetSelectedItem("Wed")
	d.SetItems("Tue", "Wed", "Sat")
	if d.SelectedItem() != "Wed" || d.SelectedIndex() != 1 {
		t.Fatalf("item %q index %d", d.SelectedItem(), d.SelectedIndex())
	}
	d.SetItems("A", "B")
	if d.SelectedIndex() != 0 {
		t.Fatalf("index %d after the item left", d.SelectedIndex())
	}
	d.SetItems()
	if d.SelectedIndex() != -1 || d.SelectedItem() != "" {
		t.Fatalf("empty list: index %d item %q", d.SelectedIndex(), d.SelectedItem())
	}
}

func TestDomainUpDownBindsTheItem(t *testing.T) {
	v := "Thu"
	d := Bind(DomainUpDown(days...), func() string { return v }, func(s string) { v = s })
	tt, s := domainTester(t, d)
	if !tt.Has(ByLabel("Thu")) {
		t.Fatalf("shows %q, not the source's Thu", tt.Texts())
	}
	s.Key(0, ui.KeyDown)
	if v != "Fri" {
		t.Fatalf("the source got %q", v)
	}
}

func TestDomainUpDownOneItemWithWrapReportsNoChange(t *testing.T) {
	changed := make(chan struct{}, 2)
	d := DomainUpDown("Only").SetWrap(true).OnChange(func() { changed <- struct{}{} })
	_, s := domainTester(t, d)
	s.Key(0, ui.KeyUp)
	<-posted()
	select {
	case <-changed:
		t.Fatal("Up on the only item ran OnChange")
	default:
	}
	if d.SelectedItem() != "Only" {
		t.Fatalf("item %q", d.SelectedItem())
	}
}

func TestDomainUpDownSetItemsKeepsTheIndexOfAnUnchangedItem(t *testing.T) {
	d := DomainUpDown("A", "X", "X").SetSelectedIndex(2)
	d.SetItems("A", "X", "X")
	if d.SelectedIndex() != 2 {
		t.Fatalf("index %d: the item at index 2 did not change", d.SelectedIndex())
	}
}

func TestDomainUpDownEmptyListAndIndexClamp(t *testing.T) {
	d := DomainUpDown()
	tt, s := domainTester(t, d)
	if d.SelectedIndex() != -1 || d.SelectedItem() != "" || !s.Disabled() {
		t.Fatalf("empty: index %d item %q disabled %v", d.SelectedIndex(), d.SelectedItem(), s.Disabled())
	}
	d.SetItems("A", "B", "C")
	tt.Settle()
	d.SetSelectedIndex(-7)
	if d.SelectedIndex() != 0 {
		t.Fatalf("index %d after SetSelectedIndex(-7)", d.SelectedIndex())
	}
	d.SetSelectedIndex(99)
	if d.SelectedIndex() != 2 {
		t.Fatalf("index %d after SetSelectedIndex(99)", d.SelectedIndex())
	}
}

func TestDomainUpDownExposesTheItemAsTheValue(t *testing.T) {
	d := DomainUpDown(days...).SetSelectedItem("Tue")
	_, s := domainTester(t, d)
	if s.Value() != "Tue" {
		t.Fatalf("the stepper's value is %q, want the item", s.Value())
	}
}

// updownLooks runs fn with no skin and with the test skin, light and dark.
func updownLooks(t *testing.T, fn func(t *testing.T, dark bool)) {
	for _, skinned := range []bool{false, true} {
		for _, dark := range []bool{false, true} {
			name := map[bool]string{false: "nil", true: "skin"}[skinned] + map[bool]string{false: "/light", true: "/dark"}[dark]
			t.Run(name, func(t *testing.T) {
				if skinned {
					useSkin(t)
				}
				fn(t, dark)
			})
		}
	}
}

// boxRows returns the first and last pixel row of column x that is not bg.
func boxRows(img *image.RGBA, x int, bg color.RGBA) (first, last int) {
	first, last = -1, -1
	for y := 0; y < img.Bounds().Dy(); y++ {
		if img.RGBAAt(x, y) != bg {
			if first < 0 {
				first = y
			}
			last = y
		}
	}
	return first, last
}

func TestDomainUpDownMatchesNumericUpDown(t *testing.T) {
	updownLooks(t, func(t *testing.T, dark bool) {
		tt := NewTester(t, HBox(8).Add(DomainUpDown("North", "South"), NumericUpDown(0, 10, 1)), 400, 200)
		tt.SetDark(dark)
		tt.Frame()
		img := tt.Image()
		bg := img.RGBAAt(2, 190)
		text := tt.Find(ByLabel("North")).Bounds()
		num := tt.Find(ByRole(ui.RoleTextField)).Bounds()
		df, dl := boxRows(img, int(text.X)-3, bg)
		nf, nl := boxRows(img, int(num.X+num.W/2), bg)
		if dl-df != nl-nf {
			t.Fatalf("DomainUpDown rows %d..%d, NumericUpDown rows %d..%d", df, dl, nf, nl)
		}
		if s := tt.Find(ByRole(ui.RoleStepper)).Bounds(); math.Abs(float64(s.H)-float64(nl-nf+1)) > 1 {
			t.Fatalf("DomainUpDown's arrows are %v high, NumericUpDown's box %d", s.H, nl-nf+1)
		}
	})
}

func TestDomainUpDownArrowsInsideItsBox(t *testing.T) {
	updownLooks(t, func(t *testing.T, dark bool) {
		tt := NewTester(t, DomainUpDown("North", "South"), 400, 200)
		tt.SetDark(dark)
		tt.Frame()
		img := tt.Image()
		bg := img.RGBAAt(2, 190)
		text := tt.Find(ByLabel("North")).Bounds()
		s := tt.Find(ByRole(ui.RoleStepper)).Bounds()
		f, l := boxRows(img, int(text.X)-3, bg)
		if math.Abs(float64(s.Y)-float64(f)) > 1 || math.Abs(float64(s.Y+s.H)-float64(l+1)) > 1 {
			t.Fatalf("stepper %v, field rows %d..%d", s, f, l)
		}
		y := int(s.Y + s.H/2)
		// The pixel left of the stepper is the field's border: a gap would
		// show the background there. (The field's fill may equal bg.)
		if img.RGBAAt(int(s.X)-1, y) == bg {
			t.Fatalf("background just left of the stepper %v: a gap", s)
		}
	})
}

// The field and the arrows are one box, as NumericUpDown's: one line
// between them, and no notch where they meet, whatever the text's height:
// text scales make the box a fractional height high (1.1: 30.84 DIPs), as
// Windows' font did, whose edges land on the nearest device rows.
func TestDomainUpDownIsOneBox(t *testing.T) {
	updownLooks(t, func(t *testing.T, dark bool) {
		for _, sc := range [][2]float32{{1, 1}, {2, 1}, {1, 14.0 / 13}, {2, 14.0 / 13}, {1, 1.1}, {1.25, 1}} {
			scale := sc[0]
			tt := NewTester(t, DomainUpDown("North", "South"), 400, 200)
			tt.SetDark(dark)
			tt.SetScale(scale)
			tt.SetPreferences(ui.Preferences{TextScale: sc[1]})
			tt.Frame()
			img := tt.Image()
			bg := img.RGBAAt(2, int(190*scale))
			text := tt.Find(ByLabel("North")).Bounds()
			s := tt.Find(ByRole(ui.RoleStepper)).Bounds()
			y := int((s.Y + s.H/2) * scale)
			// The field's left border, at mid-height, is the border's color.
			var line color.RGBA
			for x := 0; x < int(text.X*scale); x++ {
				if c := img.RGBAAt(x, y); c != bg {
					line = c
					break
				}
			}
			// Between the text and the arrows' glyph, one run of it.
			var runs, width, at int
			for x, in := int((text.X+text.W)*scale), false; x < int((s.X+4)*scale); x++ {
				on := img.RGBAAt(x, y) == line
				if on {
					if !in {
						runs, at = runs+1, x
					}
					width++
				}
				in = on
			}
			if runs != 1 || width != int(scale) {
				t.Fatalf("scale, text scale %v: %d runs of the border's color (%v), %d device pixels, between the field and the arrows %v: want one of %v", sc, runs, line, width, s, scale)
			}
			// No notch: the field just left of the line, the line and the
			// arrows just right of it span the same device rows, which start
			// and end with the border.
			xs := []int{at - 1, at, at + width}
			var rows [3][2]int
			for i, x := range xs {
				rows[i][0], rows[i][1] = boxRows(img, x, bg)
			}
			if rows[0] != rows[1] || rows[1] != rows[2] {
				t.Fatalf("scale, text scale %v: rows at x=%v (line at %d): %v, want the same (stepper %v)", sc, xs, at, rows, s)
			}
			for _, x := range xs {
				for _, y := range rows[0] {
					if c := img.RGBAAt(x, y); c != line {
						t.Fatalf("scale, text scale %v: the joint's row %d at x=%d (line at %d) is %v, want the border %v (rows %v)", sc, y, x, at, c, line, rows)
					}
				}
			}
		}
	})
}

func TestDomainUpDownStillSteps(t *testing.T) {
	updownLooks(t, func(t *testing.T, dark bool) {
		d := DomainUpDown("North", "South")
		tt := NewTester(t, d, 400, 200)
		s := tt.Find(ByRole(ui.RoleStepper)).Bounds()
		tt.ClickAt(s.X+s.W/2, s.Y+22)
		tt.Settle()
		if d.SelectedItem() != "South" {
			t.Fatalf("the down arrow: %q, want South", d.SelectedItem())
		}
		tt.ClickAt(s.X+s.W/2, s.Y+8)
		tt.Settle()
		if d.SelectedItem() != "North" {
			t.Fatalf("the up arrow: %q, want North", d.SelectedItem())
		}
	})
}

// The arrows reach the stepper's top and bottom: hovering just inside either
// edge lights an arrow.
func TestDomainUpDownArrowsFillTheirColumn(t *testing.T) {
	updownLooks(t, func(t *testing.T, dark bool) {
		tt := NewTester(t, DomainUpDown("North", "South"), 400, 200)
		tt.SetDark(dark)
		tt.Frame()
		s := tt.Find(ByRole(ui.RoleStepper)).Bounds()
		x := s.X + s.W/2 - 6 // clear of the glyph
		// Count the rows inside the border that light under the pointer.
		tt.Move(x, 150)
		tt.Settle()
		rest := map[int]color.RGBA{}
		for y := int(s.Y) + 1; y < int(s.Y+s.H)-1; y++ {
			rest[y] = tt.Image().RGBAAt(int(x), y)
		}
		lit := 0
		for y := range rest {
			tt.Move(x, float32(y)+0.5)
			tt.Settle()
			if tt.Image().RGBAAt(int(x), y) != rest[y] {
				lit++
			}
		}
		if inner := len(rest); lit < inner-1 {
			t.Fatalf("arrows cover %d of the stepper's %d inner rows (%v): an empty band", lit, inner, s)
		}
	})
}

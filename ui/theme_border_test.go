package ui

import (
	"math"
	"testing"
)

// wcagContrast returns WCAG's contrast ratio of two opaque colors, from 1 to
// 21.
func wcagContrast(a, b Color) float64 {
	lum := func(c Color) float64 {
		ch := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
	}
	l1, l2 := lum(a), lum(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// SetControlBorder gives the default themes of both appearances the colors of
// the boundary of the controls, for apps with no Context of their own to set a
// Theme with, and a change of appearance keeps them.
func TestSetControlBorder(t *testing.T) {
	var got Color
	view := func(c *context) { got = c.Theme().ControlBorder }
	light, dark := Hex("#9ca3af"), Hex("#6b7280")

	tt := coreNewTester(view, 100, 100)
	if got != LightTheme().ControlBorder {
		t.Fatalf("before the call, the light control border is %v, want the theme's %v", got, LightTheme().ControlBorder)
	}

	SetControlBorder(light, dark)
	t.Cleanup(func() { SetControlBorder(Color{}, Color{}) })

	tt.Frame()
	if got != light {
		t.Errorf("a window open when it was called shows the light border %v, want %v", got, light)
	}
	tt.SetDark(true)
	if got != dark {
		t.Errorf("after the appearance went dark, the border is %v, want %v", got, dark)
	}
	tt.SetDark(false)
	if got != light {
		t.Errorf("after it came back, the border is %v, want %v", got, light)
	}

	other := coreNewTester(view, 100, 100)
	other.SetDark(true)
	if got != dark {
		t.Errorf("a window opened later, dark, shows %v, want %v", got, dark)
	}

	// A Theme the view sets itself is its own: the call changes the defaults only.
	own := LightTheme()
	coreNewTester(func(c *context) {
		c.SetTheme(own)
		got = c.Theme().ControlBorder
	}, 100, 100)
	if got != own.ControlBorder {
		t.Errorf("a theme set by the view has the control border %v, want its own %v", got, own.ControlBorder)
	}

	// A zero color leaves a theme's own, in each appearance.
	SetControlBorder(Color{}, dark)
	coreNewTester(view, 100, 100)
	if got != LightTheme().ControlBorder {
		t.Errorf("with no light color, the light control border is %v, want the theme's %v", got, LightTheme().ControlBorder)
	}

	// Border, the decoration's, is not what it sets.
	var border Color
	coreNewTester(func(c *context) { border = c.Theme().Border }, 100, 100)
	if border != LightTheme().Border {
		t.Errorf("the border of decoration changed to %v", border)
	}
}

// The boundary of a control is what identifies it (WCAG 1.4.11 asks 3:1
// against what is next to it): the default ControlBorder is, against the
// window and against the face of a control, in both appearances; Border, which
// draws dividers and grid lines, stays the light line it was.
func TestControlBorderPassesNonTextContrast(t *testing.T) {
	for _, c := range []struct {
		name   string
		th     *Theme
		border Color
	}{
		{"light", LightTheme(), Hex("#d9d9de")},
		{"dark", DarkTheme(), Hex("#3f3f46")},
	} {
		if c.th.Border != c.border {
			t.Errorf("%s: Border is %v, want %v: it draws decoration and is unchanged", c.name, c.th.Border, c.border)
		}
		for _, bg := range []struct {
			name string
			col  Color
		}{{"Background", c.th.Background}, {"Surface", c.th.Surface}} {
			if r := wcagContrast(c.th.ControlBorder, bg.col); r < 3 {
				t.Errorf("%s: ControlBorder %v on %s %v is %.2f:1, under 3:1", c.name, c.th.ControlBorder, bg.name, bg.col, r)
			}
		}
	}
}

// Outline is ControlBorder, or Border in a Theme that sets none, as an app's
// own Theme written before ControlBorder did.
func TestThemeOutlineFallsBackToBorder(t *testing.T) {
	own := &Theme{Border: Hex("#123456")}
	if got := own.Outline(); got != own.Border {
		t.Errorf("Outline of a theme without ControlBorder is %v, want Border %v", got, own.Border)
	}
	own.ControlBorder = Hex("#abcdef")
	if got := own.Outline(); got != own.ControlBorder {
		t.Errorf("Outline is %v, want ControlBorder %v", got, own.ControlBorder)
	}
}

// What the user takes for a control has the ControlBorder around it, and
// what only divides, or holds, has Border.
func TestControlsAreOutlinedByControlBorderAndDecorationByBorder(t *testing.T) {
	pixel := func(tt *Tester, n TestNode) Color {
		b := tt.Image().RGBAAt(int(n.Bounds.X), int(n.Bounds.Y+n.Bounds.H/2))
		return Color{R: b.R, G: b.G, B: b.B, A: b.A}
	}
	same := func(a, b Color) bool { return a.R == b.R && a.G == b.G && a.B == b.B }
	for _, dark := range []bool{false, true} {
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		var text string
		var sel string
		var num float64
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreColumn(c).Gap(10).Padding(20).Children(func() {
				coreTextInput(c, &text).Width(200).Label("Input")
				coreSelect(c, &sel, []string{"a", "b", "c"}).Width(200).Label("Select")
				coreNumberInput(c, &num, 0, 10, 1).Width(200).Label("Number")
				coreButton(c, "Button")
				coreDivider(c).Height(3).Label("Divider")
			})
		}, 400, 300)
		tt.Frame()
		seen := map[string]bool{}
		for _, n := range tt.Nodes() {
			if n.Label == "Divider" {
				// Decoration keeps the lighter Border.
				c := tt.Image().RGBAAt(int(n.Bounds.X+n.Bounds.W/2), int(n.Bounds.Y+n.Bounds.H/2))
				if !same(Color{R: c.R, G: c.G, B: c.B}, th.Border) {
					t.Errorf("dark=%v: the divider is %v, want Border %v", dark, c, th.Border)
				}
				seen[n.Label] = true
				continue
			}
			if n.Label != "Input" && n.Label != "Select" && n.Label != "Number" && n.Label != "Button" {
				continue
			}
			if n.Bounds.W == 0 || n.Role == RoleText {
				continue
			}
			seen[n.Label] = true
			found := false
			img := tt.Image()
			y := int(n.Bounds.Y + n.Bounds.H/2)
			for x := int(n.Bounds.X) - 1; x <= int(n.Bounds.X)+2; x++ {
				c := img.RGBAAt(x, y)
				if same(Color{R: c.R, G: c.G, B: c.B}, th.ControlBorder) {
					found = true
				}
			}
			if !found {
				t.Errorf("dark=%v: %q (%v) is not outlined by ControlBorder %v: pixel %v", dark, n.Label, n.Bounds, th.ControlBorder, pixel(tt, n))
			}
		}
		if len(seen) != 5 {
			t.Errorf("dark=%v: the test found %v of the Input, Select, Number, Button and Divider", dark, seen)
		}
	}
}

// The rows a list keeps (RowVersion) were built with the theme the pass lent
// them: a change of the default control border builds them again (themeGen), so
// that the outline of a button in one shows the new color.
func TestSetControlBorderRebuildsKeptRows(t *testing.T) {
	s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
	tt := coreNewTester(func(c *context) {
		coreList(c, s, 6, func(i int) { coreButton(c, "Go") }).Grow(1)
	}, 300, 300)
	tt.Frame()
	tt.Frame()
	if tt.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test shows nothing")
	}
	outlined := func(want Color) bool {
		for _, n := range tt.Nodes() {
			if n.Role != RoleButton {
				continue
			}
			img := tt.Image()
			y := int(n.Bounds.Y + n.Bounds.H/2)
			for x := int(n.Bounds.X) - 1; x <= int(n.Bounds.X)+2; x++ {
				if c := img.RGBAAt(x, y); c.R == want.R && c.G == want.G && c.B == want.B {
					return true
				}
			}
			return false
		}
		t.Fatal("no button")
		return false
	}
	if !outlined(LightTheme().ControlBorder) {
		t.Fatal("before the call, a button is not outlined by the theme's ControlBorder")
	}
	red := Hex("#ff0000")
	SetControlBorder(red, red)
	t.Cleanup(func() { SetControlBorder(Color{}, Color{}) })
	tt.Frame()
	if !outlined(red) {
		t.Error("a button in a kept row keeps its old outline after the default control border changed")
	}
}

// A high contrast setting does not take the outline of a control under what
// it was.
func TestHighContrastKeepsControlBorderStrong(t *testing.T) {
	for _, dark := range []bool{false, true} {
		var got, bg Color
		tt := coreNewTester(func(c *context) { got, bg = c.Theme().ControlBorder, c.Theme().Background }, 100, 100)
		tt.SetDark(dark)
		normal := wcagContrast(got, bg)
		tt.SetPreferences(Preferences{HighContrast: true})
		if r := wcagContrast(got, bg); r < normal {
			t.Errorf("dark=%v: high contrast gives the control border %.2f:1, under the normal %.2f:1", dark, r, normal)
		}
	}
}

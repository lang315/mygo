package ui

import (
	"math"
	"testing"
	"time"
)

// diff returns how far apart two color channels are.
func diff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

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

// outlineScene builds each control of the review's list in a column, under th,
// and returns the Tester and the nodes by label.
type outlineControl struct {
	label string
	build func(c *context, label string)
}

func outlineControls() []outlineControl {
	var (
		text, area, sel, combo, auto string
		num, step                    float64
		segment                      int
		when                         time.Time
		pinned, starred              bool
		buffer                       = NewTextBuffer("")
	)
	return []outlineControl{
		{"Input", func(c *context, l string) { coreTextInput(c, &text).Width(200).Label(l) }},
		{"Area", func(c *context, l string) { coreTextArea(c, &area).Width(200).Label(l) }},
		{"Select", func(c *context, l string) { coreSelect(c, &sel, []string{"a", "b", "c"}).Width(200).Label(l) }},
		{"Combobox", func(c *context, l string) { coreCombobox(c, &combo, []string{"a", "b", "c"}).Width(200).Label(l) }},
		{"Number", func(c *context, l string) { coreNumberInput(c, &num, 0, 10, 1).Width(200).Label(l) }},
		{"Time", func(c *context, l string) { coreTimeInput(c, &when).Label(l) }},
		{"Stepper", func(c *context, l string) { coreStepper(c, &step, 0, 10, 1).Label(l) }},
		{"Segmented", func(c *context, l string) { coreSegmented(c, &segment, "One", "Two").Label(l) }},
		{"Button", func(c *context, l string) { coreButton(c, "Button").Label(l) }},
		{"Autocomplete", func(c *context, l string) { coreAutocomplete(c, &auto, []string{"a", "b"}).Width(200).Label(l) }},
		{"AreaBuffer", func(c *context, l string) { coreTextAreaBuffer(c, buffer).Width(200).Label(l) }},
		{"ToggleGroup", func(c *context, l string) {
			coreToggleGroup(c, func() {
				coreToggle(c, &pinned, "Pin")
				coreToggle(c, &starred, "Star")
			}).Label(l)
		}},
	}
}

// What the user takes for a control has the ControlBorder around it, and what
// only divides, or holds, has Border: each control of the list, in both
// appearances, and a divider.
func TestControlsAreOutlinedByControlBorderAndDecorationByBorder(t *testing.T) {
	same := func(a, b Color) bool { return a.R == b.R && a.G == b.G && a.B == b.B }
	for _, dark := range []bool{false, true} {
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		controls := outlineControls()
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreColumn(c).Gap(10).Padding(20).Children(func() {
				for _, k := range controls {
					k.build(c, k.label)
				}
				coreDivider(c).Height(3).Label("Divider")
			})
		}, 400, 1100)
		tt.Frame()
		img := tt.Image()
		seen := map[string]bool{}
		for _, n := range tt.Nodes() {
			if n.Label == "Divider" {
				c := img.RGBAAt(int(n.Bounds.X+n.Bounds.W/2), int(n.Bounds.Y+n.Bounds.H/2))
				if !same(Color{R: c.R, G: c.G, B: c.B}, th.Border) {
					t.Errorf("dark=%v: the divider is %v, want Border %v", dark, c, th.Border)
				}
				seen[n.Label] = true
				continue
			}
			isControl := false
			for _, k := range controls {
				isControl = isControl || k.label == n.Label
			}
			if !isControl || n.Bounds.W == 0 || n.Role == RoleText {
				continue
			}
			seen[n.Label] = true
			// The edge of the box: a column of pixels just inside or outside it,
			// a row at its middle (the stepper and the segmented control are
			// narrow, so the left edge serves for all).
			found := false
			y := int(n.Bounds.Y + n.Bounds.H/2)
			for x := int(n.Bounds.X) - 16; x <= int(n.Bounds.X)+2; x++ { // a field is labelled by its input, inside its padding
				c := img.RGBAAt(x, y)
				if same(Color{R: c.R, G: c.G, B: c.B}, th.ControlBorder) {
					found = true
				}
			}
			if !found {
				c := img.RGBAAt(int(n.Bounds.X), y)
				t.Errorf("dark=%v: %q (%v) is not outlined by ControlBorder %v: pixel %v", dark, n.Label, n.Bounds, th.ControlBorder, c)
			}
		}
		if len(seen) != len(controls)+1 {
			t.Errorf("dark=%v: the test found %v of the %d controls and the divider", dark, seen, len(controls))
		}
	}
}

// The boundary of a check box, a radio button and the off track of a switch
// identify the control as the outline of an input does: against the window
// they are 3:1 or more, in both appearances, and are the ControlBorder.
func TestToggleControlsReachNonTextContrast(t *testing.T) {
	for _, dark := range []bool{false, true} {
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		var on bool
		var choice string
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreColumn(c).Gap(14).Padding(20).Children(func() {
				coreCheckbox(c, &on, "Check").Label("Check")
				coreRadio(c, &choice, "a", "Radio").Label("Radio")
				coreSwitch(c, &on).Label("Switch")
			})
		}, 300, 200)
		tt.Frame()
		img := tt.Image()
		for _, n := range tt.Nodes() {
			var edge bool
			switch n.Label {
			case "Check", "Radio":
				edge = true
			case "Switch":
			default:
				continue
			}
			y := int(n.Bounds.Y + n.Bounds.H/2)
			var got Color
			ok := false
			if edge {
				// The left edge of the box: the pixel nearest the ControlBorder
				// there (the dot of a radio button is round, and anti-aliased
				// where its outline is leftmost).
				near := 1 << 30
				for x := int(n.Bounds.X) - 1; x <= int(n.Bounds.X)+2; x++ {
					c := img.RGBAAt(x, y)
					d := diff(c.R, th.ControlBorder.R) + diff(c.G, th.ControlBorder.G) + diff(c.B, th.ControlBorder.B)
					if d < near {
						near, got = d, Color{R: c.R, G: c.G, B: c.B}
					}
				}
				ok = near <= 30
			} else { // the off track, past the knob at its left
				c := img.RGBAAt(int(n.Bounds.X+n.Bounds.W)-6, y)
				got = Color{R: c.R, G: c.G, B: c.B}
				ok = got.R == th.ControlBorder.R && got.G == th.ControlBorder.G && got.B == th.ControlBorder.B
			}
			if !ok {
				t.Errorf("dark=%v: %q is %v, want ControlBorder %v", dark, n.Label, got, th.ControlBorder)
			}
			if r := wcagContrast(got, th.Background); r < 3 {
				t.Errorf("dark=%v: %q is %.2f:1 against the window, under 3:1", dark, n.Label, r)
			}
		}
	}
}

// The outline of a text input does not get lighter as the pointer comes over
// it: hovered, it is as strong against the window as at rest.
func TestHoveredInputOutlineIsNotLighterThanAtRest(t *testing.T) {
	for _, dark := range []bool{false, true} {
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		var text string
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreColumn(c).Padding(20).Children(func() { coreTextInput(c, &text).Width(200).Label("Input") })
		}, 300, 100)
		tt.Frame()
		edge := func() Color {
			for _, n := range tt.Nodes() {
				if n.Label == "Input" && n.Role == RoleTextField {
					c := tt.Image().RGBAAt(int(n.Bounds.X), int(n.Bounds.Y+n.Bounds.H/2))
					return Color{R: c.R, G: c.G, B: c.B}
				}
			}
			t.Fatal("no input")
			return Color{}
		}
		rest := edge()
		in, _ := tt.Find("Input")
		_ = in
		for _, n := range tt.Nodes() {
			if n.Label == "Input" {
				tt.Move(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
				break
			}
		}
		tt.Frame()
		hover := edge()
		if wcagContrast(hover, th.Background) < wcagContrast(rest, th.Background) {
			t.Errorf("dark=%v: hovered, the outline %v is %.2f:1 against the window, under %v at rest, %.2f:1", dark, hover,
				wcagContrast(hover, th.Background), rest, wcagContrast(rest, th.Background))
		}
		if hover == rest {
			t.Errorf("dark=%v: the outline does not change as the pointer comes over it: the test shows nothing", dark)
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

// A high contrast setting makes the outline of a control stronger than it was.
func TestHighContrastKeepsControlBorderStrong(t *testing.T) {
	for _, dark := range []bool{false, true} {
		var got, bg Color
		tt := coreNewTester(func(c *context) { got, bg = c.Theme().ControlBorder, c.Theme().Background }, 100, 100)
		tt.SetDark(dark)
		normal := wcagContrast(got, bg)
		tt.SetPreferences(Preferences{HighContrast: true})
		if r := wcagContrast(got, bg); r <= normal {
			t.Errorf("dark=%v: high contrast gives the control border %.2f:1, not above the normal %.2f:1", dark, r, normal)
		}
	}
}

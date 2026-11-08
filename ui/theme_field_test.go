package ui

import (
	"testing"
	"time"
)

// FieldFill is Field, or Surface in a Theme that sets none, so that a Theme
// written before Field has the look it had.
func TestThemeFieldFillFallsBackToSurface(t *testing.T) {
	own := &Theme{Surface: Hex("#123456")}
	if got := own.FieldFill(); got != own.Surface {
		t.Errorf("FieldFill of a theme without Field is %v, want Surface %v", got, own.Surface)
	}
	own.Field = Hex("#abcdef")
	if got := own.FieldFill(); got != own.Field {
		t.Errorf("FieldFill is %v, want Field %v", got, own.Field)
	}
}

// A field is not the face of a button, in both appearances, and what draws it
// stands out: the border against the field, the focus ring against the window.
func TestFieldIsNotTheFaceOfAButton(t *testing.T) {
	for _, c := range []struct {
		name string
		th   *Theme
	}{{"light", LightTheme()}, {"dark", DarkTheme()}} {
		th := c.th
		if th.FieldFill() == th.Surface {
			t.Errorf("%s: the fill of a field is the face of a button, %v", c.name, th.Surface)
		}
		if th.FieldFill() == th.SurfaceHover || th.FieldFill() == th.SurfacePressed {
			t.Errorf("%s: the fill of a field is a button's hover or pressed face", c.name)
		}
		if r := wcagContrast(th.ControlBorder, th.FieldFill()); r < 3 {
			t.Errorf("%s: ControlBorder on Field is %.2f:1, under 3:1", c.name, r)
		}
		if r := wcagContrast(th.Accent, th.FieldFill()); r < 3 {
			t.Errorf("%s: the focused border, Accent, on Field is %.2f:1, under 3:1", c.name, r)
		}
		if r := wcagContrast(th.Text, th.FieldFill()); r < 7 {
			t.Errorf("%s: Text on Field is %.2f:1, under 7:1", c.name, r)
		}
		ring := th.Focus.Over(th.Background)
		if r := wcagContrast(ring, th.Background); r < 3 {
			t.Errorf("%s: the focus ring %v on Background is %.2f:1, under 3:1", c.name, ring, r)
		}
	}
}

// Each control in its role, drawn: inputs have the Field's fill and the
// buttons and what looks like them the Surface's, in both appearances.
func TestFieldFillsOfInputsAndSurfacesOfButtons(t *testing.T) {
	var (
		text, area, sel, combo, auto string
		num, step                    float64
		when                         time.Time
		date                         = time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
		pinned, starred              bool
		buffer                       = NewTextBuffer("")
	)
	type control struct {
		label string
		field bool
		build func(c *context, l string)
	}
	controls := []control{
		{"Input", true, func(c *context, l string) { coreTextInput(c, &text).Width(200).Label(l) }},
		{"Area", true, func(c *context, l string) { coreTextArea(c, &area).Width(200).Label(l) }},
		{"Combobox", true, func(c *context, l string) { coreCombobox(c, &combo, []string{"a", "b"}).Width(200).Label(l) }},
		{"Autocomplete", true, func(c *context, l string) { coreAutocomplete(c, &auto, []string{"a", "b"}).Width(200).Label(l) }},
		{"Number", true, func(c *context, l string) { coreNumberInput(c, &num, 0, 10, 1).Width(200).Label(l) }},
		{"Time", true, func(c *context, l string) { coreTimeInput(c, &when).Label(l) }},
		{"Date", true, func(c *context, l string) { coreDateInput(c, &date).Label(l) }},
		{"AreaBuffer", true, func(c *context, l string) { coreTextAreaBuffer(c, buffer).Width(200).Label(l) }},
		{"Select", false, func(c *context, l string) { coreSelect(c, &sel, []string{"a", "b"}).Width(200).Label(l) }},
		{"Stepper", false, func(c *context, l string) { coreStepper(c, &step, 0, 10, 1).Label(l) }},
		{"Button", false, func(c *context, l string) { coreButton(c, "Button").Label(l) }},
		{"ToggleGroup", false, func(c *context, l string) {
			coreToggleGroup(c, func() {
				coreToggle(c, &pinned, "Pin")
				coreToggle(c, &starred, "Star")
			}).Label(l)
		}},
	}
	for _, dark := range []bool{false, true} {
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreColumn(c).Gap(10).Padding(20).Children(func() {
				for _, k := range controls {
					k.build(c, k.label)
				}
			})
		}, 400, 1300)
		tt.Frame()
		img := tt.Image()
		rgb := func(x, y int) Color {
			c := img.RGBAAt(x, y)
			return Color{R: c.R, G: c.G, B: c.B}
		}
		seen := map[string]bool{}
		for _, n := range tt.Nodes() {
			var k *control
			for i := range controls {
				if controls[i].label == n.Label {
					k = &controls[i]
				}
			}
			if k == nil || n.Bounds.W == 0 || n.Role == RoleText {
				continue
			}
			y := int(n.Bounds.Y + n.Bounds.H/2)
			edge := -1
			for x := int(n.Bounds.X) - 16; x <= int(n.Bounds.X)+2; x++ { // a field is labelled by its input, inside its padding
				if got := rgb(x, y); got.R == th.ControlBorder.R && got.G == th.ControlBorder.G && got.B == th.ControlBorder.B {
					edge = x
					break
				}
			}
			if edge < 0 {
				t.Errorf("dark=%v: %q has no ControlBorder edge", dark, k.label)
				continue
			}
			seen[k.label] = true
			want := th.Surface
			if k.field {
				want = th.FieldFill()
			}
			got := rgb(edge+2, y)
			if got != (Color{R: want.R, G: want.G, B: want.B}) {
				t.Errorf("dark=%v: %q is filled %v, want %v", dark, k.label, got, want)
			}
		}
		if len(seen) != len(controls) {
			t.Errorf("dark=%v: found %d of the %d controls: %v", dark, len(seen), len(controls), seen)
		}
	}
}

// A Theme whose Field is zero draws its fields with its Surface, as a Theme
// written before Field did.
func TestFieldZeroKeepsTheLookOfBefore(t *testing.T) {
	th := LightTheme()
	th.Field = Color{}
	th.Surface = Hex("#e0f0e0")
	var text string
	tt := coreNewTester(func(c *context) {
		c.SetTheme(th)
		coreColumn(c).Padding(20).Children(func() { coreTextInput(c, &text).Width(200).Label("Input") })
	}, 300, 120)
	tt.Frame()
	img := tt.Image()
	for _, n := range tt.Nodes() {
		if n.Label != "Input" || n.Bounds.W == 0 || n.Role == RoleText {
			continue
		}
		y := int(n.Bounds.Y + n.Bounds.H/2)
		for x := int(n.Bounds.X) - 16; x <= int(n.Bounds.X)+2; x++ {
			if c := img.RGBAAt(x, y); (Color{R: c.R, G: c.G, B: c.B}) == (Color{R: th.ControlBorder.R, G: th.ControlBorder.G, B: th.ControlBorder.B}) {
				if f := img.RGBAAt(x+2, y); (Color{R: f.R, G: f.G, B: f.B}) != (Color{R: 0xe0, G: 0xf0, B: 0xe0}) {
					t.Errorf("a field of a Theme without Field is filled %v, want its Surface", f)
				}
				return
			}
		}
		t.Fatal("no outline found")
	}
	t.Fatal("no Input node")
}

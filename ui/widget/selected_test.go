package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

var selGreen = ui.Hex("#00ff00")

// selectedStyle sets a green background while the control is selected,
// padded so the fill shows around its content.
func selectedStyle() Style {
	return NewStyle().Padding(3).Selected(NewStyle().Background(Hex("#00ff00")))
}

// pixelsOf is the number of pixels of tt within 8 of c on each channel.
func pixelsOf(tt *ui.Tester, c ui.Color) int {
	img := tt.Image()
	n := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if near(img.RGBAAt(x, y), c) {
				n++
			}
		}
	}
	return n
}

// paints says whether the fill c covers a control's padding: more pixels
// than a stray one.
func paints(tt *ui.Tester, c ui.Color) bool { return pixelsOf(tt, c) >= 40 }

// CheckBox, RadioButton, Toggle and a toggle ToolButton show Selected while
// checked or on, and not before or after.
func TestSelectedVariantControls(t *testing.T) {
	type ctl struct {
		name string
		w    Control
		set  func(bool)
	}
	cb := CheckBox("check").Apply(selectedStyle())
	rb := RadioButton("radio").Apply(selectedStyle())
	tg := Toggle("toggle").Apply(selectedStyle())
	tb := ToolButton("tool").SetToggle(true).Apply(selectedStyle())
	for _, c := range []ctl{
		{"CheckBox", cb, func(v bool) { cb.SetChecked(v) }},
		{"RadioButton", rb, func(v bool) { rb.SetChecked(v) }},
		{"Toggle", tg, func(v bool) { tg.SetActive(v) }},
		{"ToolButton", tb, func(v bool) { tb.SetChecked(v) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			tt := tester(t, c.w)
			if paints(tt, selGreen) {
				t.Errorf("unchecked paints the Selected green")
			}
			c.set(true)
			tt.Frame()
			if !paints(tt, selGreen) {
				t.Errorf("checked paints %d green pixels, want the Selected background", pixelsOf(tt, selGreen))
			}
			c.set(false)
			tt.Frame()
			if paints(tt, selGreen) {
				t.Errorf("unchecked again paints the Selected green")
			}
		})
	}
}

// A Button, pressed, and a TextBox, focused, are never selected, so Selected
// never shows.
func TestSelectedNeverOnPlainControls(t *testing.T) {
	b := Button("button").Apply(selectedStyle())
	tt := tester(t, b)
	x, y := centerOf(t, tt, "button")
	tt.Move(x, y)
	tt.Press(x, y)
	tt.Frame()
	if n := pixelsOf(tt, selGreen); n > 0 {
		t.Errorf("a pressed Button paints %d pixels of the Selected green", n)
	}
	tt.Release(x, y)

	tb := TextBox().SetText("text").Apply(selectedStyle())
	tt = tester(t, tb)
	tb.Focus()
	tt.Frame()
	tt.Frame()
	if n := pixelsOf(tt, selGreen); n > 0 {
		t.Errorf("a focused TextBox paints %d pixels of the Selected green", n)
	}
}

// A checked tool button made a plain one again is no longer selected.
func TestSelectedLeavesAToolButtonNoLongerAToggle(t *testing.T) {
	tb := ToolButton("tool").SetToggle(true).SetChecked(true).Apply(selectedStyle())
	tt := tester(t, tb)
	if !paints(tt, selGreen) {
		t.Fatalf("a checked toggle paints %d green pixels, want the Selected background", pixelsOf(tt, selGreen))
	}
	tb.SetToggle(false)
	tt.Frame()
	if n := pixelsOf(tt, selGreen); n > 0 {
		t.Errorf("no longer a toggle, it paints %d pixels of the Selected green", n)
	}
}

// Selected wins over Hover; Pressed and Disabled win over Selected.
func TestSelectedOrder(t *testing.T) {
	red, blue, gray := ui.Hex("#ff0000"), ui.Hex("#0000ff"), ui.Hex("#888888")
	style := NewStyle().Padding(3).
		Hover(NewStyle().Background(Fixed(red))).
		Selected(NewStyle().Background(Fixed(selGreen))).
		Pressed(NewStyle().Background(Fixed(blue))).
		Disabled(NewStyle().Background(Fixed(gray)))

	cb := CheckBox("order").SetChecked(true).Apply(style)
	tt := tester(t, VBox(0).Add(cb, Label("away")))
	x, y := centerOf(t, tt, "order")
	tt.Move(x, y)
	tt.Frame()
	if !paints(tt, selGreen) || pixelsOf(tt, red) > 0 {
		t.Errorf("checked and hovered: %d green and %d red pixels, want the Selected green only", pixelsOf(tt, selGreen), pixelsOf(tt, red))
	}
	tt.Press(x, y)
	tt.Frame()
	if !paints(tt, blue) || pixelsOf(tt, selGreen) > 0 {
		t.Errorf("checked and pressed: %d blue and %d green pixels, want the Pressed blue only", pixelsOf(tt, blue), pixelsOf(tt, selGreen))
	}

	dis := CheckBox("off").SetChecked(true).Apply(style)
	tt = tester(t, VBox(0).Add(dis, Label("away")))
	dis.SetDisabled(true)
	tt.Frame()
	// ui draws a disabled element at 50% opacity, over the page.
	want := mix(gray, page(tt), 0.5)
	if !paints(tt, want) || pixelsOf(tt, mix(selGreen, page(tt), 0.5)) > 0 {
		t.Errorf("checked and disabled: %d pixels of %v and %d of the faded green, want the Disabled gray only",
			pixelsOf(tt, want), want, pixelsOf(tt, mix(selGreen, page(tt), 0.5)))
	}
	if c := tt.Image().RGBAAt(int(find(t, tt, "off").X-2), int(centerY(t, tt, "off"))); !near(c, want) {
		t.Errorf("the padding left of the label is %v, want %v", c, want)
	}
}

func centerY(t *testing.T, tt *ui.Tester, s string) float32 {
	_, y := centerOf(t, tt, s)
	return y
}

// A Selected variant's Dark applies in the dark theme.
func TestSelectedDark(t *testing.T) {
	x, y := ui.Hex("#ff00ff"), ui.Hex("#00ffff")
	cb := CheckBox("dark").SetChecked(true).Apply(NewStyle().Padding(3).
		Selected(NewStyle().Background(Fixed(x)).Dark(NewStyle().Background(Fixed(y)))))
	tt := tester(t, cb)
	tt.SetDark(true)
	tt.Frame()
	if !paints(tt, y) || pixelsOf(tt, x) > 0 {
		t.Errorf("dark and checked: %d pixels of the dark %v and %d of the light %v", pixelsOf(tt, y), y, pixelsOf(tt, x), x)
	}
}

// The inspector names the Selected variant, in the state and in the layer a
// value came from.
func TestInspectorNamesSelected(t *testing.T) {
	layers := []styleLayer{{"Own methods", NewStyle().Selected(NewStyle().Background(Fixed(selGreen)))}}
	fs := inEffect(ui.LightTheme(), layers, frameState{selected: true})
	if len(fs) < 2 || fs[0].Name != "In effect (selected)" || fs[1].Value != "#00ff00   selected: Own methods" {
		t.Fatalf("fields %+v", fs)
	}
}

// The inspector shows a checked control's state as selected.
func TestInspectorSeesSelected(t *testing.T) {
	cb := CheckBox("seen").SetChecked(true).Apply(selectedStyle())
	tt := NewTester(t, cb, 600, 400)
	if err := tt.Inspect("seen"); err != nil {
		t.Fatal(err)
	}
	tt.Click("Widget")
	if !slices.Contains(tt.Texts(), "In effect (selected)") {
		t.Fatalf("the inspector shows %q, no \"In effect (selected)\"", tt.Texts())
	}
}

// The inspector names the roles a skin decides.
func TestInspectorNamesSkinRoles(t *testing.T) {
	for c, want := range map[Color]string{
		RoleChrome: "role chrome", RoleBorderStrong: "role border-strong", RoleControlBorder: "role control-border",
		RoleButtonBorder: "role button-border", RoleRowHover: "role row-hover", RoleRowSelected: "role row-selected",
		RoleInfo: "role info",
	} {
		if got := c.inspectText(); got != want {
			t.Errorf("%+v reads %q, want %q", c, got, want)
		}
	}
}

// effective allocates nothing, every state on.
func TestEffectiveAllocs(t *testing.T) {
	v := NewStyle().Background(Fixed(selGreen)).Dark(NewStyle().Padding(1))
	s := NewStyle().Padding(2).Dark(v).Hover(v).Selected(v).Focused(v).Pressed(v).Disabled(v)
	var p props
	if n := testing.AllocsPerRun(100, func() { p = s.effective(true, true, true, true, true, true) }); n != 0 {
		t.Fatalf("effective allocates %v times", n)
	}
	if p.background.fixed != selGreen {
		t.Fatalf("background %v", p.background)
	}
}

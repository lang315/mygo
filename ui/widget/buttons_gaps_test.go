package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The last look a button is given wins, hover and pressed included: a
// Destructive after a Suggested looks as Destructive alone, and the other
// way round, as Apply replaces the styles given before.
func TestButtonLastLookWins(t *testing.T) {
	for _, dark := range []bool{false, true} {
		got := buttonLook(t, Button("Delete").Suggested().Destructive(), "Delete", dark)
		want := buttonLook(t, Button("Delete").Destructive(), "Delete", dark)
		if got != want {
			t.Errorf("dark %v: Suggested().Destructive() %v, Destructive() %v", dark, got, want)
		}
		got = buttonLook(t, Button("Save").Destructive().Suggested(), "Save", dark)
		want = buttonLook(t, Button("Save").Suggested(), "Save", dark)
		if got != want {
			t.Errorf("dark %v: Destructive().Suggested() %v, Suggested() %v", dark, got, want)
		}
		got = buttonLook(t, inStrip(ToolButton("Delete").Suggested().Destructive()), "Delete", dark)
		want = buttonLook(t, inStrip(ToolButton("Delete").Destructive()), "Delete", dark)
		if got != want {
			t.Errorf("dark %v: tool button Suggested().Destructive() %v, Destructive() %v", dark, got, want)
		}
	}
}

// An icon-only button is named by its tooltip, so assistive technology and
// a Tester find it by name and not only by its place.
func TestIconButtonNamedByTooltip(t *testing.T) {
	tt := NewTester(t, VBox(0).Add(Button("").SetIcon(chevronDown).SetTooltip("Next")), 0, 0)
	n := tt.Find(ByRole(ui.RoleButton).Label("Next"))
	if n.Label() != "Next" {
		t.Fatalf("label %q, want Next", n.Label())
	}
}

// namedRow is a row of the lists and trees named in
// TestSetAccessibleNameOnEveryControl.
type namedRow struct{ id int }

// SetAccessibleName names the control itself, not a box around it, plain
// and in a Fields row, whose label it wins over; a group of controls, and a
// container, is named as a group. An icon-only button's name wins over its
// tooltip.
func TestSetAccessibleNameOnEveryControl(t *testing.T) {
	key := func(r *namedRow) int { return r.id }
	text := func(r *namedRow) string { return "x" }
	ab := []Option{{"a", "A"}, {"b", "B"}}
	for _, c := range []struct {
		name string
		role ui.Role
		make func() Control
	}{
		{"Button", ui.RoleButton, func() Control { return Button("Go").SetAccessibleName("ZZ") }},
		{"icon-only Button", ui.RoleButton, func() Control {
			return Button("").SetIcon(chevronDown).SetTooltip("Tip").SetAccessibleName("ZZ")
		}},
		{"ToolButton", ui.RoleButton, func() Control {
			return ToolStrip(ToolButton("").SetIcon(chevronDown).SetTooltip("Tip").SetAccessibleName("ZZ"))
		}},
		{"LinkLabel", ui.RoleLink, func() Control { return LinkLabel("Site", "https://example.com").SetAccessibleName("ZZ") }},
		{"Toggle", ui.RoleToggleButton, func() Control { return Toggle("T").SetAccessibleName("ZZ") }},
		{"CheckBox", ui.RoleCheckBox, func() Control { return CheckBox("C").SetAccessibleName("ZZ") }},
		{"Switch", ui.RoleSwitch, func() Control { return Switch().SetAccessibleName("ZZ") }},
		{"CheckBoxGroup", ui.RoleGroup, func() Control { return CheckBoxGroup(false, ab...).SetAccessibleName("ZZ") }},
		{"TagSelect", ui.RoleGroup, func() Control { return TagSelect(ab...).SetAccessibleName("ZZ") }},
		{"RadioGroup", ui.RoleRadioGroup, func() Control { return RadioGroup(false, ab...).SetAccessibleName("ZZ") }},
		{"RangeSpin", ui.RoleGroup, func() Control { return RangeSpin(0, 10, 0, 10, 1, "to", "").SetAccessibleName("ZZ") }},
		{"ComboBox", ui.RolePopUpButton, func() Control { return ComboBox(ab...).SetAccessibleName("ZZ") }},
		{"ProgressBar", ui.RoleProgress, func() Control { return ProgressBar().SetAccessibleName("ZZ") }},
		{"DateTimePicker", ui.RolePopUpButton, func() Control { return DateTimePicker().SetAccessibleName("ZZ") }},
		{"MonthCalendar", ui.RoleTable, func() Control { return MonthCalendar().SetAccessibleName("ZZ") }},
		{"DomainUpDown", ui.RoleStepper, func() Control { return DomainUpDown("a", "b").SetAccessibleName("ZZ") }},
		{"MaskedTextBox", ui.RoleTextField, func() Control { return MaskedTextBox("000").SetAccessibleName("ZZ") }},
		{"MenuButton", ui.RoleMenuButton, func() Control { return MenuButton("M").SetAccessibleName("ZZ") }},
		{"TextBox", ui.RoleTextField, func() Control { return TextBox().SetAccessibleName("ZZ") }},
		{"MultilineTextBox", ui.RoleTextField, func() Control { return MultilineTextBox().SetAccessibleName("ZZ") }},
		{"TrackBar", ui.RoleSlider, func() Control { return TrackBar(0, 10).SetAccessibleName("ZZ") }},
		{"ListBox", ui.RoleList, func() Control { return ListBox(key, text).SetAccessibleName("ZZ") }},
		{"TreeView", ui.RoleTree, func() Control { return TreeView(key, text).SetAccessibleName("ZZ") }},
		{"VBox", ui.RoleGroup, func() Control { return VBox(0).Add(TextBox()).SetAccessibleName("ZZ") }},
	} {
		for _, inRow := range []bool{false, true} {
			w := c.make()
			if inRow {
				w = Fields().Row("Row", w)
			}
			tt := NewTester(t, VBox(0).Add(w), 0, 0)
			named := tt.FindAll(ByLabel("ZZ"))
			if len(named) != 1 || named[0].Role() != c.role {
				t.Errorf("%s (in a row: %v): named %v, want one %s", c.name, inRow, named, roleName(c.role))
				continue
			}
			if inRow && tt.Has(ByRole(c.role).Label("Row")) {
				t.Errorf("%s: the row's label names it over its accessible name", c.name)
			}
			if tt.Has(ByLabel("Tip")) {
				t.Errorf("%s: the tooltip names it over its accessible name", c.name)
			}
		}
	}
}

// A button with no text is named by its tooltip even when its PNG icon is
// no image, so it still has a name.
func TestButtonWithBadPNGNamedByTooltip(t *testing.T) {
	junk := []byte("no image")
	tt := NewTester(t, VBox(0).Add(
		Button("").SetIconPNG(junk).SetTooltip("Paint"),
		ToolStrip(ToolButton("").SetIconPNG(junk).SetTooltip("Erase")),
	), 0, 0)
	tt.Find(ByRole(ui.RoleButton).Label("Paint"))
	tt.Find(ByRole(ui.RoleButton).Label("Erase"))
}

// twoColorPNG is a side×side PNG, red on its left half and blue on its right.
func twoColorPNG(t *testing.T, side int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := range side {
		for x := range side {
			c := color.RGBA{R: 255, A: 255}
			if x >= side/2 {
				c = color.RGBA{B: 255, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// A PNG icon draws in its own colors, not the text's, at 1× and 2×.
func TestButtonPNGIconDrawsItsColors(t *testing.T) {
	data := twoColorPNG(t, 32)
	for _, w := range []Control{
		Button("").SetIconPNG(data).SetTooltip("Paint"),
		ToolStrip(ToolButton("").SetIconPNG(data).SetTooltip("Paint")),
	} {
		for _, scale := range []float32{1, 2} {
			tt := NewTester(t, VBox(0).Add(w), 0, 0)
			tt.SetScale(scale)
			b := tt.Find(ByRole(ui.RoleButton).Label("Paint")).Bounds()
			cx, cy := b.X+b.W/2, b.Y+b.H/2
			img := tt.Image()
			left := img.RGBAAt(int((cx-3)*scale), int(cy*scale))
			right := img.RGBAAt(int((cx+3)*scale), int(cy*scale))
			if !near(left, ui.RGB(255, 0, 0)) || !near(right, ui.RGB(0, 0, 255)) {
				t.Errorf("scale %v: left of the icon %v, right %v: want red and blue", scale, left, right)
			}
		}
	}
}

// A PNG icon and an SVG one replace each other.
func TestButtonIconKinds(t *testing.T) {
	data := twoColorPNG(t, 32)
	b := Button("").SetIconPNG(data).SetIcon(chevronDown)
	if b.png != nil {
		t.Fatal("SetIcon kept the PNG")
	}
	b.SetIconPNG(data)
	if b.icon != nil {
		t.Fatal("SetIconPNG kept the SVG")
	}
}

// A PNG icon is as high as an SVG icon at every text size, as both follow
// the button's text.
func TestButtonPNGIconFollowsTextSize(t *testing.T) {
	data := twoColorPNG(t, 32)
	for _, size := range []Size{XS, Base, XXL} {
		tt := NewTester(t, VBox(0).Add(
			Button("").SetIcon(chevronDown).SetTooltip("SVG").TextSize(size),
			Button("").SetIconPNG(data).SetTooltip("PNG").TextSize(size),
		), 0, 0)
		svg := tt.Find(ByLabel("SVG")).Bounds()
		png := tt.Find(ByLabel("PNG")).Bounds()
		if svg.H != png.H || svg.W != png.W {
			t.Errorf("size %v: the SVG button is %vx%v, the PNG one %vx%v", size, svg.W, svg.H, png.W, png.H)
		}
	}
}

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

// SetAccessibleName names any control, and wins over the tooltip of an
// icon-only button and the label of a Fields row.
func TestSetAccessibleNameOnEveryControl(t *testing.T) {
	tt := NewTester(t, VBox(0).Add(
		Button("").SetIcon(chevronDown).SetTooltip("Go to the next one").SetAccessibleName("Next"),
		Fields().Row("Query", TextBox().SetAccessibleName("Search")),
		ToolStrip(ToolButton("").SetIcon(chevronDown).SetTooltip("Older").SetAccessibleName("Previous")),
	), 0, 0)
	tt.Find(ByRole(ui.RoleButton).Label("Next"))
	tt.Find(ByLabel("Search"))
	tt.Find(ByRole(ui.RoleButton).Label("Previous"))
	if tt.Has(ByLabel("Go to the next one")) {
		t.Fatal("the tooltip names the button over its accessible name")
	}
	if tt.Has(ByRole(ui.RoleTextField).Label("Query")) {
		t.Fatal("the row's label names the text box over its accessible name")
	}
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

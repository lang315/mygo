package widget

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestClipboardText(t *testing.T) {
	Clipboard.SetText("copied")
	if got := Clipboard.Text(); got != "copied" {
		t.Fatalf("Text() %q", got)
	}
}

func TestClipboardImage(t *testing.T) {
	if err := Clipboard.SetImage(testPNG(t, color.RGBA{9, 9, 9, 255})); err != nil {
		t.Fatal(err)
	}
	if Clipboard.Text() != "" {
		t.Fatal("an image on the clipboard still reads as text")
	}
}

func TestSetThemeAndIsDark(t *testing.T) {
	defer SetTheme(ThemeSystem)
	SetTheme(ThemeDark)
	if !IsDark() {
		t.Fatal("not dark after SetTheme(ThemeDark)")
	}
	SetTheme(ThemeLight)
	if IsDark() {
		t.Fatal("dark after SetTheme(ThemeLight)")
	}
}

// SetControlBorder sets the outline of the controls in each appearance, with no
// control touched, and a window that follows the appearance keeps it.
func TestSetControlBorderColorsTheOutlines(t *testing.T) {
	light, dark := ui.Hex("#9ca3af"), ui.Hex("#6b7280")
	outline := func(tt *ui.Tester, want ui.Color) bool {
		t.Helper()
		for _, n := range tt.Nodes() {
			if n.Role != ui.RoleTextField {
				continue
			}
			img, b := tt.Image(), n.Bounds
			y := int(b.Y + b.H/2)
			for x := int(b.X) - 1; x <= int(b.X)+2; x++ {
				if c := img.RGBAAt(x, y); c.R == want.R && c.G == want.G && c.B == want.B {
					return true
				}
			}
			return false
		}
		t.Fatal("no text box")
		return false
	}
	tt := tester(t, TextBox())
	if outline(tt, light) {
		t.Fatal("the outline has the light color before SetControlBorder")
	}
	SetControlBorder(light, dark)
	t.Cleanup(func() { SetControlBorder(ui.Color{}, ui.Color{}) })
	tt.Frame()
	if !outline(tt, light) {
		t.Error("the text box outline is not the light color after SetControlBorder")
	}
	tt.SetDark(true)
	if !outline(tt, dark) {
		t.Error("the text box outline is not the dark color once the appearance is dark")
	}
}

func TestScreenSize(t *testing.T) {
	if w, h := ScreenSize(); w != 1440 || h != 900 { // the fake's display
		t.Fatalf("ScreenSize %v x %v", w, h)
	}
}

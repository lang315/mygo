package widget

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func lum(c ui.Color) float64 {
	ch := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

func ratio(a, b ui.Color) float64 {
	x, y := lum(a)+0.05, lum(b)+0.05
	if x < y {
		x, y = y, x
	}
	return x / y
}

// A Destructive button's text reaches 4.5:1 on its fill at rest, hovered and
// pressed, in both themes; the light theme's fill stays the Danger color,
// darker by 8% hovered and 16% pressed.
func TestDestructiveFillContrast(t *testing.T) {
	black := ui.RGB(0, 0, 0)
	for name, th := range map[string]*ui.Theme{"light": ui.LightTheme(), "dark": ui.DarkTheme()} {
		text := RoleAccentText.resolve(th)
		fills := map[string]ui.Color{
			"rest": Color{role: roleDestructive}.resolve(th), "hover": Color{role: roleDestructiveHover}.resolve(th), "pressed": Color{role: roleDestructivePressed}.resolve(th)}
		for state, fill := range fills {
			if r := ratio(text, fill); r < 4.5 {
				t.Errorf("%s theme, %s: text on the Destructive fill is %.2f:1, want at least 4.5:1", name, state, r)
			}
		}
		if name == "light" {
			if fills["rest"] != th.Danger || fills["hover"] != th.Danger.Mix(black, 0.08) || fills["pressed"] != th.Danger.Mix(black, 0.16) {
				t.Errorf("light theme's Destructive fills changed: %+v", fills)
			}
		}
	}
}

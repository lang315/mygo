package skin

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A skin sets the focus ring and the scroll bars: themeOf copies the four
// tokens into the theme fields of the same names, and the body size into
// FontSize.
func TestThemeOfTakesFocusRingAndScrollbar(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tk := Calm.light
		if dark {
			tk = Calm.dark
		}
		tk.FocusRingWidth, tk.FocusRingOffset, tk.ScrollbarWidth = 3, 2, 10
		tk.ScrollbarStyle = ui.ScrollbarStyle{Square: true}
		th := themeOf(&tk, dark)
		if th.FocusRingWidth != 3 || th.FocusRingOffset != 2 || th.ScrollbarWidth != 10 || !th.ScrollbarStyle.Square {
			t.Errorf("dark=%v: FocusRingWidth %v, FocusRingOffset %v, ScrollbarWidth %v, Square %v; want 3, 2, 10, true",
				dark, th.FocusRingWidth, th.FocusRingOffset, th.ScrollbarWidth, th.ScrollbarStyle.Square)
		}
		if th.FontSize != tk.FontSize || th.FontSize != 13 {
			t.Errorf("dark=%v: FontSize %v, want the tokens' 13", dark, th.FontSize)
		}
	}
}

// Calm draws the focus ring and the scroll bars as the plain themes do.
func TestCalmKeepsThemeDefaults(t *testing.T) {
	for _, c := range []struct {
		dark      bool
		got, base *ui.Theme
	}{
		{false, themeOf(&Calm.light, false), ui.LightTheme()},
		{true, themeOf(&Calm.dark, true), ui.DarkTheme()},
	} {
		g, b := c.got, c.base
		if g.FocusRingWidth != 2 || g.FocusRingOffset != 1 {
			t.Errorf("dark=%v: FocusRingWidth %v, FocusRingOffset %v; want 2, 1", c.dark, g.FocusRingWidth, g.FocusRingOffset)
		}
		if g.FocusRingWidth != b.FocusRingWidth || g.FocusRingOffset != b.FocusRingOffset ||
			g.ScrollbarWidth != b.ScrollbarWidth || g.ScrollbarStyle != b.ScrollbarStyle {
			t.Errorf("dark=%v: focus ring %v/%v, scroll bar %v %+v; the plain theme's %v/%v, %v %+v", c.dark,
				g.FocusRingWidth, g.FocusRingOffset, g.ScrollbarWidth, g.ScrollbarStyle,
				b.FocusRingWidth, b.FocusRingOffset, b.ScrollbarWidth, b.ScrollbarStyle)
		}
	}
}

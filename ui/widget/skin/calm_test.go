package skin

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Calm's tokens make the theme its controls draw with.
func TestCalmTheme(t *testing.T) {
	l := themeOf(&Calm.light, false)
	for _, c := range []struct {
		name      string
		got, want ui.Color
	}{
		{"Background", l.Background, ui.Hex("#ffffff")},
		{"ControlBorder", l.ControlBorder, ui.Hex("#838a94")},
		{"Accent", l.Accent, ui.Hex("#1a63db")},
		{"AccentHover", l.AccentHover, ui.Hex("#1555c2")},
		{"AccentText", l.AccentText, ui.Hex("#ffffff")},
		{"Surface", l.Surface, ui.Hex("#ffffff")},
		{"Border", l.Border, ui.Hex("#e4e7eb")},
		{"TextMuted", l.TextMuted, ui.Hex("#565d68")},
		{"Danger", l.Danger, ui.Hex("#c42b1c")},
		{"Focus", l.Focus, ui.Hex("#1a63db")},
		{"Selection", l.Selection, ui.Hex("#cfe0fb")},
	} {
		if c.got != c.want {
			t.Errorf("light %s is %v, want %v", c.name, c.got, c.want)
		}
	}
	if l.Radius != 6 || l.FontSize != 13 || l.Dark {
		t.Errorf("light Radius %v, FontSize %v, Dark %v; want 6, 13, false", l.Radius, l.FontSize, l.Dark)
	}
	if l.Spacing != ui.LightTheme().Spacing || l.FocusRingWidth != ui.LightTheme().FocusRingWidth {
		t.Errorf("light keeps the base theme's Spacing and FocusRingWidth: got %v and %v", l.Spacing, l.FocusRingWidth)
	}

	d := themeOf(&Calm.dark, true)
	if d.Background != ui.Hex("#15171a") || d.ControlBorder != ui.Hex("#79818c") || d.Accent != ui.Hex("#4a8bff") || !d.Dark {
		t.Errorf("dark Background %v, ControlBorder %v, Accent %v, Dark %v; want #15171a, #79818c, #4a8bff, true",
			d.Background, d.ControlBorder, d.Accent, d.Dark)
	}
	if d.AccentText != ui.Hex("#0b1220") || d.Scrollbar != ui.DarkTheme().Scrollbar {
		t.Errorf("dark AccentText %v and Scrollbar %v; want #0b1220 and DarkTheme's", d.AccentText, d.Scrollbar)
	}
}

// Calm's metrics are the spec's, the same in both appearances.
func TestCalmMetrics(t *testing.T) {
	l := Calm.light
	if l.ControlHeight != 36 || l.RowHeight != 30 || l.RadiusCard != 10 || l.RadiusDialog != 12 {
		t.Errorf("ControlHeight %v, RowHeight %v, RadiusCard %v, RadiusDialog %v; want 36, 30, 10, 12",
			l.ControlHeight, l.RowHeight, l.RadiusCard, l.RadiusDialog)
	}
	d := Calm.dark
	type metrics struct {
		a [19]float32
	}
	of := func(t *skinhook.Tokens) metrics {
		return metrics{[19]float32{t.ControlHeight, t.RowHeight, t.HeaderHeight, t.SettingsRowHeight, t.TabStripHeight,
			t.MenuBarStripHeight, t.MenuBarItemHeight, t.RadiusControl, t.RadiusCard, t.RadiusDialog, t.ToolbarPadTop,
			t.ToolbarPadBottom, t.ToolbarGap, t.ToolbarGroupGap, t.Icon, t.ToolbarIcon, t.CheckBox, t.SwitchWidth, t.SwitchHeight}}
	}
	if of(&l) != of(&d) {
		t.Errorf("dark metrics %v differ from light %v", of(&d), of(&l))
	}
	if l.Body != d.Body || l.Label != d.Label || l.Hover != d.Hover || l.Toggle != d.Toggle {
		t.Errorf("dark type or motion differ from light")
	}
	if l.Body.Size != 13 || l.Body.Line != 20 || l.Hover.Milliseconds() != 90 || l.Toggle.Milliseconds() != 120 {
		t.Errorf("Body %v, Hover %v, Toggle %v; want 13/20, 90ms, 120ms", l.Body, l.Hover, l.Toggle)
	}
}

// A skin has its name.
func TestCalmName(t *testing.T) {
	if Calm.Name() != "Calm" {
		t.Errorf("Calm.Name() is %q", Calm.Name())
	}
}

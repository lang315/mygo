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
		{"Accent", l.Accent, ui.Hex("#1c71d8")},
		{"AccentHover", l.AccentHover, ui.Hex("#1a5fb4")},
		{"AccentText", l.AccentText, ui.Hex("#ffffff")},
		{"Surface", l.Surface, ui.Hex("#ffffff")},
		{"Border", l.Border, ui.Hex("#e4e7eb")},
		{"TextMuted", l.TextMuted, ui.Hex("#565d68")},
		{"Danger", l.Danger, ui.Hex("#c42b1c")},
		{"Focus", l.Focus, ui.Hex("#1c71d8")},
		{"Selection", l.Selection, ui.Hex("#b6d2f3")},
		{"SurfaceHover", l.SurfaceHover, ui.Hex("#eaedf1")},
		{"SurfacePressed", l.SurfacePressed, ui.Hex("#dfe3e9")},
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
	if d.Background != ui.Hex("#15171a") || d.ControlBorder != ui.Hex("#79818c") || d.Accent != ui.Hex("#78aeed") || !d.Dark {
		t.Errorf("dark Background %v, ControlBorder %v, Accent %v, Dark %v; want #15171a, #79818c, #78aeed, true",
			d.Background, d.ControlBorder, d.Accent, d.Dark)
	}
	if d.SurfaceHover != ui.Hex("#2d3239") || d.SurfacePressed != ui.Hex("#343a42") {
		t.Errorf("dark SurfaceHover %v and SurfacePressed %v; want #2d3239 and #343a42, unchanged", d.SurfaceHover, d.SurfacePressed)
	}
	if d.AccentText != ui.Hex("#0b1220") || d.Scrollbar != ui.Hex("#e7e9ec").Alpha(0.35) {
		t.Errorf("dark AccentText %v and Scrollbar %v; want #0b1220 and Text at 35%%", d.AccentText, d.Scrollbar)
	}
}

// themeOf maps every field it sets from its token, in both appearances: a
// copy of Calm's tokens whose Surface, radius and body size differ from
// Background and from the base theme's, so a wrong mapping of any one fails.
func TestThemeOfMapsEveryField(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tk := Calm.light
		if dark {
			tk = Calm.dark
		}
		tk.Surface = ui.Hex("#fdfcfb")
		if dark {
			tk.Surface = ui.Hex("#262a30")
		}
		tk.RadiusControl, tk.FontSize = 7, 17
		// Metrics apart from each other, so a field mapped from another fails.
		tk.ControlHeight, tk.RowHeight, tk.HeaderHeight, tk.TabHeight = 37, 31, 29, 41
		tk.CheckSize, tk.SwitchWidth, tk.SwitchHeight = 14, 43, 23
		th := themeOf(&tk, dark)
		if m := (ui.Metrics{ControlHeight: 37, RowHeight: 31, HeaderHeight: 29, TabHeight: 41, CheckSize: 14, SwitchWidth: 43, SwitchHeight: 23}); th.Metrics != m {
			t.Errorf("dark=%v: Metrics %+v, want %+v", dark, th.Metrics, m)
		}
		scrollbar := tk.Text.Alpha(0.32)
		if dark {
			scrollbar = tk.Text.Alpha(0.35)
		}
		for _, c := range []struct {
			name      string
			got, want ui.Color
		}{
			{"Background", th.Background, tk.Background},
			{"Surface", th.Surface, tk.Surface},
			{"SurfaceHover", th.SurfaceHover, tk.SurfaceHover},
			{"SurfacePressed", th.SurfacePressed, tk.SurfacePressed},
			{"Field", th.Field, tk.Surface},
			{"Border", th.Border, tk.Border},
			{"ControlBorder", th.ControlBorder, tk.ControlBorder},
			{"Text", th.Text, tk.Text},
			{"TextMuted", th.TextMuted, tk.TextMuted},
			{"Accent", th.Accent, tk.Accent.Base},
			{"AccentHover", th.AccentHover, tk.Accent.Hover},
			{"AccentPressed", th.AccentPressed, tk.Accent.Pressed},
			{"AccentText", th.AccentText, tk.Accent.On},
			{"Danger", th.Danger, tk.Danger.Base},
			{"Warning", th.Warning, tk.Warning.Base},
			{"Success", th.Success, tk.Success.Base},
			{"Selection", th.Selection, tk.Selection},
			{"Focus", th.Focus, tk.Accent.Base},
			{"ButtonBorder", th.ButtonBorder, tk.ButtonBorder},
			{"RowHover", th.RowHover, tk.RowHover},
			{"RowSelected", th.RowSelected, tk.RowSelected},
			{"RowSelectedText", th.RowSelectedText, tk.Text},
			{"Scrollbar", th.Scrollbar, scrollbar},
			{"Inverse", th.Inverse, tk.Text},
			{"InverseText", th.InverseText, tk.Background},
		} {
			if c.got != c.want {
				t.Errorf("dark=%v: %s is %v, want %v", dark, c.name, c.got, c.want)
			}
		}
		if th.Dark != dark || th.Radius != 7 || th.FontSize != 17 {
			t.Errorf("dark=%v: Dark %v, Radius %v, FontSize %v; want %v, 7, 17", dark, th.Dark, th.Radius, th.FontSize, dark)
		}
	}
	// Calm's own: Field is the control face, which in dark differs from Background.
	if f := themeOf(&Calm.dark, true).Field; f != ui.Hex("#262a30") {
		t.Errorf("Calm dark Field is %v, want #262a30", f)
	}
	if f := themeOf(&Calm.light, false).Field; f != ui.Hex("#ffffff") {
		t.Errorf("Calm light Field is %v, want #ffffff", f)
	}
}

// Calm's metrics are the spec's, the same in both appearances.
func TestCalmMetrics(t *testing.T) {
	l := Calm.light
	if l.ControlHeight != 36 || l.RowHeight != 30 || l.RadiusCard != 10 {
		t.Errorf("ControlHeight %v, RowHeight %v, RadiusCard %v; want 36, 30, 10",
			l.ControlHeight, l.RowHeight, l.RadiusCard)
	}
	d := Calm.dark
	type metrics struct {
		a [22]float32
	}
	of := func(t *skinhook.Tokens) metrics {
		return metrics{[22]float32{t.ControlHeight, t.RowHeight, t.HeaderHeight, t.SettingsRowHeight, t.TabHeight,
			t.MenuBarStripHeight, t.MenuBarItemHeight, t.RadiusControl, t.RadiusCard, t.ToolbarPadTop,
			t.ToolbarPadBottom, t.ToolbarGap, t.ToolbarGroupGap, t.Icon, t.ToolbarIcon, t.CheckSize, t.SwitchWidth, t.SwitchHeight,
			t.FontSize, t.FocusRingWidth, t.FocusRingOffset, t.ScrollbarWidth}}
	}
	if of(&l) != of(&d) {
		t.Errorf("dark metrics %v differ from light %v", of(&d), of(&l))
	}
	if l.Caption != d.Caption || l.Label != d.Label || l.Heading != d.Heading || l.Title != d.Title {
		t.Errorf("dark type differs from light")
	}
	want := [4]skinhook.Type{{Size: 11, Weight: 400}, {Size: 12, Weight: 600}, {Size: 15, Weight: 600}, {Size: 20, Weight: 600}}
	if got := [4]skinhook.Type{l.Caption, l.Label, l.Heading, l.Title}; l.FontSize != 13 || got != want {
		t.Errorf("FontSize %v, Caption, Label, Heading, Title %v; want 13, %v", l.FontSize, got, want)
	}
}

// Calm's accent tokens are Blue's byte for byte, the dark translucent fills
// included.
func TestCalmAccentValues(t *testing.T) {
	a := ui.Hex("#78aeed")
	for _, c := range []struct {
		name      string
		got, want ui.Color
	}{
		{"light Selection", Calm.light.Selection, ui.Hex("#b6d2f3")},
		{"light RowSelected", Calm.light.RowSelected, ui.Hex("#d6e5f8")},
		{"light Accent.Base", Calm.light.Accent.Base, ui.Hex("#1c71d8")},
		{"light Accent.Hover", Calm.light.Accent.Hover, ui.Hex("#1a5fb4")},
		{"light Accent.Pressed", Calm.light.Accent.Pressed, ui.Hex("#174f96")},
		{"light Accent.Subtle", Calm.light.Accent.Subtle, ui.Hex("#edf4fc")},
		{"light Accent.Border", Calm.light.Accent.Border, ui.Hex("#bbd4f3")},
		{"light Accent.Text", Calm.light.Accent.Text, ui.Hex("#1a5fb4")},
		{"light Accent.On", Calm.light.Accent.On, ui.Hex("#ffffff")},
		{"dark Selection", Calm.dark.Selection, ui.Color{R: 0x78, G: 0xae, B: 0xed, A: a.Alpha(0.35).A}},
		{"dark RowSelected", Calm.dark.RowSelected, ui.Color{R: 0x78, G: 0xae, B: 0xed, A: a.Alpha(0.16).A}},
		{"dark Accent.Base", Calm.dark.Accent.Base, a},
		{"dark Accent.Hover", Calm.dark.Accent.Hover, ui.Hex("#92bdf1")},
		{"dark Accent.Pressed", Calm.dark.Accent.Pressed, ui.Hex("#5f9be6")},
		{"dark Accent.Subtle", Calm.dark.Accent.Subtle, ui.Color{R: 0x78, G: 0xae, B: 0xed, A: a.Alpha(0.14).A}},
		{"dark Accent.Border", Calm.dark.Accent.Border, ui.Color{R: 0x78, G: 0xae, B: 0xed, A: a.Alpha(0.35).A}},
		{"dark Accent.Text", Calm.dark.Accent.Text, ui.Hex("#a8cbf5")},
		{"dark Accent.On", Calm.dark.Accent.On, ui.Hex("#0b1220")},
	} {
		if c.got != c.want {
			t.Errorf("%s is %v, want %v", c.name, c.got, c.want)
		}
	}
	// The alpha bytes themselves: 14%, 16% and 35% of 255, rounded.
	if b := [3]uint8{a.Alpha(0.14).A, a.Alpha(0.16).A, a.Alpha(0.35).A}; b != [3]uint8{36, 41, 89} {
		t.Errorf("the alpha bytes are %v, want [36 41 89]", b)
	}
}

// Calm's chart colors are the plan's, in order, and every palette keeps
// them.
func TestCalmChartValues(t *testing.T) {
	hex := func(s ...string) (c [8]ui.Color) {
		for i, h := range s {
			c[i] = ui.Hex(h)
		}
		return c
	}
	light := hex("#2563eb", "#16a34a", "#ea580c", "#9333ea", "#0891b2", "#db2777", "#a16207", "#4b5563")
	dark := hex("#60a5fa", "#4ade80", "#fb923c", "#c084fc", "#22d3ee", "#f472b6", "#facc15", "#9ca3af")
	if Calm.light.Chart != light {
		t.Errorf("light Chart is %v, want %v", Calm.light.Chart, light)
	}
	if Calm.dark.Chart != dark {
		t.Errorf("dark Chart is %v, want %v", Calm.dark.Chart, dark)
	}
	for _, p := range palettes {
		s := Calm.WithPalette(p)
		if s.light.Chart != light || s.dark.Chart != dark {
			t.Errorf("%s: Chart is %v and %v, want Calm's", p.Name(), s.light.Chart, s.dark.Chart)
		}
	}
}

// A skin has its name.
func TestCalmName(t *testing.T) {
	if Calm.Name() != "Calm" {
		t.Errorf("Calm.Name() is %q", Calm.Name())
	}
}

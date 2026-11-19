package skin

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Compact of Calm's tokens: controls 6 lower, rows 4, a settings row 8, the
// tool strip's room by 30/36, rounded; the rest as Calm's. The values are
// worked out by hand from calm.go.
func TestCompactCalmSizes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		c := compact(Calm.Tokens(dark))
		for _, s := range []struct {
			name      string
			got, want float32
		}{
			{"ControlHeight", c.ControlHeight, 30},
			{"RowHeight", c.RowHeight, 26},
			{"HeaderHeight", c.HeaderHeight, 26},
			{"TabHeight", c.TabHeight, 34},
			{"MenuBarStripHeight", c.MenuBarStripHeight, 30},
			{"MenuBarItemHeight", c.MenuBarItemHeight, 24},
			{"SettingsRowHeight", c.SettingsRowHeight, 48},
			{"ToolbarPadTop", c.ToolbarPadTop, 8},        // 10 × 30/36 = 8.33
			{"ToolbarPadBottom", c.ToolbarPadBottom, 10}, // 12 × 30/36 = 10
			{"ToolbarGap", c.ToolbarGap, 8},
			{"ToolbarGroupGap", c.ToolbarGroupGap, 10},
			{"Icon", c.Icon, 16},
			{"ToolbarIcon", c.ToolbarIcon, 17},
			{"CheckSize", c.CheckSize, 15},
			{"SwitchWidth", c.SwitchWidth, 40},
			{"SwitchHeight", c.SwitchHeight, 22},
			{"RadiusControl", c.RadiusControl, 6},
			{"RadiusCard", c.RadiusCard, 10},
			{"FontSize", c.FontSize, 13},
			{"Title.Size", c.Title.Size, 20},
		} {
			if s.got != s.want {
				t.Errorf("dark %v: compact %s is %v, want %v", dark, s.name, s.got, s.want)
			}
		}
		if c.Accent != Calm.Tokens(dark).Accent || c.Background != Calm.Tokens(dark).Background {
			t.Errorf("dark %v: compact changed a color", dark)
		}
	}
}

// A height is never under FontSize + 8.
func TestCompactFloor(t *testing.T) {
	c := compact(skinhook.Tokens{FontSize: 13, ControlHeight: 24, RowHeight: 22})
	if c.ControlHeight != 21 || c.RowHeight != 21 {
		t.Errorf("compact of ControlHeight 24 and RowHeight 22 with FontSize 13 gives %v and %v, want 21 and 21", c.ControlHeight, c.RowHeight)
	}
}

// Compact never makes a height taller: one already under FontSize + 8
// keeps its own.
func TestCompactNeverGrows(t *testing.T) {
	c := compact(skinhook.Tokens{FontSize: 13, ControlHeight: 18, RowHeight: 20})
	if c.ControlHeight != 18 || c.RowHeight != 20 {
		t.Errorf("compact of ControlHeight 18 and RowHeight 20 with FontSize 13 gives %v and %v, want 18 and 20", c.ControlHeight, c.RowHeight)
	}
}

// A size that is 0 (unset) stays 0; with no ControlHeight the tool strip's
// room keeps its size.
func TestCompactLeavesZero(t *testing.T) {
	if c := compact(skinhook.Tokens{}); !reflect.DeepEqual(c, skinhook.Tokens{}) {
		t.Errorf("compact of zero tokens gives %+v, want zero", c)
	}
	c := compact(skinhook.Tokens{FontSize: 13, RowHeight: 30, ToolbarGap: 10})
	if c.ControlHeight != 0 || c.TabHeight != 0 || c.SettingsRowHeight != 0 || c.RowHeight != 26 || c.ToolbarGap != 10 {
		t.Errorf("compact with ControlHeight 0: ControlHeight %v, TabHeight %v, SettingsRowHeight %v, RowHeight %v, ToolbarGap %v; want 0, 0, 0, 26, 10",
			c.ControlHeight, c.TabHeight, c.SettingsRowHeight, c.RowHeight, c.ToolbarGap)
	}
}

// Calm with Standard gives the hook Calm's tokens.
func TestStandardIsIdentity(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	for _, s := range []*Skin{Calm.WithDensity(Standard), Calm.WithDensity(Compact).WithDensity(Standard)} {
		Use(s)
		for _, dark := range []bool{false, true} {
			if h := skinhook.Get(dark); h == nil || !reflect.DeepEqual(*h, Calm.Tokens(dark)) {
				t.Errorf("dark %v: Standard's tokens at Use are not Calm's", dark)
			}
		}
	}
}

// The order of WithDensity and WithPalette does not matter.
func TestDensityOrder(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	Use(Calm.WithDensity(Compact).WithPalette(Teal))
	a := [2]skinhook.Tokens{*skinhook.Get(false), *skinhook.Get(true)}
	Use(Calm.WithPalette(Teal).WithDensity(Compact))
	b := [2]skinhook.Tokens{*skinhook.Get(false), *skinhook.Get(true)}
	if !reflect.DeepEqual(a, b) {
		t.Error("Calm.WithDensity(Compact).WithPalette(Teal) and Calm.WithPalette(Teal).WithDensity(Compact) give other tokens at Use")
	}
	if a[0].ControlHeight != 30 || a[1].ControlHeight != 30 || a[0].Accent.Base != ui.Hex("#0f766e") || a[1].Accent.Base != ui.Hex("#2bb5a3") {
		t.Errorf("ControlHeight %v and %v, Accent %v and %v; want 30 and Teal's #0f766e and #2bb5a3",
			a[0].ControlHeight, a[1].ControlHeight, a[0].Accent.Base, a[1].Accent.Base)
	}
}

// Tokens stay Standard on a compact skin, and WithDensity does not change
// its receiver.
func TestTokensStayStandard(t *testing.T) {
	c := Calm.WithDensity(Compact)
	if h := c.Tokens(false).ControlHeight; h != 36 {
		t.Errorf("Calm.WithDensity(Compact).Tokens(false).ControlHeight is %v, want 36", h)
	}
	if c.Density() != Compact {
		t.Errorf("Calm.WithDensity(Compact).Density() is %v, want Compact", c.Density())
	}
	if Calm.Density() != Standard {
		t.Errorf("Calm.Density() is %v after WithDensity, want Standard", Calm.Density())
	}
	if c.Palette() != Blue || c.Name() != "Calm" {
		t.Errorf("Calm.WithDensity(Compact) is %q with palette %v, want Calm with Blue", c.Name(), c.Palette().Name())
	}
}

// A compact Calm sizes the widgets: a button 30 and a row 26, by the text
// scale too; Calm again brings back 36 and 30.
func TestCompactLayout(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	tt := widget.NewTester(t, sizeForm(), 600, 700)
	check := func(when string, button, row float32) {
		t.Helper()
		tt.Frame()
		got := widgetSizes(t, tt)
		if !same(got["Button"], button) || !same(got["DataGridView row"], row) {
			t.Errorf("%s: a Button is %v and a row %v, want %v and %v", when, got["Button"], got["DataGridView row"], button, row)
		}
	}
	Use(Calm.WithDensity(Compact))
	check("Compact", 30, 26)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	check("Compact at text scale 1.5", 45, 39)
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	Use(Calm)
	check("Calm after Compact", 36, 30)
}

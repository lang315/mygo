package skin

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Case 12: Check names each new pair of the status components when a
// skin fails it.
func TestCheckStatusPairs(t *testing.T) {
	derive := func(edit func(*skinhook.Tokens)) error {
		l, d := Calm.Tokens(false), Calm.Tokens(true)
		edit(&l)
		return Calm.Derive("Test", l, d).Check()
	}
	for _, c := range []struct {
		name string
		edit func(*skinhook.Tokens)
	}{
		{"Warning Base on Subtle", func(tk *skinhook.Tokens) { tk.Warning.Base = tk.Warning.Subtle }},
		{"Info Text on Subtle over RowSelected", func(tk *skinhook.Tokens) {
			tk.Info.Subtle, tk.Info.Text = tk.RowSelected, tk.RowSelected
		}},
		{"TextMuted on SurfaceHover", func(tk *skinhook.Tokens) { tk.TextMuted = tk.SurfaceHover }},
	} {
		if err := derive(c.edit); err == nil || !strings.Contains(err.Error(), "light: ") || !strings.Contains(err.Error(), c.name) {
			t.Errorf("a skin made to fail %q gives %v, want an error naming it", c.name, err)
		}
	}
	// A translucent Subtle that fails only over a hovered row is named so:
	// Calm dark's Success text over a RowHover of its own green.
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	d.RowHover = d.Success.Text.Alpha(0.6)
	err := Calm.Derive("Test", l, d).Check()
	if err == nil || !strings.Contains(err.Error(), "Success Text on Subtle over RowHover") {
		t.Errorf("a green RowHover under Success's text gives %v, want an error naming Success Text on Subtle over RowHover", err)
	}
	// And over a selected row of its green, which RowHover's pair does not see.
	l, d = Calm.Tokens(false), Calm.Tokens(true)
	d.RowSelected = d.Success.Text.Alpha(0.6)
	err = Calm.Derive("Test", l, d).Check()
	if err == nil || !strings.Contains(err.Error(), "Success Text on Subtle over RowSelected") {
		t.Errorf("a green RowSelected under Success's text gives %v, want an error naming Success Text on Subtle over RowSelected", err)
	}
}

// Case 13: every built-in skin, with each of its palettes, passes Check in
// both appearances with the new pairs (Check measures both); the accent
// grid is TestWithAccentPassesCheck's.
func TestBuiltinsPassStatusPairs(t *testing.T) {
	for _, s := range []*Skin{Calm, Classic, HighContrast} {
		if err := s.Check(); err != nil {
			t.Errorf("%s: %v", s.Name(), err)
		}
		for _, p := range s.Palettes() {
			if err := s.WithPalette(p).Check(); err != nil {
				t.Errorf("%s with %s: %v", s.Name(), p.Name(), err)
			}
			for _, d := range []Density{Compact} {
				if err := s.WithPalette(p).WithDensity(d).Check(); err != nil {
					t.Errorf("%s with %s, compact: %v", s.Name(), p.Name(), err)
				}
			}
		}
	}
}

// Case 14: HighContrast's looks for Badge and InfoBar draw on Surface, and
// each family's Text on Surface is 7:1 or more in both appearances.
func TestHighContrastStatusOnSurface(t *testing.T) {
	l := HighContrast.Looks()
	for name, look := range map[string]string{"Badge": l.Badge, "InfoBar": l.InfoBar} {
		if !strings.Contains(" "+look+" ", " bg-surface ") || !strings.Contains(look, "border-status") {
			t.Errorf("HighContrast's %s look is %q, want bg-surface and border-status", name, look)
		}
	}
	// Dark frames each status in its Base, opaque, not a tint of it.
	d := HighContrast.Tokens(true)
	for name, f := range map[string]skinhook.Family{"Info": d.Info, "Success": d.Success, "Warning": d.Warning, "Danger": d.Danger} {
		if f.Border != f.Base || f.Border.A != 255 {
			t.Errorf("HighContrast dark %s Border is %v, want its Base %v, opaque", name, f.Border, f.Base)
		}
	}
	for _, dark := range []bool{false, true} {
		tk := HighContrast.Tokens(dark)
		for name, f := range map[string]skinhook.Family{"Info": tk.Info, "Success": tk.Success, "Warning": tk.Warning, "Danger": tk.Danger} {
			if r := contrastOf(f.Text, tk.Surface); r < 7 {
				t.Errorf("dark %v: %s Text on Surface is %.2f:1, want 7:1", dark, name, r)
			}
		}
	}
}

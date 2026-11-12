package skin

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

var palettes = []*Palette{Cham, Indigo, Ink, Teal}

// Calm's palette is Chàm.
func TestCalmPaletteIsCham(t *testing.T) {
	if Calm.Palette() != Cham {
		t.Errorf("Calm's palette is %v, want Chàm", Calm.Palette())
	}
	for _, c := range []struct {
		p    *Palette
		name string
	}{{Cham, "Chàm"}, {Indigo, "Indigo"}, {Ink, "Ink"}, {Teal, "Teal"}} {
		if c.p.Name() != c.name {
			t.Errorf("Name() is %q, want %q", c.p.Name(), c.name)
		}
	}
}

// leastDeltaE is the smallest ΔE of Info's text and a status text, over
// every eye: the number stateDistance holds to its floor.
func leastDeltaE(tk *skinhook.Tokens) float64 {
	least := math.Inf(1)
	for _, s := range []ui.Color{tk.Success.Text, tk.Warning.Text, tk.Danger.Text} {
		for _, v := range visions {
			least = min(least, deltaE(tk.Info.Text, s, v.m))
		}
	}
	return least
}

// Every palette passes contrast and the state distance in both appearances,
// its dark translucent fills blended as check blends them.
func TestPalettesPassCheck(t *testing.T) {
	for _, p := range palettes {
		s := Calm.WithPalette(p)
		if err := check(&s.light, false); err != nil {
			t.Errorf("%s light: %v", p.Name(), err)
		}
		if err := check(&s.dark, true); err != nil {
			t.Errorf("%s dark: %v", p.Name(), err)
		}
		t.Logf("%s: least state ΔE %.1f light, %.1f dark", p.Name(), leastDeltaE(&s.light), leastDeltaE(&s.dark))
	}
}

// Calm with Teal draws with Teal's accent and Info, and keeps Calm's name.
func TestWithPalette(t *testing.T) {
	s := Calm.WithPalette(Teal)
	if a := themeOf(&s.light, false).Accent; a != ui.Hex("#0f766e") {
		t.Errorf("light Accent is %v, want #0f766e", a)
	}
	if a := themeOf(&s.dark, true).Accent; a != ui.Hex("#2bb5a3") {
		t.Errorf("dark Accent is %v, want #2bb5a3", a)
	}
	if s.light.Info.Text != ui.Hex("#34507a") || s.light.Info.Text == s.light.Accent.Text {
		t.Errorf("light Info.Text is %v, want #34507a, not the accent's %v", s.light.Info.Text, s.light.Accent.Text)
	}
	if s.dark.Info.Text != ui.Hex("#b6c2f2") {
		t.Errorf("dark Info.Text is %v, want #b6c2f2", s.dark.Info.Text)
	}
	if s.Name() != "Calm" || s.Palette() != Teal {
		t.Errorf("Name() %q and Palette() %v, want Calm and Teal", s.Name(), s.Palette())
	}
	// What the palette does not decide stays Calm's.
	if s.light.Content != Calm.light.Content || s.dark.Success != Calm.dark.Success || s.light.ControlHeight != 36 {
		t.Errorf("Content %v, dark Success %v, ControlHeight %v; want Calm's", s.light.Content, s.dark.Success, s.light.ControlHeight)
	}
}

// Every token a palette decides follows the table and the mix rules: light
// Subtle, Border, RowSelected and Selection are the base mixed with white at
// 92%, 70%, 90% and 80%; dark ones the base at 14%, 35%, 16% and 35%.
func TestPaletteValues(t *testing.T) {
	white := ui.Hex("#ffffff")
	for _, c := range []struct {
		p                                   *Palette
		lb, lh, lp, lt, db, dh, dp, dt, don string
		info                                [10]string // light base, hover, pressed, text; dark base, hover, pressed, text, on; "" for the accent
	}{
		{p: Cham, lb: "#4e3bb0", lh: "#45349b", lp: "#3b2d86", lt: "#46359e", db: "#9384f0", dh: "#a79bf3", dp: "#7f6eed", dt: "#bdb4f9", don: "#0e0b2a"},
		{p: Indigo, lb: "#4f46e5", lh: "#4338ca", lp: "#3730a3", lt: "#3f37c9", db: "#8b8ff9", dh: "#a3a6fb", dp: "#7477ef", dt: "#b9bcfc", don: "#0e0f2a"},
		{p: Ink, lb: "#2c4f8c", lh: "#26457b", lp: "#203b69", lt: "#284778", db: "#7b9fdc", dh: "#93b1e4", dp: "#6a8dcf", dt: "#a9c1ea", don: "#0b1220"},
		{p: Teal, lb: "#0f766e", lh: "#0c665f", lp: "#0a5650", lt: "#0b5d56", db: "#2bb5a3", dh: "#45c4b3", dp: "#20a090", dt: "#7fdccf", don: "#06201c",
			info: [10]string{"#3d5f8f", "#36547e", "#2e486d", "#34507a", "#8c9fe0", "#9dade5", "#7e8fca", "#b6c2f2", "#0b1220"}},
	} {
		lightFamily := func(b, h, p, tx string) skinhook.Family {
			base := ui.Hex(b)
			return skinhook.Family{Base: base, Hover: ui.Hex(h), Pressed: ui.Hex(p),
				Subtle: base.Mix(white, 0.92), Border: base.Mix(white, 0.70), Text: ui.Hex(tx), On: white}
		}
		darkFamily := func(b, h, p, tx, on string) skinhook.Family {
			base := ui.Hex(b)
			return skinhook.Family{Base: base, Hover: ui.Hex(h), Pressed: ui.Hex(p),
				Subtle: base.Alpha(0.14), Border: base.Alpha(0.35), Text: ui.Hex(tx), On: ui.Hex(on)}
		}
		la, da := lightFamily(c.lb, c.lh, c.lp, c.lt), darkFamily(c.db, c.dh, c.dp, c.dt, c.don)
		li, di := la, da
		if c.info[0] != "" {
			i := c.info
			li, di = lightFamily(i[0], i[1], i[2], i[3]), darkFamily(i[4], i[5], i[6], i[7], i[8])
		}
		s := Calm.WithPalette(c.p)
		n := c.p.Name()
		if s.light.Accent != la || s.light.Info != li {
			t.Errorf("%s light Accent %v, Info %v; want %v, %v", n, s.light.Accent, s.light.Info, la, li)
		}
		if s.dark.Accent != da || s.dark.Info != di {
			t.Errorf("%s dark Accent %v, Info %v; want %v, %v", n, s.dark.Accent, s.dark.Info, da, di)
		}
		if s.light.OnAccent != white || s.light.RowSelected != la.Base.Mix(white, 0.90) || s.light.Selection != la.Base.Mix(white, 0.80) {
			t.Errorf("%s light OnAccent %v, RowSelected %v, Selection %v", n, s.light.OnAccent, s.light.RowSelected, s.light.Selection)
		}
		if s.dark.OnAccent != ui.Hex(c.don) || s.dark.RowSelected != da.Base.Alpha(0.16) || s.dark.Selection != da.Base.Alpha(0.35) {
			t.Errorf("%s dark OnAccent %v, RowSelected %v, Selection %v", n, s.dark.OnAccent, s.dark.RowSelected, s.dark.Selection)
		}
	}
}

// WithPalette makes a new skin: Calm stays Chàm, in both appearances and
// in the tokens Use(Calm) gives the hook afterwards; WithPalette(nil) is the
// skin itself.
func TestWithPaletteLeavesTheSkin(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	if s := Calm.WithPalette(Teal); s == Calm {
		t.Fatal("Calm.WithPalette(Teal) is Calm itself")
	}
	if Calm.light.Accent.Base != ui.Hex("#4e3bb0") || Calm.dark.Accent.Base != ui.Hex("#9384f0") || Calm.Palette() != Cham {
		t.Errorf("after WithPalette(Teal), Calm's accent is %v and %v, palette %v; want #4e3bb0, #9384f0, Chàm",
			Calm.light.Accent.Base, Calm.dark.Accent.Base, Calm.Palette())
	}
	if Calm.light.Info.Text != ui.Hex("#46359e") || Calm.dark.Selection != ui.Hex("#9384f0").Alpha(0.35) {
		t.Errorf("after WithPalette(Teal), Calm's Info.Text %v and dark Selection %v", Calm.light.Info.Text, Calm.dark.Selection)
	}
	Use(Calm)
	if l, d := skinhook.Get(false), skinhook.Get(true); l.Accent.Base != ui.Hex("#4e3bb0") || d.Accent.Base != ui.Hex("#9384f0") {
		t.Errorf("Use(Calm) after WithPalette(Teal) gives the hook %v and %v, want Chàm's", l.Accent.Base, d.Accent.Base)
	}
	if Calm.WithPalette(nil) != Calm {
		t.Error("Calm.WithPalette(nil) is not Calm")
	}
}

// Teal's own text is too near green, so Teal's Info is a blue of its own:
// Teal's tokens with Info as the accent fail the check, naming the pair.
func TestTealNeedsItsOwnInfo(t *testing.T) {
	tk := Calm.WithPalette(Teal).light
	if err := check(&tk, false); err != nil {
		t.Fatalf("Teal light: %v", err)
	}
	tk.Info = tk.Accent
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Info Text near Success Text") {
		t.Errorf("Teal with Info = Accent gives %v, want an error naming Info Text near Success Text", err)
	}
}

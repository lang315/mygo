package skin

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/ui"
)

// labOfColor is c in Oklab.
func labOfColor(c ui.Color) [3]float32 {
	return scene.Oklab(scene.Color{R: c.R, G: c.G, B: c.B, A: c.A})
}

// fromOklab undoes scene.Oklab, to 1 per channel.
func TestOklabRoundTrip(t *testing.T) {
	n := 0
	for _, r := range []uint8{0, 85, 170, 255} {
		for _, g := range []uint8{0, 85, 170, 255} {
			for _, b := range []uint8{0, 85, 170, 255} {
				c := ui.RGB(r, g, b)
				got := fromOklab(labOfColor(c))
				d := func(x, y uint8) int { return int(math.Abs(float64(x) - float64(y))) }
				if d(got.R, r) > 1 || d(got.G, g) > 1 || d(got.B, b) > 1 || got.A != 255 {
					t.Errorf("fromOklab(Oklab(%v)) is %v", c, got)
				}
				n++
			}
		}
	}
	if n != 64 {
		t.Fatalf("%d colors, want 64", n)
	}
}

// accents are the colors TestWithAccentPassesCheck gives Calm: hues every
// 30°, four lightnesses and two chromas, and colors near the edges: yellow,
// a very light blue, near black, red, green and amber.
func accents() []ui.Color {
	var cs []ui.Color
	for h := 0; h < 360; h += 30 {
		for _, l := range []float64{0.35, 0.55, 0.75, 0.9} {
			for _, ch := range []float64{0.05, 0.15} {
				a := float64(h) * math.Pi / 180
				cs = append(cs, fromOklab([3]float32{float32(l), float32(ch * math.Cos(a)), float32(ch * math.Sin(a))}))
			}
		}
	}
	for _, s := range []string{"#ffd400", "#e0f2ff", "#0a0a0a", "#d32f2f", "#2e7d32", "#f59e0b"} {
		cs = append(cs, ui.Hex(s))
	}
	return cs
}

// Every accent makes a skin that passes Check.
func TestWithAccentPassesCheck(t *testing.T) {
	cs := accents()
	if len(cs) != 102 {
		t.Fatalf("%d accents, want 102", len(cs))
	}
	for _, c := range cs {
		if s := Calm.WithAccent(c); s.Check() != nil {
			t.Errorf("Calm.WithAccent(%v), light Base %v, dark Base %v: %v", hexOf(c), hexOf(s.Tokens(false).Accent.Base), hexOf(s.Tokens(true).Accent.Base), s.Check())
		}
	}
}

// A color that passes is the light accent as it is.
func TestWithAccentKeepsAPassingColor(t *testing.T) {
	if got := Calm.WithAccent(ui.Hex("#1c71d8")).Tokens(false).Accent.Base; got != ui.Hex("#1c71d8") {
		t.Errorf("Calm.WithAccent(#1c71d8)'s light Base is %v, want #1c71d8", hexOf(got))
	}
}

// hueOf is c's Oklab hue in degrees.
func hueOf(c ui.Color) float64 {
	l := labOfColor(c)
	return math.Atan2(float64(l[2]), float64(l[1])) * 180 / math.Pi
}

// A yellow or a cyan too light for white text darkens to one of its hue,
// not to black.
func TestWithAccentKeepsTheHue(t *testing.T) {
	for _, s := range []string{"#ffd400", "#2bf8ff"} {
		in := ui.Hex(s)
		base := Calm.WithAccent(in).Tokens(false).Accent.Base
		if d := math.Abs(hueOf(base) - hueOf(in)); d > 2 {
			t.Errorf("the light Base of %s is %v, hue %.1f°, %.1f° from the input's %.1f°; want 2° or less", s, hexOf(base), hueOf(base), d, hueOf(in))
		}
		if r := contrastOf(white, base); r < 4.5 {
			t.Errorf("white on the light Base %v of %s is %.2f:1, want 4.5:1", hexOf(base), s, r)
		}
		if l, _ := lightChroma(base); base == in || l < 0.3 {
			t.Errorf("the light Base of %s is %v, Oklab L %.2f: want a darker color of its hue, L 0.3 or more", s, hexOf(base), l)
		}
	}
}

// A near-black accent lightens just enough for text on its selected row:
// #0a0a0a to a gray of about #131313, as the gray scan before the rule
// found the boundary.
func TestWithAccentLightensNearBlack(t *testing.T) {
	b := Calm.WithAccent(ui.Hex("#0a0a0a")).Tokens(false).Accent.Base
	if b.R != b.G || b.G != b.B || b.R < 0x13 || b.R > 0x15 {
		t.Errorf("the light Base of #0a0a0a is %v, want a gray #131313 to #151515", hexOf(b))
	}
	if b := Calm.WithAccent(ui.Hex("#131313")).Tokens(false).Accent.Base; b != ui.Hex("#131313") {
		t.Errorf("the light Base of #131313, which passes, is %v, want #131313", hexOf(b))
	}
}

// The light base lightens for a skin's Text too, when its Text is lighter
// than its TextMuted.
func TestWithAccentRowText(t *testing.T) {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Text, l.TextMuted = ui.Hex("#6b7280"), ui.Hex("#1a1d21")
	tk := New("X", l, d).WithAccent(ui.Hex("#0a0a0a")).Tokens(false)
	if r := ratio(&tk, tk.Text, tk.RowSelected); r < 4.5 {
		t.Errorf("Text on RowSelected is %.2f:1 with the light Base %v, want 4.5:1", r, hexOf(tk.Accent.Base))
	}
}

// The dark On is a dark color of the accent's hue, Oklab L 0.18 and chroma
// 0.04; a gray's is a gray.
func TestWithAccentDarkOn(t *testing.T) {
	in := ui.Hex("#7c3aed")
	on := Calm.WithAccent(in).Tokens(true).Accent.On
	if l, c := lightChroma(on); math.Abs(l-0.18) > 0.01 || math.Abs(c-0.04) > 0.01 || math.Abs(hueOf(on)-hueOf(in)) > 10 {
		t.Errorf("the dark On of #7c3aed is %v, L %.3f, chroma %.3f, hue %.0f°; want L 0.18, chroma 0.04, hue %.0f°", hexOf(on), l, c, hueOf(on), hueOf(in))
	}
	if on := Calm.WithAccent(ui.Hex("#808080")).Tokens(true).Accent.On; on.R != on.G || on.G != on.B {
		t.Errorf("the dark On of #808080 is %v, want a gray", hexOf(on))
	}
}

// An accent near a status color keeps the skin's Info; one apart from them
// is the Info too.
func TestWithAccentInfoNearStatus(t *testing.T) {
	red := Calm.WithAccent(ui.Hex("#d32f2f"))
	if got, want := red.Tokens(false).Info, Blue.light.info; got != want {
		t.Errorf("Calm.WithAccent(#d32f2f)'s light Info Base is %v, want Blue's %v", hexOf(got.Base), hexOf(want.Base))
	}
	if got, want := red.Tokens(true).Info, Blue.dark.info; got != want {
		t.Errorf("Calm.WithAccent(#d32f2f)'s dark Info Base is %v, want Blue's %v", hexOf(got.Base), hexOf(want.Base))
	}
	violet := Calm.WithAccent(ui.Hex("#7c3aed"))
	for _, dark := range []bool{false, true} {
		if tk := violet.Tokens(dark); tk.Info != tk.Accent {
			t.Errorf("dark %v: Calm.WithAccent(#7c3aed)'s Info Base is %v, want its Accent %v", dark, hexOf(tk.Info.Base), hexOf(tk.Accent.Base))
		}
	}
	if violet.Tokens(false).Accent.Base != ui.Hex("#7c3aed") {
		t.Errorf("Calm.WithAccent(#7c3aed)'s light Base is %v, want #7c3aed", hexOf(violet.Tokens(false).Accent.Base))
	}
}

// The palette is named by the color's hex.
func TestWithAccentName(t *testing.T) {
	s := Calm.WithAccent(ui.Hex("#7c3aed"))
	if n := s.Palette().Name(); n != "#7c3aed" {
		t.Errorf("Calm.WithAccent(#7c3aed).Palette().Name() is %q, want #7c3aed", n)
	}
	if s.Name() != "Calm" || Calm.Palette() != Blue {
		t.Errorf("WithAccent named the skin %q and left Calm's palette %v; want Calm and Blue", s.Name(), Calm.Palette().Name())
	}
}

// WithAccent keeps the density, and WithDensity the accent.
func TestWithAccentKeepsTheDensity(t *testing.T) {
	a := Calm.WithDensity(Compact).WithAccent(ui.Hex("#7c3aed"))
	b := Calm.WithAccent(ui.Hex("#7c3aed")).WithDensity(Compact)
	if a.Density() != Compact || b.Density() != Compact || !reflect.DeepEqual([2]Tokens{a.Tokens(false), a.Tokens(true)}, [2]Tokens{b.Tokens(false), b.Tokens(true)}) {
		t.Error("WithAccent and WithDensity in either order give other skins")
	}
}

// The last of WithAccent and WithPalette wins.
func TestLastCallWins(t *testing.T) {
	if got := Calm.WithAccent(ui.Hex("#7c3aed")).WithPalette(Teal).Tokens(false).Accent; got != Teal.light.accent {
		t.Errorf("Calm.WithAccent(#7c3aed).WithPalette(Teal)'s light Accent Base is %v, want Teal's #0f766e", hexOf(got.Base))
	}
	if got := Calm.WithPalette(Teal).WithAccent(ui.Hex("#7c3aed")).Tokens(false).Accent.Base; got != ui.Hex("#7c3aed") {
		t.Errorf("Calm.WithPalette(Teal).WithAccent(#7c3aed)'s light Accent Base is %v, want #7c3aed", hexOf(got))
	}
}

// Calm lists its palettes, Blue first; a skin made with New none; the With
// methods keep the list.
func TestPalettes(t *testing.T) {
	names := func(ps []*Palette) []string {
		var n []string
		for _, p := range ps {
			n = append(n, p.Name())
		}
		return n
	}
	want := []string{"Blue", "Chàm", "Indigo", "Ink", "Teal"}
	if got := names(Calm.Palettes()); !slices.Equal(got, want) {
		t.Errorf("Calm.Palettes() are %q, want %q", got, want)
	}
	if ps := New("X", Calm.Tokens(false), Calm.Tokens(true)).Palettes(); ps != nil {
		t.Errorf("New(...).Palettes() is %v, want nil", names(ps))
	}
	for name, s := range map[string]*Skin{
		"WithAccent": Calm.WithAccent(ui.Hex("#7c3aed")), "WithPalette": Calm.WithPalette(Teal), "WithDensity": Calm.WithDensity(Compact),
	} {
		if got := names(s.Palettes()); !slices.Equal(got, want) {
			t.Errorf("Calm.%s(...).Palettes() are %q, want %q", name, got, want)
		}
	}
	Calm.Palettes()[0] = Teal // the list is a copy
	if Calm.Palettes()[0] != Blue {
		t.Error("changing the list Palettes returned changed Calm's")
	}
}

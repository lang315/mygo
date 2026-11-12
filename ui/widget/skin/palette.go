package skin

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Palette is the accent of a skin: the accent family, the Info family of
// the running state, the text on the accent, the selected row and the text
// selection, in light and dark. Every palette passes the skin's contrast
// floors and keeps its Info apart from the status colors, also for a
// red-green color-blind eye. A skin takes one with WithPalette:
//
//	skin.Use(skin.Calm.WithPalette(skin.Teal))
type Palette struct {
	name        string
	light, dark paletteTokens
}

// paletteTokens are the tokens a palette decides in one appearance.
type paletteTokens struct {
	accent, info                     skinhook.Family
	onAccent, rowSelected, selection ui.Color
}

// Name returns the palette's name, as "Teal".
func (p *Palette) Name() string { return p.name }

// The palettes: Chàm, Calm's own, a blue-violet indigo dye; Indigo, a
// brighter violet-blue; Ink, a deep blue; Teal, whose Info is a blue of
// its own, since its accent sits too near green.
var (
	Cham   = newPalette("Chàm", lightFamily("#4e3bb0", "#45349b", "#3b2d86", "#46359e"), darkFamily("#9384f0", "#a79bf3", "#7f6eed", "#bdb4f9", "#0e0b2a"))
	Indigo = newPalette("Indigo", lightFamily("#4f46e5", "#4338ca", "#3730a3", "#3f37c9"), darkFamily("#8b8ff9", "#a3a6fb", "#7477ef", "#b9bcfc", "#0e0f2a"))
	Ink    = newPalette("Ink", lightFamily("#2c4f8c", "#26457b", "#203b69", "#284778"), darkFamily("#7b9fdc", "#93b1e4", "#6a8dcf", "#a9c1ea", "#0b1220"))
	Teal   = newPalette("Teal", lightFamily("#0f766e", "#0c665f", "#0a5650", "#0b5d56"), darkFamily("#2bb5a3", "#45c4b3", "#20a090", "#7fdccf", "#06201c")).
		withInfo(lightFamily("#3d5f8f", "#36547e", "#2e486d", "#34507a"), darkFamily("#8c9fe0", "#9dade5", "#7e8fca", "#b6c2f2", "#0b1220"))
)

var white = ui.Hex("#ffffff")

// lightFamily is a light family from its base, hover, pressed and text:
// white on the base, and the subtle fill and border the base mixed with
// white at 92% and 70%.
func lightFamily(base, hover, pressed, text string) skinhook.Family {
	b := ui.Hex(base)
	return skinhook.Family{Base: b, Hover: ui.Hex(hover), Pressed: ui.Hex(pressed),
		Subtle: b.Mix(white, 0.92), Border: b.Mix(white, 0.70), Text: ui.Hex(text), On: white}
}

// darkFamily is a dark family from its base, hover, pressed, text and the
// color on the base: the subtle fill and border the base at 14% and 35%.
func darkFamily(base, hover, pressed, text, on string) skinhook.Family {
	b := ui.Hex(base)
	return skinhook.Family{Base: b, Hover: ui.Hex(hover), Pressed: ui.Hex(pressed),
		Subtle: b.Alpha(0.14), Border: b.Alpha(0.35), Text: ui.Hex(text), On: ui.Hex(on)}
}

// newPalette is a palette whose Info is its accent: the selected row and the
// selection are the light base mixed with white at 90% and 80%, and the dark
// base at 16% and 35%.
func newPalette(name string, light, dark skinhook.Family) *Palette {
	return &Palette{name: name,
		light: paletteTokens{accent: light, info: light, onAccent: white,
			rowSelected: light.Base.Mix(white, 0.90), selection: light.Base.Mix(white, 0.80)},
		dark: paletteTokens{accent: dark, info: dark, onAccent: dark.On,
			rowSelected: dark.Base.Alpha(0.16), selection: dark.Base.Alpha(0.35)},
	}
}

// withInfo gives p an Info of its own.
func (p *Palette) withInfo(light, dark skinhook.Family) *Palette {
	p.light.info, p.dark.info = light, dark
	return p
}

// apply writes the palette's tokens over t.
func (pt *paletteTokens) apply(t *skinhook.Tokens) {
	t.Accent, t.Info = pt.accent, pt.info
	t.OnAccent, t.RowSelected, t.Selection = pt.onAccent, pt.rowSelected, pt.selection
}

// WithPalette returns a new skin, s with the palette p: the same name and
// tokens but those p decides. s does not change; WithPalette(nil) returns s.
//
//	skin.Use(skin.Calm.WithPalette(skin.Indigo))
func (s *Skin) WithPalette(p *Palette) *Skin {
	if p == nil {
		return s
	}
	c := *s
	c.palette = p
	p.light.apply(&c.light)
	p.dark.apply(&c.dark)
	return &c
}

// Palette returns the skin's palette, as Cham for Calm.
func (s *Skin) Palette() *Palette { return s.palette }

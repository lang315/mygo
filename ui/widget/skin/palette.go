package skin

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Palette is the accent of a skin: the accent family, the Info family of
// the running state, the selected row and the text selection, in light and
// dark. Every palette MyGo ships passes the skin's contrast floors and keeps
// its Info apart from the status colors, also for a red-green color-blind
// eye. A skin takes one with WithPalette:
//
//	skin.Use(skin.Calm.WithPalette(skin.Teal))
type Palette struct {
	name        string
	light, dark paletteTokens
}

// paletteTokens are the tokens a palette decides in one appearance.
type paletteTokens struct {
	accent, info           skinhook.Family
	rowSelected, selection ui.Color
}

// Name returns the palette's name, as "Teal".
func (p *Palette) Name() string { return p.name }

// The palettes: Chàm, a blue-violet indigo dye; Indigo, a brighter
// violet-blue; Ink, a deep blue; Teal, whose Info is a blue of its own,
// since its accent sits too near green; Blue, Calm's own, the accent of
// GTK's Adwaita, GNOME's blue 4 in light, since white on Adwaita's own
// #3584e4 is under 4.5:1.
var (
	Cham = NewPalette("Chàm",
		LightFamily(ui.Hex("#4e3bb0"), ui.Hex("#45349b"), ui.Hex("#3b2d86"), ui.Hex("#46359e")),
		DarkFamily(ui.Hex("#9384f0"), ui.Hex("#a79bf3"), ui.Hex("#7f6eed"), ui.Hex("#bdb4f9"), ui.Hex("#0e0b2a")))
	Indigo = NewPalette("Indigo",
		LightFamily(ui.Hex("#4f46e5"), ui.Hex("#4338ca"), ui.Hex("#3730a3"), ui.Hex("#3f37c9")),
		DarkFamily(ui.Hex("#8b8ff9"), ui.Hex("#a3a6fb"), ui.Hex("#7477ef"), ui.Hex("#b9bcfc"), ui.Hex("#0e0f2a")))
	Ink = NewPalette("Ink",
		LightFamily(ui.Hex("#2c4f8c"), ui.Hex("#26457b"), ui.Hex("#203b69"), ui.Hex("#284778")),
		DarkFamily(ui.Hex("#7b9fdc"), ui.Hex("#93b1e4"), ui.Hex("#6a8dcf"), ui.Hex("#a9c1ea"), ui.Hex("#0b1220")))
	Teal = NewPalette("Teal",
		LightFamily(ui.Hex("#0f766e"), ui.Hex("#0c665f"), ui.Hex("#0a5650"), ui.Hex("#0b5d56")),
		DarkFamily(ui.Hex("#2bb5a3"), ui.Hex("#45c4b3"), ui.Hex("#20a090"), ui.Hex("#7fdccf"), ui.Hex("#06201c"))).
		withInfo(LightFamily(ui.Hex("#3d5f8f"), ui.Hex("#36547e"), ui.Hex("#2e486d"), ui.Hex("#34507a")),
			DarkFamily(ui.Hex("#8c9fe0"), ui.Hex("#9dade5"), ui.Hex("#7e8fca"), ui.Hex("#b6c2f2"), ui.Hex("#0b1220")))
	Blue = NewPalette("Blue",
		LightFamily(ui.Hex("#1c71d8"), ui.Hex("#1a5fb4"), ui.Hex("#174f96"), ui.Hex("#1a5fb4")),
		DarkFamily(ui.Hex("#78aeed"), ui.Hex("#92bdf1"), ui.Hex("#5f9be6"), ui.Hex("#a8cbf5"), ui.Hex("#0b1220")))
)

var white = ui.Hex("#ffffff")

// LightFamily is a family of the light appearance from its base, hover,
// pressed and text colors: white on the base, and the subtle fill and border
// the base mixed with white at 92% and 70%. On is always white: for a light
// base, set the returned family's On to a dark color (Check reports white on
// it as "Accent On on Base").
func LightFamily(base, hover, pressed, text ui.Color) Family {
	return Family{Base: base, Hover: hover, Pressed: pressed,
		Subtle: base.Mix(white, 0.92), Border: base.Mix(white, 0.70), Text: text, On: white}
}

// DarkFamily is a family of the dark appearance from its base, hover,
// pressed and text colors and the color on the base: the subtle fill and
// border the base at 14% and 35% alpha.
func DarkFamily(base, hover, pressed, text, on ui.Color) Family {
	return Family{Base: base, Hover: hover, Pressed: pressed,
		Subtle: base.Alpha(0.14), Border: base.Alpha(0.35), Text: text, On: on}
}

// NewPalette returns a palette of the accent families light and dark, whose
// Info is its accent: the selected row and the text selection are the light
// base mixed with white at 82% and 68%, and the dark base at 16% and 35%
// alpha. The light selection stands 1.2:1 from the row, for text selected in
// a cell edited in place. Skin.Check measures a skin with it.
//
//	skin.Use(skin.Calm.WithPalette(skin.NewPalette("Brand", light, dark)))
//
// An accent near the status colors makes an Info near them too, which
// Check reports as "Info Text near … Text". Such a skin gives Info a family
// of its own through its tokens:
//
//	s := skin.Calm.WithPalette(brand)
//	l, d := s.Tokens(false), s.Tokens(true)
//	l.Info, d.Info = lightInfo, darkInfo
//	skin.Use(skin.New("Brand", l, d))
func NewPalette(name string, light, dark Family) *Palette {
	return &Palette{name: name,
		light: paletteTokens{accent: light, info: light,
			rowSelected: light.Base.Mix(white, 0.82), selection: light.Base.Mix(white, 0.68)},
		dark: paletteTokens{accent: dark, info: dark,
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
	t.RowSelected, t.Selection = pt.rowSelected, pt.selection
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

// Palette returns the skin's palette, as Blue for Calm. It is nil for a skin
// made with New until WithPalette gives it one.
func (s *Skin) Palette() *Palette { return s.palette }

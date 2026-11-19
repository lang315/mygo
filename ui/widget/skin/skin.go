// Package skin gives the controls of package widget one look, set by tokens:
// colors, metrics, radii, type, shadows, the focus ring and the scroll bars,
// in light and dark. An
// app turns a skin on with one line at startup,
//
//	skin.Use(skin.Calm)
//
// and every window takes it in its next frame. An app that does not import
// the package keeps the plain look and links none of it.
//
// A brand color and a tighter density take a line too: WithAccent makes a
// palette from one color, moving it only as much as the contrast floors
// need, and WithDensity(Compact) sets controls lower, at Use:
//
//	skin.Use(skin.Calm.WithAccent(ui.Hex("#7c3aed")).WithDensity(skin.Compact))
//
// An app makes a skin of its own from another's tokens, changing those it
// wants, and an accent of its own with a palette; a font of its own comes in
// with ui.RegisterFont:
//
//	l, d := skin.Calm.Tokens(false), skin.Calm.Tokens(true)
//	l.Background, l.RowHeight = ui.Hex("#fffaf0"), 28
//	ui.RegisterFont(interTTF, "Inter")
//	l.Font, d.Font = "Inter", "Inter"
//	brand := skin.NewPalette("Brand",
//		skin.LightFamily(ui.Hex("#7c3aed"), ui.Hex("#6d28d9"), ui.Hex("#5b21b6"), ui.Hex("#6d28d9")),
//		skin.DarkFamily(ui.Hex("#a78bfa"), ui.Hex("#c4b5fd"), ui.Hex("#8b5cf6"), ui.Hex("#ddd6fe"), ui.Hex("#1e1033")))
//	s := skin.New("Brand", l, d).WithPalette(brand)
//	if err := s.Check(); err != nil {
//		log.Print(err) // the pairs under the contrast floors; the skin still works
//	}
//	skin.Use(s)
package skin

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Tokens are every value a skin decides in one appearance: colors, sizes,
// radii, type, shadows, the focus ring and the scroll bars, and the font.
type Tokens = skinhook.Tokens

// Family is a semantic role's colors: the accent, Info and the status colors.
type Family = skinhook.Family

// Type is a role of text: its size and weight.
type Type = skinhook.Type

// Shadow is a drop shadow: a card's or a drop-down's.
type Shadow = skinhook.Shadow

// New returns a skin of the tokens light and dark, which it copies. It does
// not check them: Check does.
//
//	skin.Use(skin.New("Brand", light, dark))
func New(name string, light, dark Tokens) *Skin {
	return &Skin{name: name, light: light, dark: dark}
}

// Tokens returns a copy of the skin's tokens in the dark or the light
// appearance, to change and make a skin of with New. Its sizes are the
// Standard ones, whatever the skin's Density: give the new skin
// WithDensity again.
func (s *Skin) Tokens(dark bool) Tokens {
	if dark {
		return s.dark
	}
	return s.light
}

// Check measures the skin's contrast in both appearances, as the skins MyGo
// ships pass it: text 4.5:1, outlines and chart colors 3:1, the control
// states apart, and Info apart from the status colors. The error names the
// appearance and every failing pair; nil means all pass. A skin that fails
// still works: Use does not check.
func (s *Skin) Check() error {
	var errs []error
	if err := check(&s.light, false); err != nil {
		errs = append(errs, fmt.Errorf("light: %w", err))
	}
	if err := check(&s.dark, true); err != nil {
		errs = append(errs, fmt.Errorf("dark: %w", err))
	}
	return errors.Join(errs...)
}

// Skin is a look: a name, its palette and its tokens in the light and the
// dark appearance.
type Skin struct {
	name        string
	palette     *Palette
	palettes    []*Palette
	density     Density
	light, dark skinhook.Tokens
	looks       Looks
	parsed      *[skinhook.NumKinds]any // looks' Styles, as Use hands them to widget
}

// Name returns the skin's name, as "Calm".
func (s *Skin) Name() string { return s.name }

var active atomic.Pointer[Skin]

// Use gives every window the skin s, in light and dark, from its next frame:
// the one line an app calls, at startup as a rule, though it may change the
// skin while it runs. Use(nil) brings back the plain look. It does not
// change the appearance, which widget.SetTheme chooses, and a view that sets
// its own theme keeps it. Call it from any goroutine.
//
//	skin.Use(skin.Calm)
func Use(s *Skin) {
	active.Store(s)
	if s == nil {
		skinhook.Set(nil, nil)
		skinhook.SetLooks(nil)
		ui.SetDefaultThemes(nil, nil)
		return
	}
	// The hook first and the themes last: the generation the themes bump
	// tells kept rows that the tokens changed too.
	l, d := &s.light, &s.dark
	if s.density == Compact {
		cl, cd := compact(s.light), compact(s.dark)
		l, d = &cl, &cd
	}
	skinhook.Set(l, d)
	skinhook.SetLooks(s.parsed)
	ui.SetDefaultThemes(themeOf(l, false), themeOf(d, true))
}

// Active returns the skin Use set, or nil.
func Active() *Skin { return active.Load() }

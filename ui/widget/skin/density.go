package skin

import (
	"math"

	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Density is how tight a skin sets its controls: Standard, its own sizes, or
// Compact, for screens full of data.
type Density uint8

const (
	Standard Density = iota // the skin's own sizes
	Compact                 // controls 6 DIPs lower, rows 4
)

// WithDensity returns a new skin, s with the density d: Compact makes
// controls and tab headers 6 DIPs lower, rows, headers and the menu bar 4, a
// settings row 8, and the tool strip's room as much smaller as its
// controls, never a height under the font's size and 8; icons, check boxes,
// switches, corners and text keep their size. s does not change.
//
// The sizes change at Use: Tokens stays Standard, so a skin made with New
// from a compact skin's tokens is Standard until it is given Compact again.
//
//	skin.Use(skin.Calm.WithDensity(skin.Compact))
func (s *Skin) WithDensity(d Density) *Skin {
	c := *s
	c.density = d
	return &c
}

// Density returns the skin's density, Standard unless WithDensity set one.
func (s *Skin) Density() Density { return s.density }

// compact returns the tokens t at the Compact density.
func compact(t skinhook.Tokens) skinhook.Tokens {
	h := func(v *float32, by float32) {
		if *v > 0 {
			// Never under FontSize + 8, nor taller than it was.
			*v = max(*v-by, min(*v, t.FontSize+8))
		}
	}
	ch := t.ControlHeight
	h(&t.ControlHeight, 6)
	h(&t.TabHeight, 6)
	h(&t.RowHeight, 4)
	h(&t.HeaderHeight, 4)
	h(&t.MenuBarStripHeight, 4)
	h(&t.MenuBarItemHeight, 4)
	h(&t.SettingsRowHeight, 8)
	if ch > 0 {
		r := t.ControlHeight / ch
		for _, v := range []*float32{&t.ToolbarPadTop, &t.ToolbarPadBottom, &t.ToolbarGap, &t.ToolbarGroupGap} {
			*v = float32(math.Round(float64(*v * r)))
		}
	}
	return t
}

package skin

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Builtin returns the skins MyGo ships, Calm, Classic and HighContrast, in
// that order, in a new slice each call. It builds none of them: a skin is
// made at its first use.
func Builtin() []*Skin { return []*Skin{Calm, Classic, HighContrast} }

// ID names the skin, its palette and its density, as
// "Calm/Teal/compact": the palette by its name, a WithAccent one by the
// color given as "#rrggbb", and nothing for a skin without a palette
// ("HighContrast//standard"). An app keeps it in its settings and gets the
// skin back with FromID. Looks are not in it: they are the skin's of that
// name.
func (s *Skin) ID() string {
	s = s.get()
	p := ""
	if s.palette != nil {
		p = s.palette.name // a WithAccent palette is named by its color
	}
	d := "standard"
	if s.density == Compact {
		d = "compact"
	}
	return s.name + "/" + p + "/" + d
}

// FromID returns the skin the ID names (see Skin.ID): the first skin of
// that name in extra, the app's own, then in Builtin; with the palette of
// that name among its Palettes or its own, or WithAccent of a "#rrggbb"
// color, or its own palette when the part is empty; and the density. It
// returns nil when any part is unknown, for the app to fall back to its
// default:
//
//	if s := skin.FromID(saved, mine...); s != nil {
//		skin.Use(s)
//	}
func FromID(id string, extra ...*Skin) *Skin {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil
	}
	var s *Skin
	for _, list := range [][]*Skin{extra, Builtin()} {
		for _, x := range list {
			if x != nil && nameOf(x) == parts[0] {
				s = x
				break
			}
		}
		if s != nil {
			break
		}
	}
	if s == nil {
		return nil
	}
	s = s.get()
	switch p := parts[1]; {
	case p == "":
	case p[0] == '#':
		v, err := strconv.ParseUint(p[1:], 16, 32)
		if len(p) != 7 || err != nil {
			return nil
		}
		s = s.WithAccent(ui.RGB(uint8(v>>16), uint8(v>>8), uint8(v)))
	default:
		var found *Palette
		for _, x := range append(s.Palettes(), s.palette) {
			if x != nil && x.name == p {
				found = x
				break
			}
		}
		if found == nil {
			return nil
		}
		s = s.WithPalette(found)
	}
	switch parts[2] {
	case "standard":
		return s.WithDensity(Standard)
	case "compact":
		return s.WithDensity(Compact)
	}
	return nil
}

// nameOf is s's name, without building a lazy skin, whose literal holds its
// name too.
func nameOf(s *Skin) string {
	if s.lazy != nil && s.name != "" {
		return s.name
	}
	return s.Name()
}

// Package skin gives the controls of package widget one look, set by tokens:
// colors, metrics, radii, type, elevation and motion, in light and dark. An
// app turns a skin on with one line at startup,
//
//	skin.Use(skin.Calm)
//
// and every window takes it in its next frame. An app that does not import
// the package keeps the plain look and links none of it.
package skin

import "github.com/egoist/mygo/ui/widget/internal/skinhook"

// Skin is a look: a name and its tokens in the light and the dark
// appearance.
type Skin struct {
	name        string
	light, dark skinhook.Tokens
}

// Name returns the skin's name, as "Calm".
func (s *Skin) Name() string { return s.name }

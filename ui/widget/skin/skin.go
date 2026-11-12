// Package skin gives the controls of package widget one look, set by tokens:
// colors, metrics, radii, type, elevation and motion, in light and dark. An
// app turns a skin on with one line at startup,
//
//	skin.Use(skin.Calm)
//
// and every window takes it in its next frame. An app that does not import
// the package keeps the plain look and links none of it.
package skin

import (
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Skin is a look: a name, its palette and its tokens in the light and the
// dark appearance.
type Skin struct {
	name        string
	palette     *Palette
	light, dark skinhook.Tokens
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
		ui.SetDefaultThemes(nil, nil)
		return
	}
	// The hook first and the themes last: the generation the themes bump
	// tells kept rows that the tokens changed too.
	skinhook.Set(&s.light, &s.dark)
	ui.SetDefaultThemes(themeOf(&s.light, false), themeOf(&s.dark, true))
}

// Active returns the skin Use set, or nil.
func Active() *Skin { return active.Load() }

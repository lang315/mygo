// Package skinhook holds the tokens of the active skin, light and dark,
// which package skin sets (skin.Use) and package widget reads: an
// unexported variable of widget cannot be set from another package.
//
// A reader reads through Get while it builds or paints a frame, and never
// keeps a value past that frame unless it keys it by the theme's
// generation: skin.Use ends with ui.SetDefaultThemes, whose generation bump
// is what tells kept rows and caches that the tokens changed. Get returns
// nil when no skin is on, and the reader keeps its plain look.
package skinhook

import (
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// Family is a semantic role: its base fill, the fill hovered and pressed,
// the subtle fill and its border, text on the subtle fill and text on the
// base.
type Family struct{ Base, Hover, Pressed, Subtle, Border, Text, On ui.Color }

// Type is a role of text: its size in DIPs and its weight (100 to 900).
// The text keeps its own line height.
type Type struct {
	Size   float32
	Weight int
}

// Shadow is a drop shadow: its offset down, blur, spread and color.
type Shadow struct {
	Y, Blur, Spread float32
	Color           ui.Color
}

// Tokens are every value a skin decides in one appearance.
type Tokens struct {
	// The layers: Chrome under the frame (header, menus, tool and status
	// strips), Background under the work area.
	Chrome, Background ui.Color
	// Border divides; BorderStrong ends headers and the window.
	Border, BorderStrong ui.Color
	// Text, TextMuted (secondary) and TextPlaceholder (placeholders), which
	// takes no effect yet: no control reads it.
	Text, TextMuted, TextPlaceholder ui.Color
	// The face of a control at rest, hovered and pressed.
	Surface, SurfaceHover, SurfacePressed ui.Color
	// ButtonBorder outlines buttons; ControlBorder inputs, check boxes and a
	// switch that is off.
	ButtonBorder, ControlBorder ui.Color
	// Selection is selected text; RowHover and RowSelected rows.
	Selection, RowHover, RowSelected ui.Color

	// The families. Of Accent, Subtle and Text take no effect yet; of Info,
	// Warning and Danger only Base and On take effect, and of Success only
	// Base: no control reads the others.
	Accent, Success, Warning, Danger Family
	// Info is the running and info state, kept apart from the status
	// colors.
	Info Family
	// Chart colors the series and slices of charts, in order.
	Chart [8]ui.Color

	// The sizes, in DIPs. SettingsRowHeight and Icon take no effect yet: no
	// control reads them.
	ControlHeight, RowHeight, HeaderHeight, SettingsRowHeight    float32
	TabHeight, MenuBarStripHeight, MenuBarItemHeight             float32
	RadiusControl, RadiusCard                                    float32
	ToolbarPadTop, ToolbarPadBottom, ToolbarGap, ToolbarGroupGap float32
	Icon, ToolbarIcon, CheckSize, SwitchWidth, SwitchHeight      float32

	// FontSize is the size of text, in DIPs.
	FontSize float32
	// The roles of text. They take no effect yet: no control reads them.
	Caption, Label, Heading, Title Type
	// Font is the family of text, "" the system's; a font of the app's own
	// comes in with ui.RegisterFont.
	Font string

	// CardShadow is under a card. PopupShadow takes no effect yet: no
	// control reads it.
	CardShadow, PopupShadow Shadow

	// The focus ring is FocusRingWidth DIPs wide (0 is ui's default, 2) and
	// FocusRingOffset DIPs outside the control's edge (0 puts it on the
	// edge). Scroll bar thumbs are ScrollbarWidth DIPs wide (0 is ui's
	// default, 6), and ScrollbarStyle styles the bars where it sets a field
	// (ui.ScrollbarStyle).
	FocusRingWidth, FocusRingOffset, ScrollbarWidth float32
	ScrollbarStyle                                  ui.ScrollbarStyle
}

var active atomic.Pointer[[2]*Tokens]

// Set makes light and dark the active tokens; nil, nil turns the skin off.
// Call it from any goroutine.
func Set(light, dark *Tokens) {
	if light == nil && dark == nil {
		active.Store(nil)
		return
	}
	active.Store(&[2]*Tokens{light, dark})
}

// Get returns the active tokens of the dark or the light appearance, or
// nil when no skin is on.
func Get(dark bool) *Tokens {
	p := active.Load()
	if p == nil {
		return nil
	}
	if dark {
		return p[1]
	}
	return p[0]
}

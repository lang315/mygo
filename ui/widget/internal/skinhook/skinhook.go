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
	"time"

	"github.com/egoist/mygo/ui"
)

// Family is a semantic role: its base fill, the fill hovered and pressed,
// the subtle fill and its border, text on the subtle fill and text on the
// base.
type Family struct{ Base, Hover, Pressed, Subtle, Border, Text, On ui.Color }

// Type is a role of text: its size, line height and weight.
type Type struct {
	Size, Line float32
	Weight     int
}

// Shadow is a drop shadow: its offset down, blur, spread and color.
type Shadow struct {
	Y, Blur, Spread float32
	Color           ui.Color
}

// Tokens are every value a skin decides in one appearance.
type Tokens struct {
	// The layers: Chrome under the frame (header, menus, tool and status
	// strips), Content under the work area.
	Chrome, Content ui.Color
	// Line divides; LineStrong ends headers and the window.
	Line, LineStrong ui.Color
	// Text, Text2 (secondary) and Text3 (placeholders).
	Text, Text2, Text3 ui.Color
	// The face of a control at rest, hovered and pressed.
	Control, ControlHover, ControlPressed ui.Color
	// ButtonLine outlines buttons; FieldLine inputs, check boxes and a
	// switch that is off.
	ButtonLine, FieldLine ui.Color
	OnAccent              ui.Color
	// Selection is selected text; RowHover and RowSelected rows.
	Selection, RowHover, RowSelected ui.Color

	Accent, Success, Warning, Danger Family
	// Info is the running and info state, kept apart from the status
	// colors.
	Info Family
	// Chart colors the series and slices of charts, in order.
	Chart [8]ui.Color

	ControlHeight, RowHeight, HeaderHeight, SettingsRowHeight    float32
	TabStripHeight, MenuBarStripHeight, MenuBarItemHeight        float32
	RadiusControl, RadiusCard, RadiusDialog                      float32
	ToolbarPadTop, ToolbarPadBottom, ToolbarGap, ToolbarGroupGap float32
	Icon, ToolbarIcon, CheckBox, SwitchWidth, SwitchHeight       float32

	Body, Caption, Label, Heading, Title Type

	// Elevation is 0 flat, 1 a card, 2 a popover, 3 a dialog.
	Elevation [4]Shadow

	// Hover is how long a hover or press change takes, Toggle a switch's.
	Hover, Toggle time.Duration
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

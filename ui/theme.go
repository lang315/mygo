package ui

import (
	"runtime"
	"sync/atomic"
	"time"
)

// Theme holds the colors and metrics widgets use. Change a copy of
// LightTheme or DarkTheme and set it with Context.SetTheme.
type Theme struct {
	Dark bool
	// Background fills the window.
	Background Color
	// Surface is the face of buttons, inputs and other controls;
	// SurfaceHover and SurfacePressed while hovered or pressed.
	Surface        Color
	SurfaceHover   Color
	SurfacePressed Color
	// Field is the fill of text inputs and what looks like one: TextBox,
	// the editable combo box, the number, date and time fields. It is not
	// Surface, the face of buttons, so that a field is not read as a
	// button: light, it is the window's own color, and dark darker than
	// the window. Left zero, as in a Theme written before it, fields take
	// Surface (FieldFill). A Theme copied from LightTheme or DarkTheme that
	// changes Surface or Background keeps their Field: set it too, or zero
	// it for fields to follow Surface.
	Field Color
	// Border draws the lines that divide and hold: dividers, grid lines,
	// the edges of containers and popups.
	Border Color
	// ControlBorder is the boundary of an interactive control, which is what
	// identifies it: the outline of an input, a combo box, a button, a
	// toggle. It is the one a user must see against the window and the face
	// of the control (WCAG 1.4.11 asks 3:1), so it is darker than Border.
	// Left zero, as in a Theme written before it, controls take Border
	// (Outline). A Theme copied from LightTheme or DarkTheme that changes
	// only Border keeps their ControlBorder: set it too, or zero it for the
	// controls to follow Border.
	ControlBorder Color
	// Text and TextMuted color text; TextMuted is for secondary text and
	// placeholders.
	Text      Color
	TextMuted Color
	// Accent colors primary buttons, checked controls and focus rings;
	// AccentText is text on it.
	Accent        Color
	AccentHover   Color
	AccentPressed Color
	AccentText    Color
	// InactiveAccent is what the accent fades to in a window that is not
	// drawn as active (Context.WindowActive), as AppKit fades an inactive
	// window's: the fill of selected rows, of checked check boxes, radio
	// buttons and switches, of sliders and progress bars, the line of the
	// selected tab and the icons of a sidebar, with AccentText on it as on
	// the accent; primary buttons show as the others there. Left zero, it
	// is a gray AccentText stands out on; set it to Accent to keep the
	// accent in every window.
	InactiveAccent Color
	Danger         Color
	// Warning and Success color what calls for attention and what went
	// well, as a Meter's levels.
	Warning Color
	Success Color
	// Selection highlights selected text.
	Selection Color
	// Focus is the ring around the control with the keyboard focus,
	// FocusRingWidth DIPs wide (2 by default; 0 is 2) and FocusRingOffset
	// DIPs outside its edge (1 in LightTheme and DarkTheme): 0 puts the
	// ring on the edge, and a negative offset inside it.
	// Element.FocusRingOffset and Element.FocusRingColor change them for
	// one element.
	Focus           Color
	FocusRingWidth  float32
	FocusRingOffset float32
	// Inverse fills tooltips and toasts, the theme turned over so that
	// they stand out from what they are over: dark in a light theme, light
	// in a dark one. InverseText is text on it. Left zero, they are Text
	// and Background.
	Inverse     Color
	InverseText Color
	// Scrollbar colors scroll bar thumbs, ScrollbarWidth DIPs wide (6 by
	// default; 0 is 6).
	Scrollbar      Color
	ScrollbarWidth float32
	// Radius rounds the corners of controls.
	Radius float32
	// Spacing is the unit of the room widgets leave inside and between
	// their parts: their paddings and gaps, and the sizes of check boxes,
	// switches, sliders and the rows of tables and trees, are multiples
	// of it. 4 by default: 3 makes widgets compact, 5 roomy; 0 is 4.
	Spacing float32
	// FontSize is the size of text, Font its family ("" is the system's).
	FontSize float32
	Font     string
	// TooltipDelay is how long the pointer rests on an element before its
	// tooltip shows (Tooltip, TooltipBase): 0 is 0.6 seconds, and a
	// negative delay shows it at once. Within two thirds of it after
	// another tip goes, as the pointer moves along a toolbar, the next one
	// shows at once: 0.4 seconds by default.
	TooltipDelay time.Duration
}

// Space returns n units of the theme's Spacing, to size elements of your
// own in step with the widgets: ui.Column(c).Gap(t.Space(2)).
func (t *Theme) Space(n float32) float32 {
	s := t.Spacing
	if s <= 0 {
		s = 4
	}
	return n * s
}

// inactiveAccent returns what the accent fades to in a window that is not
// drawn as active (InactiveAccent).
func (t *Theme) inactiveAccent() Color {
	if t.InactiveAccent != Transparent {
		return t.InactiveAccent
	}
	// As far from AccentText as text on it needs, 4.5 times its contrast.
	far := Color{A: 255}
	if t.AccentText.gray().R <= 127 {
		far = Color{R: 255, G: 255, B: 255, A: 255}
	}
	return t.AccentText.Mix(far, 0.55)
}

// inverse returns the fill of tooltips and toasts, and the color of text
// on it.
func (t *Theme) inverse() (fill, text Color) {
	fill, text = t.Inverse, t.InverseText
	if fill == Transparent {
		fill = t.Text
	}
	if text == Transparent {
		text = t.Background
	}
	return fill, text
}

// focusRingWidth returns how wide the focus ring is (FocusRingWidth).
func (t *Theme) focusRingWidth() float32 {
	if t.FocusRingWidth <= 0 {
		return 2
	}
	return t.FocusRingWidth
}

func (t *Theme) scrollbarWidth() float32 {
	if t.ScrollbarWidth <= 0 {
		return 6
	}
	return t.ScrollbarWidth
}

// tooltipDelay returns how long the pointer rests on an element before its
// tip shows (TooltipDelay), and how soon after a tip goes the next one
// shows at once.
func (t *Theme) tooltipDelay() (delay, warm time.Duration) {
	switch d := t.TooltipDelay; {
	case d == 0:
		return tooltipDelay, tooltipWarm
	case d < 0:
		return 0, 0
	default:
		return d, d * 2 / 3
	}
}

// Rem returns n times the theme's FontSize, as CSS's rem, to size elements
// with the text.
func (t *Theme) Rem(n float32) float32 {
	if t.FontSize <= 0 {
		return n * defaultFontSize()
	}
	return n * t.FontSize
}

func defaultFontSize() float32 {
	if runtime.GOOS == "darwin" {
		return 13
	}
	return 14
}

// LightTheme returns the theme for a light appearance.
func LightTheme() *Theme {
	return &Theme{
		Background:      Hex("#ffffff"),
		Surface:         Hex("#f4f4f5"),
		Field:           Hex("#ffffff"), // 1.10:1 against Surface, 3.42:1 against its ControlBorder
		SurfaceHover:    Hex("#e9e9ec"),
		SurfacePressed:  Hex("#dddde1"),
		Border:          Hex("#d9d9de"),
		ControlBorder:   Hex("#8a8a93"), // 3.42:1 on Background, 3.11:1 on Surface
		Text:            Hex("#18181b"),
		TextMuted:       Hex("#71717a"),
		Accent:          Hex("#2563eb"),
		AccentHover:     Hex("#1d4ed8"),
		AccentPressed:   Hex("#1e40af"),
		AccentText:      Hex("#ffffff"),
		Danger:          Hex("#dc2626"),
		Warning:         Hex("#d97706"),
		Success:         Hex("#16a34a"),
		Selection:       RGBA(37, 99, 235, 0.25),
		Focus:           RGBA(37, 99, 235, 0.8), // 3.62:1 on Background
		FocusRingWidth:  2,
		FocusRingOffset: 1,
		Scrollbar:       RGBA(0, 0, 0, 0.32),
		Radius:          6,
		Spacing:         4,
		FontSize:        defaultFontSize(),
	}
}

// DarkTheme returns the theme for a dark appearance.
func DarkTheme() *Theme {
	return &Theme{
		Dark:            true,
		Background:      Hex("#18181b"),
		Surface:         Hex("#27272a"),
		Field:           Hex("#101012"), // 1.28:1 against Surface, 3.93:1 against its ControlBorder
		SurfaceHover:    Hex("#323236"),
		SurfacePressed:  Hex("#3c3c41"),
		Border:          Hex("#3f3f46"),
		ControlBorder:   Hex("#71717a"), // 3.67:1 on Background, 3.08:1 on Surface
		Text:            Hex("#f4f4f5"),
		TextMuted:       Hex("#a1a1aa"),
		Accent:          Hex("#3b82f6"),
		AccentHover:     Hex("#60a5fa"),
		AccentPressed:   Hex("#2563eb"),
		AccentText:      Hex("#ffffff"),
		Danger:          Hex("#ef4444"),
		Warning:         Hex("#f59e0b"),
		Success:         Hex("#22c55e"),
		Selection:       RGBA(59, 130, 246, 0.4),
		Focus:           RGBA(96, 165, 250, 0.6),
		FocusRingWidth:  2,
		FocusRingOffset: 1,
		Scrollbar:       RGBA(255, 255, 255, 0.35),
		Radius:          6,
		Spacing:         4,
		FontSize:        defaultFontSize(),
	}
}

// FieldFill returns the fill of text inputs: Field, or Surface in a Theme
// that sets none.
func (t *Theme) FieldFill() Color {
	if t.Field.A != 0 {
		return t.Field
	}
	return t.Surface
}

// Outline returns the color of the boundary of a control: ControlBorder, or
// Border in a Theme that sets none.
func (t *Theme) Outline() Color {
	if t.ControlBorder.A != 0 {
		return t.ControlBorder
	}
	return t.Border
}

// controlBorder are the colors SetControlBorder set, light then dark, and
// borderGen counts its calls, which a window takes up as it builds.
var (
	controlBorder atomic.Pointer[[2]Color]
	borderGen     atomic.Uint64
)

// SetControlBorder sets Theme.ControlBorder of the default theme of each
// appearance, light and dark: for an app that has no Context to set a Theme
// with, as one of ui/widget's, which draws the outline of every control in
// it. A color that is zero leaves the theme's own for that
// appearance. The windows follow the appearance as before, and a window that
// is open takes the colors in its next frame. A Theme a view sets itself
// (Context.SetTheme) is its own: this changes the defaults only. Call it
// from any goroutine, once at startup as a rule.
//
//	ui.SetControlBorder(ui.Hex("#6b7280"), ui.Hex("#9ca3af"))
func SetControlBorder(light, dark Color) {
	controlBorder.Store(&[2]Color{light, dark})
	borderGen.Add(1)
}

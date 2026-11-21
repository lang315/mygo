// Package skinhook holds the tokens of the active skin, light and dark,
// and its looks, which package skin sets (skin.Use) and package widget
// reads: an unexported variable of widget cannot be set from another
// package.
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
	// Text, TextMuted (secondary) and TextPlaceholder, the placeholder of a
	// TextBox and a MultilineTextBox.
	Text, TextMuted, TextPlaceholder ui.Color
	// The face of a control at rest, hovered and pressed.
	Surface, SurfaceHover, SurfacePressed ui.Color
	// ButtonBorder outlines buttons; ControlBorder inputs, check boxes and a
	// switch that is off.
	ButtonBorder, ControlBorder ui.Color
	// Selection is selected text; RowHover and RowSelected rows.
	Selection, RowHover, RowSelected ui.Color

	// The families. Of Accent, Subtle and Text take no effect yet. Badges and
	// info bars draw Info's, Success's, Warning's and Danger's Subtle (their
	// fill), Text, Border and Base (a running dot, an info bar's icon), and
	// buttons and the rest Base and On.
	Accent, Success, Warning, Danger Family
	// Info is the running and info state, kept apart from the status
	// colors.
	Info Family
	// Chart colors the series and slices of charts, in order.
	Chart [8]ui.Color

	// The sizes, in DIPs at text scale 1: the heights of controls, of table
	// and list rows, of their headers, of the rows of a settings page
	// (Fields.SettingsRows, the least), of tab headers, of the menu bar's
	// strip and its items; the corners of controls and of cards; the room
	// of a tool strip above and below its buttons, between them and between
	// their groups; the side of the icons of controls and lists (a
	// TextBox's, a DataGridView cell's, a ListBox's, a TreeView's, a
	// ListView's small ones and a ComboBox section's) and of tool strip
	// buttons; the check box's side and the switch's track.
	ControlHeight, RowHeight, HeaderHeight, SettingsRowHeight    float32
	TabHeight, MenuBarStripHeight, MenuBarItemHeight             float32
	RadiusControl, RadiusCard                                    float32
	ToolbarPadTop, ToolbarPadBottom, ToolbarGap, ToolbarGroupGap float32
	Icon, ToolbarIcon, CheckSize, SwitchWidth, SwitchHeight      float32

	// FontSize is the size of text, in DIPs.
	FontSize float32
	// The roles of text: Caption a status label's, Label a group box's
	// title and a DataGridView's column headers (their size: ui keeps their
	// weight), Heading the section headings of Fields and Title a dialog's
	// title. The app's own size or weight on the text wins.
	Caption, Label, Heading, Title Type
	// Font is the family of text, "" the system's; a font of the app's own
	// comes in with ui.RegisterFont.
	Font string

	// CardShadow is under a card, PopupShadow under the drop-downs of
	// ui/widget (a ComboBox's); ui's own popup menus keep theirs. A shadow of
	// zero color draws none.
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

// Kind is a kind of control or part a skin gives a look, one for each field
// of skin.Looks, in its order.
type Kind uint8

const (
	KindButton Kind = iota
	KindToolButton
	KindField
	KindComboBox
	KindCheckBox
	KindRadioButton
	KindGroupBox
	KindCard
	KindTabStrip
	KindTabHeader
	KindGridHeader
	KindToolStrip
	KindStatusStrip
	KindMenuBar
	KindHeaderBar
	KindBadge
	KindInfoBar
	KindEmptyState
	KindSettingsCard
	NumKinds
)

var looks atomic.Pointer[[NumKinds]any]

// SetLooks makes l the active looks, each a *widget.Style or nil for no
// look of its kind; nil turns them all off. The skin boxes them once, so
// that reading one allocates nothing. Call it from any goroutine.
func SetLooks(l *[NumKinds]any) { looks.Store(l) }

// Look returns the active look of kind k, nil for none.
func Look(k Kind) any {
	if l := looks.Load(); l != nil && k < NumKinds {
		return l[k]
	}
	return nil
}

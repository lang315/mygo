package widget

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Clipboard is the system clipboard.
var Clipboard clipboard

type clipboard struct{}

// SetText puts s on the clipboard.
func (clipboard) SetText(s string) { mygo.Clipboard.WriteText(s) }

// SetImage puts a PNG image on the clipboard.
func (clipboard) SetImage(png []byte) error { return mygo.Clipboard.WriteImage(png) }

// Text returns the text on the clipboard, "" for none.
func (clipboard) Text() string { return mygo.Clipboard.ReadText() }

// Theme is the app's appearance.
type Theme = mygo.ThemeSource

const (
	ThemeSystem = mygo.ThemeSystem // follow the desktop
	ThemeLight  = mygo.ThemeLight
	ThemeDark   = mygo.ThemeDark
)

// SetTheme sets the app's appearance; every window follows.
func SetTheme(t Theme) { mygo.Theme.SetSource(t) }

// SetControlBorder sets the color of the outline of the controls (TextBox,
// ComboBox, NumericUpDown, buttons and the other inputs), in the light and the
// dark appearance. The theme's own is 3.4:1 against the window in the light
// appearance and 3.7:1 in the dark (WCAG 1.4.11 asks 3:1); dividers, grid
// lines and the edges of containers keep the theme's lighter border. A zero
// color leaves the theme's. Every window follows, as the appearance changes.
// Call it once at startup.
//
//	widget.SetControlBorder(ui.Hex("#6b7280"), ui.Hex("#9ca3af")) // gray-500, gray-400
func SetControlBorder(light, dark ui.Color) { ui.SetControlBorder(light, dark) }

// IsDark reports whether the app draws dark.
func IsDark() bool { return mygo.Theme.IsDark() }

// ScreenSize returns the size of the main display, in DIPs. Call it once
// the app runs (in Run's function or later).
func ScreenSize() (w, h float32) {
	b := mygo.Screen.PrimaryDisplay().Bounds
	return float32(b.Width), float32(b.Height)
}

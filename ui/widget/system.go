package widget

import "github.com/egoist/mygo"

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

// IsDark reports whether the app draws dark.
func IsDark() bool { return mygo.Theme.IsDark() }

// ScreenSize returns the size of the main display, in DIPs. Call it once
// the app runs (in Run's function or later).
func ScreenSize() (w, h float32) {
	b := mygo.Screen.PrimaryDisplay().Bounds
	return float32(b.Width), float32(b.Height)
}

// Package bridge embeds the renderer runtime that MyGo injects into pages.
//
// bridge.js is built from ../../packages/bridge by `bun run build` and
// committed, so building a MyGo application never requires Bun.
package bridge

import (
	_ "embed"
	"strconv"

	"github.com/egoist/mygo/internal/jsonlite"
	"github.com/egoist/mygo/internal/platform"
)

//go:embed bridge.js
var source string

// Config is passed to the bridge as __MYGO_CONFIG__.
type Config struct {
	Platform string `json:"platform"`
	WindowID int    `json:"windowId"`
	Version  string `json:"version"`
	// Secret prefixes every message the bridge posts.
	Secret string `json:"secret"`
	// Locale is the language the app shows, e.g. "zh-Hans", and Dir the
	// direction it is written in, "ltr" or "rtl", which pages read as
	// mygo.locale and mygo.dir from document start.
	Locale string `json:"locale"`
	Dir    string `json:"dir"`
	// Fit is the limits of a window that takes the size of its page's
	// content (FitContent), nil for other windows.
	Fit *FitContent `json:"fit,omitempty"`
}

// Script returns the bridge source configured for one window.
func Script(cfg Config) string {
	return "(function(__MYGO_CONFIG__){\n" + source + "})(" + string(cfg.JSON()) + ");"
}

// The JSON of the types below is written here, as encoding/json writes their
// fields (their tags), because the scripts and the events of every window say
// it, and a native app, which has windows and no pages, has no other use for
// encoding/json. TestJSONIsMarshal holds it to the same bytes.

func appendKey(b []byte, key string) []byte {
	b = jsonlite.AppendString(b, key)
	return append(b, ':')
}

func appendNumber(b []byte, f float64) []byte {
	if out, ok := jsonlite.AppendFloat(b, f); ok {
		return out
	}
	b = append(b, '0')
	return b
}

// JSON returns the configuration as JSON.
func (c Config) JSON() []byte {
	b := append([]byte(nil), `{"platform":`...)
	b = jsonlite.AppendString(b, c.Platform)
	b = append(b, `,"windowId":`...)
	b = strconv.AppendInt(b, int64(c.WindowID), 10)
	b = append(b, `,"version":`...)
	b = jsonlite.AppendString(b, c.Version)
	b = append(b, `,"secret":`...)
	b = jsonlite.AppendString(b, c.Secret)
	b = append(b, `,"locale":`...)
	b = jsonlite.AppendString(b, c.Locale)
	b = append(b, `,"dir":`...)
	b = jsonlite.AppendString(b, c.Dir)
	if c.Fit != nil {
		b = append(b, `,"fit":`...)
		b = c.Fit.AppendJSON(b)
	}
	return append(b, '}')
}

// AppendJSON appends the limits as JSON.
func (f FitContent) AppendJSON(b []byte) []byte {
	b = append(b, `{"minWidth":`...)
	b = appendNumber(b, f.MinWidth)
	b = append(b, `,"minHeight":`...)
	b = appendNumber(b, f.MinHeight)
	b = append(b, `,"maxWidth":`...)
	b = appendNumber(b, f.MaxWidth)
	b = append(b, `,"maxHeight":`...)
	b = appendNumber(b, f.MaxHeight)
	return append(b, '}')
}

// JSON returns the payload of LocaleEvent as JSON, written as encoding/json/v2
// writes it for the IPC (U+2028 and U+2029 escaped); false if a string is not
// valid UTF-8, which v2 does not write either.
func (l Locale) JSON() ([]byte, bool) {
	b, ok := jsonlite.AppendStringJS(append([]byte(nil), `{"locale":`...), l.Locale)
	if !ok {
		return nil, false
	}
	b = append(b, `,"dir":`...)
	if b, ok = jsonlite.AppendStringJS(b, l.Dir); !ok {
		return nil, false
	}
	return append(b, '}'), true
}

// JSON returns the payload of TitleBarEvent as JSON.
func (t TitleBar) JSON() []byte {
	b := append([]byte(nil), `{"height":`...)
	b = appendNumber(b, t.Height)
	b = append(b, `,"left":`...)
	b = appendNumber(b, t.Left)
	b = append(b, `,"right":`...)
	b = appendNumber(b, t.Right)
	return append(b, '}')
}

// TitleBarEvent tells a page of a window with a hidden title bar the room
// the window controls take. The bridge keeps it in the --mygo-titlebar-*
// CSS variables.
const TitleBarEvent = "mygo:title-bar"

// LocaleEvent tells the pages of a window that the language the app shows
// changed, with a Locale. The bridge configuration tells pages the
// language at document start; the event is sent as a page loads too, for
// one that started loading as the language changed, and the bridge tells
// its listeners only of a language the page did not have.
const LocaleEvent = "mygo:locale"

// Locale is the payload of LocaleEvent.
type Locale struct {
	Locale string `json:"locale"`
	Dir    string `json:"dir"`
}

// TitleBar is the payload of TitleBarEvent, in CSS pixels.
type TitleBar struct {
	Height float64 `json:"height"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
}

// NewTitleBar converts the room of the window controls to the CSS pixels
// of a page shown at zoom.
func NewTitleBar(tb platform.TitleBar, zoom float64) TitleBar {
	if zoom <= 0 {
		zoom = 1
	}
	return TitleBar{Height: float64(tb.Height) / zoom, Left: float64(tb.Left) / zoom, Right: float64(tb.Right) / zoom}
}

// FitContentEvent tells the page of a window that takes the size of its
// content (FitContent) the limits of that size, with a FitContent, when
// the page's zoom changed them. The bridge configuration tells them at
// document start.
const FitContentEvent = "mygo:fit-content"

// FitContent is the limits of the size of a window that takes its
// content's, in CSS pixels; 0 is none. The bridge lays the page's root
// element out within them.
type FitContent struct {
	MinWidth  float64 `json:"minWidth"`
	MinHeight float64 `json:"minHeight"`
	MaxWidth  float64 `json:"maxWidth"`
	MaxHeight float64 `json:"maxHeight"`
}

// NewFitContent converts the limits of a window's size, in DIPs, to the CSS
// pixels of a page shown at zoom.
func NewFitContent(minSize, maxSize platform.Size, zoom float64) FitContent {
	if zoom <= 0 {
		zoom = 1
	}
	return FitContent{
		MinWidth:  float64(minSize.Width) / zoom,
		MinHeight: float64(minSize.Height) / zoom,
		MaxWidth:  float64(maxSize.Width) / zoom,
		MaxHeight: float64(maxSize.Height) / zoom,
	}
}

// TitleBarScript delivers TitleBarEvent at document start, after the
// bridge, so that a page lays out around the window controls before it
// first paints.
func TitleBarScript(tb platform.TitleBar, zoom float64) string {
	p := NewTitleBar(tb, zoom).JSON()
	return `window.__mygo&&window.__mygo.receive({"t":"event","n":"` + TitleBarEvent + `","p":` + string(p) + `});`
}

// PreferencesEvent tells a page the accent color and the accessibility
// settings of the desktop, as mygo.ThemePreferences encodes them. The
// bridge keeps the latest for mygo-runtime's preferences(), the accent in
// the --mygo-accent CSS variable and the text scale in --mygo-text-scale.
const PreferencesEvent = "mygo:preferences"

// PreferencesScript delivers PreferencesEvent with the JSON prefs at
// document start, after the bridge has registered its handler.
func PreferencesScript(prefs []byte) string {
	return `window.__mygo&&window.__mygo.receive({"t":"event","n":"` + PreferencesEvent + `","p":` + string(prefs) + `});`
}

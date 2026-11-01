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
	// Locale is the language the app shows, e.g. "zh-Hans", and Dir the
	// direction it is written in, "ltr" or "rtl", which pages read as
	// mygo.locale and mygo.dir from document start.
	Locale string `json:"locale"`
	Dir    string `json:"dir"`
	// Fit is the limits of a window that takes the size of its page's
	// content (FitContent), nil for other windows.
	Fit *FitContent `json:"fit,omitempty"`
}

// HideHandlerScript is the user script, for every frame, that takes the message
// handler away from the frames that are not the page's own, on WebKitGTK and
// WebView2. It is defense in depth, not the barrier. WebKitGTK does not say which
// frame posted a message; there the barrier is that the page is never restored
// from the back/forward cache, so the bridge's hello comes before the page has a
// frame (enable-page-cache, see "Trust" in the architecture guide). WebView2
// raises its message event for the top-level document only. macOS rejects the
// messages of subframes natively.
//
// The script runs at document start, before any script of the frame, and
// removes the handler of MyGo only (webkit's messageHandlers.mygo,
// chrome.webview), so pages that look for other things on those objects keep
// them. It does not run in the initial empty document of a frame (WebKit gives
// those no user script), so a frame that makes a child with a source that has
// not committed can still see the handler there. The bridge runs in the top
// frame only, and nothing else in a subframe uses the handler.
const HideHandlerScript = `(function(){
if(window===window.top)return;
var hide=function(o,k){
if(!o||!(k in o))return;
try{delete o[k]}catch(e){}
if(k in o&&o[k]!==undefined)try{Object.defineProperty(o,k,{value:undefined,configurable:true})}catch(e){}
};
try{var w=window.webkit;if(w)hide(w.messageHandlers,"mygo");if(w&&w.messageHandlers&&w.messageHandlers.mygo)hide(window,"webkit")}catch(e){}
try{if(window.chrome)hide(window.chrome,"webview")}catch(e){}
})();`

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

// appendNumber appends f; false for a NaN or an infinity, which JSON has no
// text for and encoding/json refuses to write.
func appendNumber(b []byte, f float64) ([]byte, bool) {
	return jsonlite.AppendFloat(b, f)
}

// JSON returns the configuration as JSON, or nil when the limits in Fit cannot
// be written, as json.Marshal returned nothing for them.
func (c Config) JSON() []byte {
	b := append([]byte(nil), `{"platform":`...)
	b = jsonlite.AppendString(b, c.Platform)
	b = append(b, `,"windowId":`...)
	b = strconv.AppendInt(b, int64(c.WindowID), 10)
	b = append(b, `,"version":`...)
	b = jsonlite.AppendString(b, c.Version)
	b = append(b, `,"locale":`...)
	b = jsonlite.AppendString(b, c.Locale)
	b = append(b, `,"dir":`...)
	b = jsonlite.AppendString(b, c.Dir)
	if c.Fit != nil {
		b = append(b, `,"fit":`...)
		var ok bool
		if b, ok = c.Fit.AppendJSON(b); !ok {
			return nil
		}
	}
	return append(b, '}')
}

// AppendJSON appends the limits as JSON; false if one is a NaN or an infinity,
// which encoding/json does not write either.
func (f FitContent) AppendJSON(b []byte) ([]byte, bool) {
	var ok bool
	b = append(b, `{"minWidth":`...)
	if b, ok = appendNumber(b, f.MinWidth); !ok {
		return nil, false
	}
	b = append(b, `,"minHeight":`...)
	if b, ok = appendNumber(b, f.MinHeight); !ok {
		return nil, false
	}
	b = append(b, `,"maxWidth":`...)
	if b, ok = appendNumber(b, f.MaxWidth); !ok {
		return nil, false
	}
	b = append(b, `,"maxHeight":`...)
	if b, ok = appendNumber(b, f.MaxHeight); !ok {
		return nil, false
	}
	return append(b, '}'), true
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

// JSON returns the payload of TitleBarEvent as JSON; false if a number is a NaN
// or an infinity, which encoding/json does not write either.
func (t TitleBar) JSON() ([]byte, bool) {
	var ok bool
	b := append([]byte(nil), `{"height":`...)
	if b, ok = appendNumber(b, t.Height); !ok {
		return nil, false
	}
	b = append(b, `,"left":`...)
	if b, ok = appendNumber(b, t.Left); !ok {
		return nil, false
	}
	b = append(b, `,"right":`...)
	if b, ok = appendNumber(b, t.Right); !ok {
		return nil, false
	}
	return append(b, '}'), true
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
	p, _ := NewTitleBar(tb, zoom).JSON()
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

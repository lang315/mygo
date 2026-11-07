//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"

	"github.com/egoist/mygo/internal/platform"
)

// surfaceHost is what the window, the message loop and the caption bar call of
// the surface of a window of native UI (surface.go). They name this interface and not *surface:
// the linker keeps the methods of a type that is converted to an interface only
// where the conversion is, and that is in newSurface, which UseSurface alone
// reaches. An app of web views never calls UseSurface, and links none of the
// surface: the GPU renderers, the text shaping, UI Automation, the drop target.
type surfaceHost interface {
	showing()
	fit()
	fillUncovered()
	fillUnder()
	checkShown()
	message(hwnd uintptr, m uint32, wp, lp uintptr) (uintptr, bool)
	pointer(kind platform.SurfaceEventKind, lp uintptr, extra int)
	send(platform.SurfaceEvent) bool
	menuKey() bool
	liftKeys()
}

// makeSurfaceFunc makes the surface of a window. It is nil until UseSurface.
var makeSurfaceFunc func(w *window) *surface

// UseSurface lets windows have a surface, which native UI needs (package ui calls
// it when it is imported). Without it a window made with WindowOptions.Surface
// is an error.
func UseSurface() { makeSurfaceFunc = newSurface }

var errNoSurface = errors.New("mygo: Content needs package ui")

// makeSurface gives the window its surface, or says why it cannot.
func (w *window) makeSurface() error {
	if makeSurfaceFunc == nil {
		return errNoSurface
	}
	if makeSurfaceFunc(w) == nil {
		return errors.New("mygo: cannot create the window's surface")
	}
	return nil
}

// surf returns the window's surface, nil without one.
func (w *window) surf() *surface {
	s, _ := w.sf.(*surface)
	return s
}

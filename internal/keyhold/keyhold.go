// Package keyhold holds the key events typed after a menu bar's key, until
// the frame that answers it: Alt alone (ui.KeyMenuBar) or F10 take the
// keyboard to a menu bar drawn with ui, which needs a frame built, and on
// Windows a frame comes only once the queue holds no input, after the keys
// typed quickly behind them. It also keeps F10 held from going in and out
// of the bar's keyboard mode. widget.MenuBar enables it; apps without one
// do not link it, as ui reaches it only through New.
package keyhold

import (
	"sync/atomic"

	"github.com/egoist/mygo/internal/platform"
)

// Holder holds the key events of one window.
type Holder interface {
	// Hold takes ev before the view does, and reports whether it took it:
	// a key, modifiers, text or command event while holding, and a repeat
	// of F10, which would take the bar out of its mode again. The changes
	// of the modifiers wait with the keys, so that a modifier let go after
	// a held key comes back after it, not held for good. Alt alone (KeyMenuBar) or F10
	// pressed, which the view gets, hold the events after them until the
	// frame they ask for, taken or not. A window going inactive or
	// out of sight (SurfaceBlur, SurfaceShown) drops the events held: the
	// mode they were typed into is over, as Windows' own menus cancel on
	// deactivation, and the state they would replay into is another.
	Hold(ev platform.SurfaceEvent) bool
	// Release ends the hold, once a frame is built, and sends the events
	// held in order: one that starts a hold again has those after it held.
	Release()
}

// maker makes a window's Holder once Enable stored it: Enable may run on
// the event goroutine (a handler opening a form with a MenuBar) while ui
// reads it on the main thread.
var maker atomic.Pointer[func(send func(platform.SurfaceEvent) bool) Holder]

// Enable makes ui hold the keys typed after a menu bar's key. It is
// idempotent and safe from any goroutine.
func Enable() {
	f := func(send func(platform.SurfaceEvent) bool) Holder { return &holder{send: send} }
	maker.Store(&f)
}

// Disable undoes Enable, for tests: windows that made a Holder keep it.
func Disable() { maker.Store(nil) }

// Enabled reports whether Enable ran.
func Enabled() bool { return maker.Load() != nil }

// New makes the Holder of a window whose events send handles, nil until
// Enable, so that ui holds nothing.
func New(send func(platform.SurfaceEvent) bool) Holder {
	if f := maker.Load(); f != nil {
		return (*f)(send)
	}
	return nil
}

type holder struct {
	send    func(platform.SurfaceEvent) bool
	holding bool
	held    []platform.SurfaceEvent
}

// menuKey reports whether ev is a menu bar's key pressed: Alt alone, or F10.
func menuKey(ev platform.SurfaceEvent) bool {
	return ev.Kind == platform.KeyPressed && ev.Mods == 0 && (ev.Key == platform.KeyMenuBar || ev.Key == platform.KeyF10)
}

func (h *holder) Hold(ev platform.SurfaceEvent) bool {
	switch ev.Kind {
	case platform.KeyPressed, platform.KeyReleased, platform.ModifiersChanged, platform.TextInput, platform.TextComposition, platform.SurfaceCommand:
		if h.holding {
			h.held = append(h.held, ev)
			return true
		}
		if menuKey(ev) {
			h.holding = !ev.Repeat
			return ev.Repeat
		}
	case platform.SurfaceBlur, platform.SurfaceShown:
		h.holding, h.held = false, nil
	}
	return false
}

func (h *holder) Release() {
	held := h.held
	h.holding, h.held = false, nil
	for _, ev := range held {
		h.send(ev) // a menu bar's key among them holds those after it again
	}
}

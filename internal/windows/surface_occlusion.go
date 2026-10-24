//go:build windows && (amd64 || arm64)

package windows

import (
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

const (
	dwmwaCloak   = 13 // DWMWA_CLOAK: a window cloaks itself with it, as the tests do
	dwmwaCloaked = 14 // DWMWA_CLOAKED: why a window is cloaked, 0 if not
	timerCloak   = 3  // the surface's timer that asks whether the window is uncloaked
	cloakPollMS  = 250
)

var procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")

// Occluded reports whether nothing of the window shows: it is minimized,
// hidden, or cloaked (on another virtual desktop, or a UWP app's suspended
// window). Windows does not tell when other windows cover this one, and
// DWM keeps composing it all the same. SurfaceShown follows once it shows:
// a restore or a show is a message to the window, and a cloaked window is
// asked about every cloakPollMS, as Windows sends no message when it is
// uncloaked.
func (s *surface) Occluded() bool {
	out, cloaked := s.out()
	if out {
		s.wasOut = true
		if cloaked {
			procSetTimer.Call(s.hwnd, timerCloak, cloakPollMS, 0)
		}
	}
	return out
}

// out reports whether nothing of the window shows, and whether it is
// cloaked.
func (s *surface) out() (out, cloaked bool) {
	if iconic, _, _ := procIsIconic.Call(s.w.hwnd); iconic != 0 {
		return true, false
	}
	if visible, _, _ := procIsWindowVisible.Call(s.w.hwnd); visible == 0 {
		return true, false
	}
	if has(procDwmGetWindowAttribute) {
		var c uint32
		if hr, _, _ := procDwmGetWindowAttribute.Call(s.w.hwnd, dwmwaCloaked, uintptr(unsafe.Pointer(&c)), 4); hr == 0 && c != 0 {
			return true, true
		}
	}
	return false, false
}

// checkShown tells the content that some of the window shows again, if
// Occluded last said none did.
func (s *surface) checkShown() {
	if !s.wasOut {
		return
	}
	if out, cloaked := s.out(); !out {
		s.wasOut = false
		procKillTimer.Call(s.hwnd, timerCloak)
		s.send(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	} else if !cloaked {
		procKillTimer.Call(s.hwnd, timerCloak)
	}
}

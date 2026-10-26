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

// Occluded reports whether nothing of the window shows: it is minimized,
// hidden, or cloaked (on another virtual desktop, or a UWP app's suspended
// window). Windows does not tell when other windows cover this one, and
// DWM keeps composing it all the same. SurfaceShown follows once it shows:
// a restore is a WM_SIZE, and a hidden or cloaked window is
// asked about every cloakPollMS, as Windows sends no message when it is
// uncloaked.
func (s *surface) Occluded() bool {
	out, poll := s.out()
	if out {
		s.wasOut = true
		if poll {
			procSetTimer.Call(s.hwnd, timerCloak, cloakPollMS, 0)
		}
	}
	return out
}

// out reports whether nothing of the window shows, and whether it is to be
// asked again (poll): a hidden or a cloaked window is, a minimized one
// restores with a WM_SIZE.
func (s *surface) out() (out, poll bool) {
	if iconic, _, _ := procIsIconic.Call(s.w.hwnd); iconic != 0 {
		return true, false
	}
	if visible, _, _ := procIsWindowVisible.Call(s.w.hwnd); visible == 0 {
		return true, true
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
	if out, poll := s.out(); !out {
		s.wasOut = false
		procKillTimer.Call(s.hwnd, timerCloak)
		s.send(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	} else if !poll {
		procKillTimer.Call(s.hwnd, timerCloak)
	}
}

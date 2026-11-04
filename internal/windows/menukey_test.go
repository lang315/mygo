//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// keyContent is native UI that takes Alt alone when it has a menu bar of its
// own, and notes the events it gets.
type keyContent struct {
	platform.WindowHandler
	bar bool
	got []platform.SurfaceEvent
}

func (k *keyContent) SurfaceEvent(ev platform.SurfaceEvent) bool {
	k.got = append(k.got, ev)
	return k.bar && ev.Kind == platform.KeyPressed && ev.Key == platform.KeyMenuBar
}

// Alt alone (SC_KEYMENU without a letter, after a key down that was not
// F10) goes to the content as KeyMenuBar, never as a key a handler would
// take; F10 the content saw going down, and its answer then stands.
func TestSurfaceMenuKey(t *testing.T) {
	for _, c := range []struct {
		name      string
		bar       bool    // the content draws a menu bar
		lastDown  uintptr // the last key down
		taken     bool    // the content took it
		want      bool
		wantSends int
	}{
		{"Alt alone, a menu bar", true, vkMenu, false, true, 1},
		{"Alt alone, no menu bar", false, vkMenu, false, false, 1},
		{"F10 taken going down", true, vkF10, true, true, 0},
		{"F10 not taken going down", true, vkF10, false, false, 0},
		{"Alt alone after a key taken before", false, vkMenu, true, false, 1},
	} {
		content := &keyContent{bar: c.bar}
		s := &surface{w: &window{h: content}, lastDown: c.lastDown, keyTaken: c.taken}
		if got := s.menuKey(); got != c.want {
			t.Errorf("%s: taken %v, want %v", c.name, got, c.want)
		}
		if len(content.got) != c.wantSends {
			t.Errorf("%s: the content got %d events, want %d", c.name, len(content.got), c.wantSends)
		}
		for _, ev := range content.got {
			if ev.Kind != platform.KeyPressed || ev.Key != platform.KeyMenuBar || ev.Mods != 0 {
				t.Errorf("%s: the content got %+v, want KeyMenuBar", c.name, ev)
			}
		}
	}
}

// The window answers SC_KEYMENU without a letter itself, so that
// DefWindowProc starts no window menu loop, only when it has no menu bar of
// Win32's and its content took the key.
func TestWindowSysCommandKeyMenu(t *testing.T) {
	for _, c := range []struct {
		name string
		bar  bool
		want bool
	}{
		{"a menu bar in the content", true, true},
		{"none", false, false},
	} {
		content := &keyContent{bar: c.bar}
		w := &window{b: &Backend{}, h: content}
		w.sf = &surface{w: w, lastDown: vkMenu}
		if _, ok := w.message(wmSysCommand, scKeyMenu, 0); ok != c.want {
			t.Errorf("%s: answered %v, want %v", c.name, ok, c.want)
		}
	}
	// Alt and a letter, and Alt and Space, are not this key.
	content := &keyContent{bar: true}
	w := &window{b: &Backend{}, h: content}
	w.sf = &surface{w: w, lastDown: vkMenu}
	for _, lp := range []uintptr{'f', ' '} {
		if _, ok := w.message(wmSysCommand, scKeyMenu, lp); ok {
			t.Errorf("SC_KEYMENU with %q was answered", rune(lp))
		}
	}
	if len(content.got) != 0 {
		t.Errorf("the content got %v", content.got)
	}
}

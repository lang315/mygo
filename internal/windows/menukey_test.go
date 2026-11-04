//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// keyContent is native UI that takes F10 when it has a menu bar of its own,
// and notes the events it gets.
type keyContent struct {
	platform.WindowHandler
	bar bool
	got []platform.SurfaceEvent
}

func (k *keyContent) SurfaceEvent(ev platform.SurfaceEvent) bool {
	k.got = append(k.got, ev)
	return k.bar && ev.Kind == platform.KeyPressed && ev.Key == platform.KeyF10 && ev.Mods == 0
}

// Alt alone or F10 (SC_KEYMENU without a letter) in a window without a menu
// bar of the system's go to the content as F10; the window menu's loop
// starts only when the content does not take them.
func TestSurfaceMenuKeyOffersAltAloneAndF10ToTheContent(t *testing.T) {
	for _, c := range []struct {
		name      string
		bar       bool // the content draws a menu bar
		f10Taken  bool // the key down was F10, which the content took
		want      bool
		wantSends int
	}{
		{"Alt alone, a menu bar", true, false, true, 1},
		{"Alt alone, no menu bar", false, false, false, 1},
		{"F10 taken as it went down", true, true, true, 0},
		{"F10 not taken, no menu bar", false, false, false, 1},
	} {
		content := &keyContent{bar: c.bar}
		s := &surface{w: &window{h: content}, keyTaken: c.f10Taken}
		if got := s.menuKey(); got != c.want {
			t.Errorf("%s: taken %v, want %v", c.name, got, c.want)
		}
		if len(content.got) != c.wantSends {
			t.Errorf("%s: the content got %d events, want %d", c.name, len(content.got), c.wantSends)
		}
		for _, ev := range content.got {
			if ev.Kind != platform.KeyPressed || ev.Key != platform.KeyF10 || ev.Mods != 0 {
				t.Errorf("%s: the content got %+v, want F10", c.name, ev)
			}
		}
	}
}

package keyhold

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// While a menu bar's key holds the events after it, a lone modifier's raw key
// waits in its place among them: on Windows it comes with ModifiersChanged,
// and one that jumped ahead of the keys held would reach the view first.
func TestHoldKeepsRawKeysInOrder(t *testing.T) {
	var sent []platform.SurfaceEventKind
	h := &holder{send: func(ev platform.SurfaceEvent) bool {
		sent = append(sent, ev.Kind)
		return true
	}}
	if h.Hold(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyMenuBar}) {
		t.Fatal("the menu bar's key was taken: the view gets it")
	}
	order := []platform.SurfaceEventKind{
		platform.KeyPressed, platform.ModifiersChanged, platform.RawKeyPressed, platform.RawKeyReleased, platform.KeyReleased,
	}
	for _, k := range order {
		if !h.Hold(platform.SurfaceEvent{Kind: k, Key: platform.KeyA}) {
			t.Errorf("event %v was not held while the hold lasts", k)
		}
	}
	h.Release()
	if !slices.Equal(sent, order) {
		t.Errorf("released %v, want %v", sent, order)
	}
}

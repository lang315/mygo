//go:build !js

package mygo

import (
	"slices"
	"testing"
)

// A window that was closed or destroyed is no longer one of Windows(), which
// the quit sequence goes through asking each to close.
func TestClosedWindowsLeaveTheList(t *testing.T) {
	for name, end := range map[string]func(*Window){
		"Close":   func(w *Window) { w.Close() },
		"Destroy": func(w *Window) { w.Destroy() },
	} {
		t.Run(name, func(t *testing.T) {
			w, _ := testWindow(t, WindowOptions{})
			if !slices.Contains(Windows(), w) {
				t.Fatal("a new window is not in Windows()")
			}
			end(w)
			if slices.Contains(Windows(), w) {
				t.Errorf("a window ended by %s is still in Windows()", name)
			}
		})
	}
}

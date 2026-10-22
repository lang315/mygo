//go:build windows

package e2e

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestAppFocusKeepsFront brings the app forward with the window in front
// staying there, as macOS does, rather than another of the app's windows.
func TestAppFocusKeepsFront(t *testing.T) {
	var ws []*mygo.Window
	for i := range 3 {
		ws = append(ws, newWindow(t, mygo.WindowOptions{
			Title: fmt.Sprint("Window ", i), Width: 300, Height: 200, X: 100 + 40*i, Y: 100 + 40*i,
			Content: ui.View(func(c *ui.Context) { ui.Text(c, "window") }),
		}))
	}
	for _, w := range ws {
		w.Focus()
		// Not eventually: its message has no word on why the window is not in front.
		if !wait.Poll(w.IsFocused) {
			t.Fatalf("%s focused never happened (waited %v); %s", w.Title(), wait.Guard, foregroundDiag(w))
		}
		mygo.App.Focus()
		for _, other := range ws {
			if other != w && other.IsFocused() {
				t.Errorf("with %s in front, App.Focus brought %s forward; %s", w.Title(), other.Title(), foregroundState(other))
			}
		}
		if !w.IsFocused() {
			t.Errorf("App.Focus took the focus from %s; %s", w.Title(), foregroundState(w))
		}
	}
}

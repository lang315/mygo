//go:build !mygo_noinspector && !js

package e2e

import (
	"bytes"
	"image/color"
	"image/png"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestContentWindowInspector opens the inspector of native UI with the
// Toggle Developer Tools menu item, beside the content, and closes it.
func TestContentWindowInspector(t *testing.T) {
	var frames atomic.Int32
	var width atomic.Value
	width.Store(float32(0))
	view := func(c *ui.Context) {
		frames.Add(1)
		w, _ := c.Size()
		width.Store(w)
		ui.Box(c).Fill().Background(ui.RGB(30, 144, 255))
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Inspector", Width: 600, Height: 400, Content: ui.View(view),
		Page: mygo.PageOptions{DevTools: mygo.DevToolsEnabled}})
	// A frame with a width: the content's full width is what the inspector
	// takes from, and a 0 read before the first build stored one would
	// leave nothing to take from.
	eventually(t, "a frame", func() bool { return width.Load().(float32) > 0 })
	full := width.Load().(float32)
	// The item's Click runs right after its role (menuItemClicked), on the
	// main thread, before the frame the toggle asked for: it counts the
	// toggles the menu delivered, and the frames built until then.
	var toggles, built atomic.Int32
	menu := mygo.NewMenu([]*mygo.MenuItem{{Label: "Test", Submenu: []*mygo.MenuItem{{Role: mygo.RoleToggleDevTools,
		Click: func(*mygo.MenuItem, *mygo.Window) {
			built.Store(frames.Load())
			toggles.Add(1)
		}}}}})
	mygo.App.SetMenu(menu)
	defer mygo.App.SetMenu(nil)
	w.Focus()
	time.Sleep(100 * time.Millisecond)
	if err := activateMenu(w, "Test", "Toggle Developer Tools"); err != nil {
		t.Skip("menu automation not available on this platform: ", err)
	}
	// If it never comes: no toggle says the menu did not reach the window;
	// a toggle and no frame since, that its frame was never built; frames
	// at the full width, that the inspector did not open, or closed again.
	if !wait.Poll(func() bool {
		if w := width.Load().(float32); w > 0 && w < full {
			return true
		}
		keepFront()
		return false
	}) {
		t.Fatalf("the content beside the inspector never happened (waited %v); the content was %v wide before, %v now; the menu delivered %d toggles; %d frames built before the toggle, %d now; %s%s",
			wait.Guard, full, width.Load(), toggles.Load(), built.Load(), frames.Load(), foregroundState(w), inputDiag())
	}
	beside := width.Load().(float32)
	data, err := w.CapturePage()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	s := deviceScale(w)
	b := img.Bounds()
	at := func(x, y float64) color.RGBA {
		return color.RGBAModel.Convert(img.At(b.Min.X+int(x*s), b.Min.Y+int(y*s))).(color.RGBA)
	}
	if c := at(float64(beside)-20, 200); c.B < 200 || c.R > 60 {
		t.Errorf("the content's background is %v", c)
	}
	// The panel's background, in the theme's colors, not the content's.
	if c := at(float64(beside)+100, 200); c.A < 255 || c.B >= 200 && c.R <= 60 {
		t.Errorf("the inspector's background is %v", c)
	}
	if err := activateMenu(w, "Test", "Toggle Developer Tools"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the content filling the window again", func() bool { return width.Load().(float32) == full })
}

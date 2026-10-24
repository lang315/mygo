//go:build windows

package e2e

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// TestWindowOutOfSightDrawsNoFrames checks that a window out of sight asks for
// no frames of what moves, and draws again once it shows
// (platform.OccludableSurface). Windows does not paint a minimized or a
// hidden window by itself; a cloaked one (another virtual desktop) it does,
// so that case is the one the surface must answer for.
func TestWindowOutOfSightDrawsNoFrames(t *testing.T) {
	for _, c := range []struct {
		name      string
		out, back func(w *mygo.Window)
	}{
		{"minimized", func(w *mygo.Window) { w.Minimize() }, func(w *mygo.Window) { w.Restore() }},
		{"hidden", func(w *mygo.Window) { w.Hide() }, func(w *mygo.Window) { w.Show() }},
		{"cloaked",
			func(w *mygo.Window) {
				var err error
				mygo.RunOnMain(func() { err = win.TestCloak(w.NativeHandle(), true) })
				if err != nil {
					t.Fatal(err)
				}
			},
			func(w *mygo.Window) {
				mygo.RunOnMain(func() { win.TestCloak(w.NativeHandle(), false) })
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var frames atomic.Int32
			w := newWindow(t, mygo.WindowOptions{Title: "Out of sight " + c.name, Width: 300, Height: 200, Content: ui.View(func(c *ui.Context) {
				c.AnimationFrame()
				ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) { frames.Add(1) })
			})})
			eventually(t, "frames while shown", func() bool { return frames.Load() >= 10 })
			c.out(w)
			// Frames already asked for end; a negative can only be checked
			// by waiting, as the chart's idle test does.
			time.Sleep(500 * time.Millisecond)
			before := frames.Load()
			time.Sleep(time.Second)
			n := frames.Load() - before
			t.Logf("%s: %d frames in a second out of sight", c.name, n)
			if n > 1 {
				t.Fatalf("a window out of sight (%s) drew %d frames in a second", c.name, n)
			}
			c.back(w)
			eventually(t, "frames after the window shows again", func() bool { return frames.Load() >= before+10 })
		})
	}
}

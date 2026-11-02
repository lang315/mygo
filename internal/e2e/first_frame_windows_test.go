//go:build windows && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// cloaked reports whether DWM keeps w out of sight, and whether its first
// frame reached its surface.
func cloaked(w *mygo.Window) (cloaked, drawn bool) {
	mygo.RunOnMain(func() { cloaked, drawn = win.TestCloaked(w.NativeHandle()) })
	return
}

// TestContentWindowCloakedUntilFirstFrame checks that a window of native
// UI stays out of sight, though shown, until its first frame shows in it:
// it showed white until then, as the GPU renderer started (#905).
func TestContentWindowCloakedUntilFirstFrame(t *testing.T) {
	var w atomic.Pointer[mygo.Window]
	// atBuild is how the first build found the window: 1 cloaked, 2 not,
	// 3 not found. The build may run while NewWindow creates the window,
	// before w is set (at 200%, say), so it looks for windows not drawn yet.
	var atBuild atomic.Int32
	var ready atomic.Bool
	mygo.RunOnMain(func() {
		nw := newWindow(t, mygo.WindowOptions{Width: 320, Height: 240, Content: ui.View(func(c *ui.Context) {
			c.Root().Background(ui.RGB(30, 30, 30))
			if atBuild.Load() != 0 {
				return
			}
			switch n, cloaked := win.TestUndrawnCloaked(); {
			case n == 0:
				atBuild.Store(3)
			case cloaked:
				atBuild.Store(1)
			default:
				atBuild.Store(2)
			}
		})})
		w.Store(nw)
		nw.OnReadyToShow(func() { ready.Store(true) })
	})
	eventually(t, "ready to show", ready.Load)
	switch atBuild.Load() {
	case 2:
		t.Fatal("shown before its first frame: not cloaked as it drew it")
	case 3:
		t.Fatal("no window of native UI waiting for its first frame as it built")
	}
	eventually(t, "the window uncloaked after its first frame", func() bool {
		c, drawn := cloaked(w.Load())
		return !c && drawn
	})
	if !w.Load().IsVisible() {
		t.Error("not visible")
	}
}

// firstFrameRevealed reports whether w, whose first frame was drawn, is
// out of DWM's cloak: shown, it showed its frame.
func firstFrameRevealed(w *mygo.Window) bool {
	c, drawn := cloaked(w)
	return !c && drawn
}

// uncloaks counts the windows that came out of the cloak by their frame and by
// the timer that shows one no frame showed in.
func uncloaks() (byFrame, byTimer int) {
	mygo.RunOnMain(func() { byFrame, byTimer = win.TestUncloaks() })
	return
}

// TestWindowsUncloakWithTheirFrame checks that a window of native UI and its
// modal child come out of the cloak by the frame that showed in them, not by
// the timer that rescues a window no frame shows in, which would have them
// appear a second late.
func TestWindowsUncloakWithTheirFrame(t *testing.T) {
	frames0, timers0 := uncloaks()
	parent := newWindow(t, mygo.WindowOptions{Title: "Uncloak parent", Width: 320, Height: 240,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "parent") })})
	child := newWindow(t, mygo.WindowOptions{Title: "Uncloak child", Width: 200, Height: 120, Parent: parent, Modal: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "child") })})
	eventually(t, "the parent uncloaked", func() bool { return firstFrameRevealed(parent) })
	eventually(t, "the modal child uncloaked", func() bool { return firstFrameRevealed(child) })
	frames, timers := uncloaks()
	if timers != timers0 {
		t.Errorf("%d windows came out of the cloak by the timer, not by a frame", timers-timers0)
	}
	if frames-frames0 < 2 {
		t.Errorf("%d windows came out of the cloak by a frame, want the parent and its modal child", frames-frames0)
	}
}

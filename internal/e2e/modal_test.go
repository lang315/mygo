//go:build !js

package e2e

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestModalChildBlocksItsParent settles V1 of ui/widget's spec: a modal
// child window blocks its parent's input while it is open, and the parent
// takes input again once it closes.
func TestModalChildBlocksItsParent(t *testing.T) {
	var clicks, frames atomic.Int32
	parent := newWindow(t, mygo.WindowOptions{Title: "Modal parent", Width: 400, Height: 300,
		Content: ui.View(func(c *ui.Context) {
			frames.Add(1)
			ui.Column(c).Fill().OnClick(func() { clicks.Add(1) })
		})})
	eventually(t, "the parent's first frame", func() bool { return frames.Load() > 0 })
	// Low on the left, inside the content (Height counts the title bar):
	// a macOS sheet hangs from the top, and a modal window centers over
	// the parent.
	const x, y = 60, 230
	if !appClick(parent, x, y) {
		t.Skip("no system clicks on this platform")
	}
	eventually(t, "a click on the parent", func() bool { return clicks.Load() == 1 })

	var closed atomic.Bool
	child := newWindow(t, mygo.WindowOptions{Title: "Modal child", Width: 200, Height: 100,
		Parent: parent, Modal: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "modal") })})
	child.OnClosed(func() { closed.Store(true) })
	time.Sleep(time.Second) // the child shows; a sheet slides down
	appClick(parent, x, y)
	time.Sleep(500 * time.Millisecond) // a click let through has counted by now
	if n := clicks.Load(); n != 1 {
		t.Fatalf("the parent took %d clicks while its modal child was open", n-1)
	}

	child.Close()
	eventually(t, "the child to close", closed.Load)
	appClick(parent, x, y)
	eventually(t, "a click on the parent after the child closed", func() bool { return clicks.Load() == 2 })
}

// TestModalChildAndOtherWindows settles V7 of ui/widget's spec, the
// platform behaviors it quoted from general knowledge: GTK3 modality
// blocks every window of the app, while a macOS sheet and a Windows modal
// window block their parent only; and a second sheet on the same parent
// waits on macOS for the first to close.
func TestModalChildAndOtherWindows(t *testing.T) {
	var parentClicks, otherClicks, frames atomic.Int32
	view := func(clicks *atomic.Int32) *ui.Content {
		return ui.View(func(c *ui.Context) {
			frames.Add(1)
			ui.Column(c).Fill().OnClick(func() { clicks.Add(1) })
		})
	}
	// Side by side, inside Xvfb's 640x480 screen.
	parent := newWindow(t, mygo.WindowOptions{Title: "Modal parent", X: 10, Y: 40, Width: 300, Height: 260, Content: view(&parentClicks)})
	other := newWindow(t, mygo.WindowOptions{Title: "Other window", X: 330, Y: 40, Width: 300, Height: 260, Content: view(&otherClicks)})
	eventually(t, "both windows' first frames", func() bool { return frames.Load() >= 2 })
	const x, y = 60, 200
	if !appClick(other, x, y) {
		t.Skip("no system clicks on this platform")
	}
	eventually(t, "a click on the other window", func() bool { return otherClicks.Load() == 1 })

	modal := func(title string) (w *mygo.Window, drawn, closed *atomic.Bool) {
		drawn, closed = new(atomic.Bool), new(atomic.Bool)
		w = newWindow(t, mygo.WindowOptions{Title: title, Width: 200, Height: 100, Parent: parent, Modal: true,
			Content: ui.View(func(c *ui.Context) { drawn.Store(true); ui.Text(c, title) })})
		w.OnClosed(func() { closed.Store(true) })
		return w, drawn, closed
	}
	first, _, firstClosed := modal("First modal")
	time.Sleep(time.Second) // a sheet slides down
	appClick(other, x, y)
	blocked := !happens(500*time.Millisecond, func() bool { return otherClicks.Load() == 2 })
	t.Logf("%s: the other window blocked while a modal child is open: %v", runtime.GOOS, blocked)
	if want := runtime.GOOS == "linux"; blocked != want {
		t.Errorf("other window blocked %v, want %v on %s", blocked, want, runtime.GOOS)
	}

	second, secondDrawn, secondClosed := modal("Second modal")
	queued := !happens(time.Second, second.IsVisible)
	t.Logf("%s: a second modal child of the same parent waits for the first: %v", runtime.GOOS, queued)
	if want := runtime.GOOS == "darwin"; queued != want {
		t.Errorf("second modal child queued %v, want %v on %s", queued, want, runtime.GOOS)
	}
	if !queued {
		// Closing the first before the second drew broke the input
		// method of a later test's window under Xvfb.
		eventually(t, "the second child's first frame", secondDrawn.Load)
	}
	first.Close()
	eventually(t, "the first child to close", firstClosed.Load)
	eventually(t, "the second child to show", second.IsVisible)
	second.Close()
	eventually(t, "the second child to close", secondClosed.Load)
}

// TestNestedModalBlocksTheOuterChild settles the nested case of
// ui/widget's dialogs: a modal child of a modal child blocks the outer
// child, which takes input again once the inner one closes.
func TestNestedModalBlocksTheOuterChild(t *testing.T) {
	var clicks, frames atomic.Int32
	parent := newWindow(t, mygo.WindowOptions{Title: "Nested parent", X: 10, Y: 40, Width: 600, Height: 420,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "parent") })})
	outer := newWindow(t, mygo.WindowOptions{Title: "Outer child", Width: 400, Height: 300, Parent: parent, Modal: true,
		Content: ui.View(func(c *ui.Context) {
			frames.Add(1)
			ui.Column(c).Fill().OnClick(func() { clicks.Add(1) })
		})})
	eventually(t, "the outer child's first frame", func() bool { return frames.Load() > 0 })
	time.Sleep(time.Second) // a sheet slides down
	// Low on the left, away from the small inner child, which centers or
	// hangs from the top.
	const x, y = 30, 260
	if !appClick(outer, x, y) {
		t.Skip("no system clicks on this platform")
	}
	eventually(t, "a click on the outer child", func() bool { return clicks.Load() == 1 })

	var closed atomic.Bool
	inner := newWindow(t, mygo.WindowOptions{Title: "Inner child", Width: 160, Height: 80, Parent: outer, Modal: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "inner") })})
	inner.OnClosed(func() { closed.Store(true) })
	time.Sleep(time.Second)
	appClick(outer, x, y)
	time.Sleep(500 * time.Millisecond)
	if n := clicks.Load(); n != 1 {
		t.Fatalf("the outer child took %d clicks while its own modal child was open", n-1)
	}
	inner.Close()
	eventually(t, "the inner child to close", closed.Load)
	appClick(outer, x, y)
	eventually(t, "a click on the outer child after the inner closed", func() bool { return clicks.Load() == 2 })
}

// happens reports whether cond holds within d, as soon as it does.
func happens(d time.Duration, cond func() bool) bool {
	for deadline := time.Now().Add(d); !cond(); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			return false
		}
	}
	return true
}

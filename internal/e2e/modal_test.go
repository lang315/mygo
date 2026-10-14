//go:build !js

package e2e

import (
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

//go:build windows

package e2e

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Where the front goes as a modal child closes, beyond
// TestParentStaysBlockedWhileAModalChildIsOpen: Windows gives it to the next
// window down, which can be another app's.

// TestClosingAModalChildFrontsTheOneUnderIt: of three modal children of one
// parent, closing the one in front gives the front to the one under it.
func TestClosingAModalChildFrontsTheOneUnderIt(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Title: "Modal parent", Width: 400, Height: 300,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "parent") })})
	var children [3]*mygo.Window
	for i := range children {
		title := fmt.Sprintf("Modal %d", i+1)
		children[i] = newWindow(t, mygo.WindowOptions{Title: title, Width: 200, Height: 100, Parent: parent, Modal: true,
			Content: ui.View(func(c *ui.Context) { ui.Text(c, title) })})
		children[i].Focus()
		eventually(t, title+" in front", children[i].IsFocused)
	}
	var closed atomic.Bool
	children[2].OnClosed(func() { closed.Store(true) })
	children[2].Close()
	eventually(t, "the child in front to close", closed.Load)
	eventually(t, "the child under it to be in front", children[1].IsFocused)
}

// TestClosingAModalChildPassesOverAHiddenOne: with another modal child not
// shown yet, closing the one in front gives the front to the parent, which
// the hidden one keeps blocked, not to the hidden one.
func TestClosingAModalChildPassesOverAHiddenOne(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Title: "Modal parent", Width: 400, Height: 300,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "parent") })})
	newWindow(t, mygo.WindowOptions{Title: "Hidden modal", Width: 200, Height: 100, Parent: parent, Modal: true, Hidden: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "hidden") })})
	shown := newWindow(t, mygo.WindowOptions{Title: "Shown modal", Width: 200, Height: 100, Parent: parent, Modal: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "shown") })})
	shown.Focus()
	eventually(t, "the shown child in front", shown.IsFocused)
	var closed atomic.Bool
	shown.OnClosed(func() { closed.Store(true) })
	shown.Close()
	eventually(t, "the shown child to close", closed.Load)
	eventually(t, "the parent to be in front", parent.IsFocused)
}

// TestClosingAModalChildWithItsOwnFrontsTheParent: closing a modal child
// while its own modal child is in front, which goes with it, gives the
// front to the parent.
func TestClosingAModalChildWithItsOwnFrontsTheParent(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Title: "Nested parent", Width: 600, Height: 420,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "parent") })})
	outer := newWindow(t, mygo.WindowOptions{Title: "Outer child", Width: 400, Height: 300, Parent: parent, Modal: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "outer") })})
	inner := newWindow(t, mygo.WindowOptions{Title: "Inner child", Width: 160, Height: 80, Parent: outer, Modal: true,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "inner") })})
	inner.Focus()
	eventually(t, "the inner child in front", inner.IsFocused)
	var closed atomic.Bool
	outer.OnClosed(func() { closed.Store(true) })
	outer.Close()
	eventually(t, "the outer child to close", closed.Load)
	eventually(t, "the parent to be in front", parent.IsFocused)
}

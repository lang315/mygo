//go:build !js

package e2e

import (
	"testing"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui/widget"
)

// TestWidgetDialogEscape presses a real Escape on a ui/widget dialog: it
// closes as its close button closes it, and Run returns.
func TestWidgetDialogEscape(t *testing.T) {
	w := widget.NewForm("Escape parent", 400, 300).SetContent(widget.Label("parent"))
	w.Show()
	t.Cleanup(w.Close)
	eventually(t, "the parent window", func() bool { return w.Native() != nil })
	d := widget.NewDialog(w, "Escape me", "", 300, 200, widget.Label("press Escape"))
	done := make(chan struct{})
	go func() { d.Run(); close(done) }()
	eventually(t, "the dialog to show", func() bool { n := d.Native(); return n != nil && n.IsVisible() })
	// A sheet takes the keyboard as it shows: that is what Escape needs.
	eventually(t, "the dialog to take the keyboard", func() bool { return d.Native().IsFocused() })
	if !pressEscape(d.Native()) {
		t.Skip("no system keys on this platform")
	}
	if !wait.Poll(func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}) {
		state := foregroundState(d.Native())
		d.Close()
		t.Fatalf("Escape did not close the dialog; %s", state)
	}
}

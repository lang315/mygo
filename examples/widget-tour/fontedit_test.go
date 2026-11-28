package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Dialogs tab has a FontEdit after the font dialog's button, showing
// its family. It is not opened: that reads the system's families, which
// differ from one machine to the next.
func TestTourFontEdit(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Dialogs")).Click()
	button := tt.Find(widget.ByLabel("Font dialog")).Bounds()
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleComboBox)) {
		if n.Value() != "Menlo" {
			continue
		}
		// The next in the flow: after the button on its line, or on the next.
		if r := n.Bounds(); r.Y+r.H < button.Y || r.Y < button.Y+button.H && r.X < button.X+button.W {
			t.Fatalf("the FontEdit at %v does not follow the Font dialog button at %v", r, button)
		}
		return
	}
	t.Fatalf("no FontEdit showing Menlo on the Dialogs tab: %q", tt.Texts())
}

package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A text box on a tab page fills the page, alone or in a box or group box,
// as a text box does as the only child of a form: the page does not leave it
// its natural height. A table page is the control case.
func TestTabControlPageFillsATextBox(t *testing.T) {
	const w, h = 600, 400
	for name, page := range map[string]func() Control{
		"alone":          func() Control { return MultilineTextBox().SetPlaceholder("note") },
		"in a box":       func() Control { return VBox(4).Add(MultilineTextBox().SetPlaceholder("note")) },
		"in a group box": func() Control { return GroupBox("G").Add(MultilineTextBox().SetPlaceholder("note")) },
		"table":          func() Control { return testTable(accountRows(3)...) },
	} {
		t.Run(name, func(t *testing.T) {
			tt := NewTester(t, TabControl().AddPage("One", page()).Expand(), w, h)
			tt.Frame()
			var area ui.TestNode
			for _, n := range tt.Tester.Nodes() {
				if name == "table" && n.Role == ui.RoleTable || name != "table" && n.Role == ui.RoleTextField {
					area = n
				}
			}
			if bottom := area.Bounds.Y + area.Bounds.H; area.Bounds.H == 0 || bottom < h-24 {
				t.Fatalf("control %v ends at %v, short of the window's %d", area.Bounds, bottom, h)
			}
		})
	}
}

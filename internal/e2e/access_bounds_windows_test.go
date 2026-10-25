//go:build windows

package e2e

import (
	"testing"

	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui/widget"
)

// UI Automation gives a screen reader the place of each element: its bounding
// rectangle, on the screen. A client reads it through UI Automation's core
// (CUIAutomation), which gave 0,0,0,0 for every element of a widget window
// on Windows ARM64 (the callbacks of the provider changed the registers
// F8-F15 the core keeps its rectangles in: see internal/wincb).
func TestUIAClientGetsTheBoundsOfWidgets(t *testing.T) {
	f := widget.NewForm("Bounds", 400, 300)
	f.SetContent(widget.VBox(4).Add(
		widget.Label("Settings"),
		widget.TextBox().Focus(),
		widget.MaskedTextBox("(000) 000-0000"),
		widget.Button("Save"),
	))
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	hwnd := f.Native().NativeHandle()
	var rects [][4]int32
	eventually(t, "two Edit elements", func() bool {
		var ok bool
		rects, ok = win.TestUIAClientEdits(hwnd) // off the main thread, which answers the core
		return ok && len(rects) >= 2
	})
	for i, r := range rects {
		if r[2] <= 0 || r[3] <= 0 || r[0] <= 0 || r[1] <= 0 {
			t.Errorf("Edit %d: bounding rectangle %v (left, top, width, height), want its place on the screen", i, r)
		}
	}
}

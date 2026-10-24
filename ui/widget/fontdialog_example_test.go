package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func applyFont(ui.Font) {}

// The snippet of docs/ui/widget/fontdialog.md, which must compile.
func ExampleFontDialog() {
	var form *widget.Form
	fd := widget.FontDialog().SetTitle("Editor font").SetFont(ui.Font{Family: "Menlo", Size: 13})
	if fd.ShowDialog(form) {
		applyFont(fd.Font())
	}
}

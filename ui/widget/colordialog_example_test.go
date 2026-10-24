package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func apply(ui.Color) {}

// The snippet of docs/ui/widget/colordialog.md, which must compile.
func ExampleColorDialog() {
	var form *widget.Form
	cd := widget.ColorDialog().SetTitle("Tint").SetColor(ui.Hex("#2563eb"))
	if cd.ShowDialog(form) {
		apply(cd.Color())
	}
}

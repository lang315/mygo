package ui_test

import "github.com/egoist/mygo/ui"

// The snippet of docs/ui/number-input.md, which must compile.
func ExampleSpinBox() {
	var app struct{ copies float64 }
	_ = func(c *ui.Context) {
		ui.SpinBox(c, &app.copies, 1, 99, 1).Label("Copies")
	}
}

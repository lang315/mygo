package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func save() {}

// The snippet of docs/ui/widget/form-keys.md, which must compile.
func ExampleForm_Shortcut() {
	var form *widget.Form
	var body widget.Control
	ok, cancel := widget.Button("OK"), widget.Button("Cancel")
	dlg := widget.NewDialog(form, "Sign in", "", 360, 200, body)
	dlg.Shortcut(ui.Ctrl, ui.KeyS, func() { save() })
	dlg.SetAcceptButton(ok)
	dlg.SetCancelButton(cancel)
}

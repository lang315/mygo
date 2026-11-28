package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The SuperTip of docs/ui/widget/supertip.md: a paste button's rich
// tooltip, with an icon at the start of its body and a footer.
func ExampleSuperTip() {
	var icon *ui.Bitmap // the app's clipboard icon, decoded once
	widget.Button("Paste").SetSuperTip(widget.SuperTip{
		Title:  "Paste (Ctrl+V)",
		Body:   "Puts the text of the clipboard where the caret is.",
		Image:  icon,
		Footer: "Press F1 for more help.",
	})
}

package main

import "github.com/egoist/mygo/ui/widget"

// fontEditField is a font family field, beside the font dialog's button:
// show shows the family the user picks.
func fontEditField(show func(format string, a ...any)) widget.Control {
	var font *widget.FontEditWidget
	font = widget.FontEdit().SetFamily("Menlo").OnChange(func() { show("FontEdit: %q", font.Family()) })
	return font.SetSize(220, 0)
}

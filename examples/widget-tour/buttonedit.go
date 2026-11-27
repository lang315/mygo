package main

import "github.com/egoist/mygo/ui/widget"

// buttonEditRow is a file field with a clear button and a browse button.
func buttonEditRow() widget.Control {
	status := widget.Label("File: none").Muted()
	var file *widget.ButtonEditWidget
	show := func() {
		if s := file.Text(); s != "" {
			status.SetText("File: " + s)
		} else {
			status.SetText("File: none")
		}
	}
	file = widget.ButtonEdit(
		widget.EditButton{Kind: widget.EditClear},
		widget.EditButton{Kind: widget.EditEllipsis, Tooltip: "Browse", OnClick: func() {
			file.SetText("report.pdf") // a real app opens its file dialog here
			show()
		}},
	).SetPlaceholder("File (Ctrl+Enter browses)").OnChange(show)
	return widget.HBox(8).Add(file, status)
}

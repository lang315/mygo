package widget_test

import "github.com/egoist/mygo/ui/widget"

// The ButtonEdit of docs/ui/widget/buttonedit.md: a clear button and a
// browse button at the end of the field.
func ExampleButtonEdit() {
	var path *widget.ButtonEditWidget
	path = widget.ButtonEdit(
		widget.EditButton{Kind: widget.EditClear},
		widget.EditButton{Kind: widget.EditEllipsis, Tooltip: "Browse", OnClick: func() {
			path.SetText("/tmp/report.pdf")
		}},
	).SetPlaceholder("Choose a file").OnChange(func() { println(path.Text()) })
	path.SetButtonVisible(0, false)
}

package main

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui/widget"
)

// popupWait shows a PopupEdit, a ProgressPanel and the wait form of w (none
// when w is nil, as in tests of the tour alone).
func popupWait(w *widget.Form) widget.Control {
	cols := widget.CheckBoxGroup(false,
		widget.Option{Value: "Name", Text: "Name"},
		widget.Option{Value: "City", Text: "City"},
		widget.Option{Value: "Phone", Text: "Phone"},
	)
	var edit *widget.PopupEditWidget
	edit = widget.PopupEdit(widget.Label("Columns to show").Bold(), cols).SetText("No columns").OnClosed(func() {
		if v := cols.Values(); len(v) > 0 {
			edit.SetText(strings.Join(v, ", "))
		} else {
			edit.SetText("No columns")
		}
	})
	row := widget.FlowPanel(8).Add(widget.ProgressPanel("Loading").SetDescription("An inline ProgressPanel"))
	if w != nil {
		var done *widget.TimerWidget
		done = widget.Timer(3 * time.Second).OnTick(func() { done.Stop(); w.CloseWait() })
		row.Add(widget.Button("Show wait form").OnClick(func() {
			w.ShowWait("Working", "This closes in 3 seconds")
			done.Start()
		}))
	}
	return widget.VBox(8).Add(
		widget.Label("Popup and wait").TextSize(widget.XL).Bold(),
		widget.Fields().Row("Columns", edit),
		row,
	)
}

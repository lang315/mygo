package main

import (
	"time"

	"github.com/egoist/mygo/ui/widget"
)

// timeSpanRow is a timeout field, from a second to two hours, with the
// timeout it holds beside it.
func timeSpanRow() widget.Control {
	said := widget.Label("Timeout 1m30s").Muted()
	var timeout *widget.TimeSpanEditWidget
	timeout = widget.TimeSpanEdit().SetRange(time.Second, 2*time.Hour).SetValue(90 * time.Second).
		OnChange(func() { said.SetText("Timeout " + timeout.Value().String()) })
	return widget.HBox(8).Add(timeout, said)
}

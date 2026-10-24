package main

import "github.com/egoist/mygo/ui/widget"

// maskedRow is a phone number input with a mask.
func maskedRow() widget.Control {
	status := widget.Label("Phone: incomplete").Muted()
	var phone *widget.MaskedTextBoxWidget
	phone = widget.MaskedTextBox("(000) 000-0000").OnChange(func() {
		if phone.MaskCompleted() {
			status.SetText("Phone: " + phone.TextWithLiterals())
		} else {
			status.SetText("Phone: incomplete")
		}
	})
	return widget.HBox(8).Add(phone, status)
}

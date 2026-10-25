package main

import (
	"strings"

	"github.com/egoist/mygo/ui/widget"
)

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

// validationRow is an email input with an ErrorProvider and a button that
// asks the form to validate.
func validationRow(w *widget.Form) widget.Control {
	ep := widget.ErrorProvider()
	status := widget.Label("Not validated").Muted()
	email := widget.TextBox().SetPlaceholder("Email")
	email.OnValidating(func() bool {
		if strings.Contains(email.Text(), "@") {
			ep.SetError(email, "")
			return true
		}
		ep.SetError(email, "Enter an email address")
		return false
	})
	return widget.HBox(8).Add(email, widget.Button("Validate").OnClick(func() {
		if w.ValidateChildren() {
			status.SetText("Valid")
		} else {
			status.SetText("Not valid")
		}
	}), status)
}

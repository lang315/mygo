package main

import (
	"regexp"

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

// validationRow is an email input with rules on an ErrorProvider, checked
// when the focus leaves it, and a button that asks the form to validate.
func validationRow(w *widget.Form) widget.Control {
	ep := widget.ErrorProvider().SetValidateOnLeave(true)
	status := widget.Label("Not validated").Muted()
	email := widget.Rule(ep, widget.TextBox().SetPlaceholder("Email"), widget.Required("Enter an email address"))
	widget.Rule(ep, email, widget.MatchRegexp(regexp.MustCompile(`^\S+@\S+\.\S+$`), "Not an email address"))
	return widget.HBox(8).Add(email, widget.Button("Validate").OnClick(func() {
		if w.ValidateChildren() {
			status.SetText("Valid")
		} else {
			status.SetText("Not valid")
		}
	}), status)
}

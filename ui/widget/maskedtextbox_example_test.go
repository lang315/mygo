package widget_test

import "github.com/egoist/mygo/ui/widget"

func savePhone(string) {}
func beep()            {}
func call(string)      {}

// The snippet of docs/ui/widget/maskedtextbox.md, which must compile.
func ExampleMaskedTextBox() {
	var phone *widget.MaskedTextBoxWidget
	phone = widget.MaskedTextBox("(000) 000-0000").
		OnChange(func() { savePhone(phone.Text()) }).
		OnMaskInputRejected(func(r rune) { beep() })
	if phone.MaskCompleted() {
		call(phone.TextWithLiterals())
	}
}

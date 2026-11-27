package widget_test

import "github.com/egoist/mygo/ui/widget"

// The CalcEdit of docs/ui/widget/calcedit.md: an amount with a calculator
// in its drop-down, shown with no decimals.
func ExampleCalcEdit() {
	var amount *widget.CalcEditWidget
	amount = widget.CalcEdit().SetDecimals(0).SetValue(1200).
		OnChange(func() { println(amount.Value()) })
	amount.ShowPopup()
	amount.ClosePopup()
}

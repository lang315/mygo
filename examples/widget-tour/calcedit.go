package main

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// calcEditRow is an amount with a calculator in its drop-down, and a label
// saying the amount.
func calcEditRow() widget.Control {
	said := widget.Label("Total 120.00")
	var total *widget.CalcEditWidget
	total = widget.CalcEdit().SetValue(120).SetAccessibleName("Total").
		OnChange(func() { said.SetText(fmt.Sprintf("Total %.2f", total.Value())) })
	return widget.HBox(8).Add(total, said)
}

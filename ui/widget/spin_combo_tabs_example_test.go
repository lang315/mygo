package widget_test

import (
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/tw"
)

// The NumericUpDown of docs/ui/widget.md: a spin box whose box Apply
// styles, and one with the buttons beside it.
func ExampleNumericUpDownWidget_SideButtons() {
	widget.VBox(8).Add(
		widget.NumericUpDown(1, 99, 1).Apply(tw.S("bg-amber-50 border-amber-500 rounded-lg")),
		widget.NumericUpDown(1, 99, 1).SideButtons(),
	)
}

// The ComboBox of docs/ui/widget.md: a submenu, a title that can be
// chosen, and a second AddSection adding to the first.
func ExampleComboBoxWidget_AddSubmenu() {
	widget.ComboBox(widget.Option{Value: "any", Text: "Any"}).
		AddSubmenu("Fruits", nil, widget.Option{Value: "apple", Text: "Apple"}).
		AddSection("Vegetables", nil, widget.Option{Value: "kale", Text: "Kale"}).
		AddSection("Vegetables", nil, widget.Option{Value: "leek", Text: "Leek"}).
		SetSectionValue("Vegetables", "vegetables")
}

// The TabControl of docs/ui/widget.md, its strip styled.
func ExampleTabControlWidget_StripStyle() {
	widget.TabControl().
		AddPage("General", widget.Label("General")).
		StripStyle(tw.S("bg-slate-100 rounded-md"))
}

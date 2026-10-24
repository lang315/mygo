package widget_test

import "github.com/egoist/mygo/ui/widget"

func plan(string) {}

// The snippet of docs/ui/widget/domainupdown.md, which must compile.
func ExampleDomainUpDown() {
	var day *widget.DomainUpDownWidget
	day = widget.DomainUpDown("Mon", "Tue", "Wed", "Thu", "Fri").
		SetWrap(true).
		OnChange(func() { plan(day.SelectedItem()) })
}

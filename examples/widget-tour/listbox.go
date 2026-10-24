package main

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// listBoxes shows a ListBox of countries and a CheckedListBox of toppings,
// each reporting what the user did in a label below it.
func listBoxes() widget.Control {
	type item struct{ id, name string }
	key := func(i *item) string { return i.id }
	text := func(i *item) string { return i.name }

	chosen := widget.Label("Choose countries (Cmd-click or Shift-click for several)")
	countries := widget.ListBox(key, text).Multi().SetItems([]item{
		{"vn", "Vietnam"}, {"us", "United States"}, {"jp", "Japan"}, {"fr", "France"}, {"br", "Brazil"},
	})
	countries.OnSelect(func(ks []string) { chosen.SetText(fmt.Sprintf("%d chosen: %v", len(ks), ks)) })

	counted := widget.Label("Check some toppings")
	var toppings *widget.ListBoxWidget[item, string]
	toppings = widget.CheckedListBox(key, text).SetItems([]item{
		{"cheese", "Cheese"}, {"ham", "Ham"}, {"olives", "Olives"}, {"basil", "Basil"},
	})
	toppings.OnCheck(func(string, bool) {
		counted.SetText(fmt.Sprintf("%d checked: %v", len(toppings.Checked()), toppings.Checked()))
	})

	return widget.VBox(8).Add(
		widget.Label("ListBox").Bold(), countries, chosen,
		widget.Label("CheckedListBox").Bold(), toppings, counted,
	)
}

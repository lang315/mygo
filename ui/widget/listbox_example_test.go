package widget_test

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of docs/ui/widget/listbox.md, which must compile.
func ExampleCheckedListBox() {
	status := widget.Label("")
	basket := struct{ Add, Remove func(string) }{func(string) {}, func(string) {}}
	open := func(id string) {}

	type fruit struct{ id, name string }

	list := widget.CheckedListBox(
		func(f *fruit) string { return f.id },
		func(f *fruit) string { return f.name },
	).SetItems([]fruit{{"a", "Apple"}, {"b", "Banana"}, {"c", "Cherry"}}).
		Multi()

	list.OnSelect(func(selected []string) {
		status.SetText(fmt.Sprint(len(selected), " chosen"))
	}).OnCheck(func(id string, on bool) {
		if on {
			basket.Add(id)
		} else {
			basket.Remove(id)
		}
	}).OnActivate(func(id string) {
		open(id)
	})

	picked := list.Checked() // the keys of the items checked, in the order of the list
	_ = picked
}

package widget_test

import (
	"strings"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of docs/ui/widget/treeview.md, "Sorting, filtering and
// renaming", which must compile.
func ExampleTreeViewWidget_SetSort() {
	rename := func(path, name string) error { return nil }
	tree := widget.TreeView(
		func(e *entry) string { return e.path },
		func(e *entry) string { return e.name },
	)

	tree.SetSort(func(a, b *entry) int {
		return strings.Compare(a.name, b.name)
	}).SetLabelEdit(true).OnAfterLabelEdit(func(path, name string) bool {
		if err := rename(path, name); err != nil {
			return false // the old name shows again
		}
		tree.UpdateNode(path, func(e *entry) { e.name = name })
		return true
	})

	search := widget.SearchBox()
	search.OnChange(func() {
		q := strings.ToLower(search.Text())
		if q == "" {
			tree.SetFilter(nil)
			return
		}
		tree.SetFilter(func(e *entry) bool { return strings.Contains(strings.ToLower(e.name), q) })
	})
}

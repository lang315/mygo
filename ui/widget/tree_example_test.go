package widget_test

import "github.com/egoist/mygo/ui/widget"

// The snippets of docs/ui/widget/treeview.md, which must compile.
func ExampleTreeView() {
	status := widget.Label("")
	listDir := func(path string) []entry { return nil }

	tree := widget.TreeView(
		func(e *entry) string { return e.path },
		func(e *entry) string { return e.name },
	).CheckBoxes().
		SetRoots([]entry{{"/home", "Home", true}, {"/tmp", "Temp", true}}).
		SetHasChildren("/home", true).
		SetHasChildren("/tmp", true)

	tree.OnExpand(func(path string) {
		go func() {
			kids := listDir(path) // the entries of a directory, read from the disk
			tree.SetChildren(path, kids)
			for _, k := range kids {
				if k.dir {
					tree.SetHasChildren(k.path, true)
				}
			}
		}()
	}).OnSelect(func(path string, ok bool) {
		if ok {
			status.SetText(path)
		}
	})

	tree.OnCheck(func(k string, on bool) {
		tree.SetChecked(tree.Children(k), on)
	})
}

type entry struct {
	path, name string
	dir        bool
}

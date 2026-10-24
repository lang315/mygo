package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A list, a table or a tree packed to expand and fill takes the room its
// boxes leave, however the boxes nest, and builds only the rows in it.
func TestPackedVirtualizedWidgetsTakeTheRemainingHeightWhateverTheNesting(t *testing.T) {
	const rows = 300
	kinds := []struct {
		name string
		role ui.Role
		make func() Control
	}{
		{"ListBox", ui.RoleListItem, func() Control {
			items := make([]litem, rows)
			for i := range items {
				items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
			}
			return newListBox().SetItems(items)
		}},
		{"DataGridView", ui.RoleRow, func() Control {
			all := make([]acct, rows)
			for i := range all {
				all[i] = acct{i, fmt.Sprint("name ", i), i}
			}
			return testTable(all...)
		}},
		{"TreeView", ui.RoleTreeItem, func() Control {
			roots := make([]tnode, rows)
			for i := range roots {
				roots[i] = tnode{fmt.Sprint("n", i), fmt.Sprint("Node ", i)}
			}
			return newTree().SetRoots(roots)
		}},
	}
	nestings := []struct {
		name string
		wrap func(w Control) Control
	}{
		{"VBox", func(w Control) Control { return VBox(4).Pack(w, true, true) }},
		{"VBox>VBox", func(w Control) Control { return VBox(4).Pack(VBox(4).Pack(w, true, true), true, true) }},
		{"VBox>HBox", func(w Control) Control { return VBox(4).Pack(HBox(4).Pack(w, true, true), true, true) }},
		{"HBox>VBox", func(w Control) Control { return HBox(4).Pack(VBox(4).Pack(w, true, true), true, true) }},
		{"VBox>HBox>VBox with a label", func(w Control) Control {
			return VBox(4).Pack(HBox(4).Pack(VBox(4).Add(Label("title")).Pack(w, true, true), true, true), true, true)
		}},
	}
	for _, k := range kinds {
		for _, n := range nestings {
			f := NewForm("t", 400, 460)
			f.SetContent(n.wrap(k.make()))
			tt := TestForm(t, f)
			tt.Settle()
			got := len(tt.FindAll(ByRole(k.role)))
			if got < 10 || got > 40 {
				t.Errorf("%s in %s: %d rows built of %d, want the ~20 that fit", k.name, n.name, got, rows)
			}
		}
	}
}

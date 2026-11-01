package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// items are the nodes of the list's items and the nodes named like them.
func itemNodes(tt *ui.Tester, role ui.Role) (items []ui.TestNode, named map[string]int) {
	named = map[string]int{}
	for _, n := range tt.Nodes() {
		if n.Role == role {
			items = append(items, n)
		}
		if n.Label != "" {
			named[n.Label]++
		}
	}
	return
}

// An item of a checked list box is a list item and its check box, not also
// the text: a screen reader reads the item and the box's state, not the name a
// third time. (The platforms' states give a list item its choice only, so the
// check is the box's.)
func TestCheckedListBoxItemIsAnItemAndItsBox(t *testing.T) {
	lb := newChecked().SetItems(fruit)
	lb.SetChecked([]string{"b"}, true)
	tt := tester(t, lb)
	tt.Frame()
	items, _ := itemNodes(tt, ui.RoleListItem)
	if len(items) != len(fruit) {
		t.Fatalf("%d list items for %d fruit", len(items), len(fruit))
	}
	roles := map[string][]ui.Role{}
	var boxes []ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Label != "" {
			roles[n.Label] = append(roles[n.Label], n.Role)
		}
		if n.Role == ui.RoleCheckBox {
			boxes = append(boxes, n)
		}
	}
	for _, f := range fruit {
		if got := roles[f.name]; len(got) != 2 || got[0] != ui.RoleListItem || got[1] != ui.RoleCheckBox {
			t.Errorf("nodes named %q: roles %v, want the item and its box", f.name, got)
		}
	}
	for _, b := range boxes {
		if want := b.Label == "Banana"; b.Checked != want {
			t.Errorf("box %q checked %v, want %v", b.Label, b.Checked, want)
		}
	}
}

func TestCheckedListBoxSpaceTogglesTheBoxSeenByAssistiveTechnology(t *testing.T) {
	lb := newChecked().SetItems(fruit)
	tt := tester(t, lb)
	click(t, tt, "Apple")
	checkedOf := func() bool {
		boxes, _ := itemNodes(tt, ui.RoleCheckBox)
		return boxes[0].Checked
	}
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	tt.Frame()
	if !checkedOf() || !lb.IsChecked("a") {
		t.Fatalf("after Space: box %v, IsChecked %v", checkedOf(), lb.IsChecked("a"))
	}
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	tt.Frame()
	if checkedOf() || lb.IsChecked("a") {
		t.Fatal("a second Space did not uncheck")
	}
}

// A click on the box still checks the item.
func TestCheckedListBoxBoxClickStillChecks(t *testing.T) {
	lb := newChecked().SetItems(fruit)
	tt := tester(t, lb)
	x, y := checkBoxAt(t, tt, "Cherry")
	tt.ClickAt(x, y)
	tt.Frame()
	tt.Frame()
	if !lb.IsChecked("c") {
		t.Fatal("a click on the box did not check")
	}
	boxes, _ := itemNodes(tt, ui.RoleCheckBox)
	if !boxes[2].Checked {
		t.Fatalf("box %q not checked in the tree", boxes[2].Label)
	}
}

// ListBox and TreeView rows are as they were.
func TestListBoxAndTreeViewKeepTheirNodes(t *testing.T) {
	tt := tester(t, fruitBox())
	if items, _ := itemNodes(tt, ui.RoleListItem); len(items) != len(fruit) {
		t.Errorf("ListBox: %d items", len(items))
	}
	for _, n := range tt.Nodes() {
		if n.Checked {
			t.Errorf("a plain ListBox item is checked: %v", n)
		}
	}
	tr := tester(t, fileTree())
	if _, named := itemNodes(tr, ui.RoleTreeItem); named["Alpha"] != 2 {
		t.Errorf("TreeView: %d nodes named Alpha, the rows deliberately show their text", named["Alpha"])
	}
}

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

func boundsIn(t *testing.T, c Control, w, h int, q Query) ui.Rect {
	t.Helper()
	f := NewForm("t", w, h)
	f.SetContent(c)
	tt := TestForm(t, f)
	tt.Settle()
	return tt.Find(q).Bounds()
}

// The widget's own alignment wins over the stretch of a control that scrolls.
func TestFillKeepsTheAlignmentAChildAsksFor(t *testing.T) {
	// a box is one of the controls that stretch
	b := VBox(0).Add(Button("AE")).Align(End)
	r := boundsIn(t, VBox(4).Pack(b, true, true), 800, 600, ByLabel("AE"))
	if r.W > 200 || r.X+r.W < 790 {
		t.Errorf("Align(End) in a VBox: %v, want the button's width at the right", r)
	}
	v := VBox(0).Add(Button("VE")).VAlign(End)
	r = boundsIn(t, VBox(4).Pack(HBox(4).Pack(v, true, true), true, true), 800, 600, ByLabel("VE"))
	if r.H > 100 || r.Y+r.H < 590 {
		t.Errorf("VAlign(End) in a tall HBox: %v, want the button's height at the bottom", r)
	}
}

// Plain controls keep their height in a tall row.
func TestFillLeavesPlainControlsTheirHeightInATallRow(t *testing.T) {
	row := VBox(4).Pack(HBox(4).Pack(Button("Go"), true, true).Pack(TextBox(), true, true).Pack(Label("lbl"), true, true), true, true).Add(Label("status"))
	f := NewForm("t", 800, 600)
	f.SetContent(row)
	tt := TestForm(t, f)
	tt.Settle()
	for _, q := range []Query{ByLabel("Go"), ByRole(ui.RoleTextField), ByLabel("lbl")} {
		if r := tt.Find(q).Bounds(); r.H > 40 {
			t.Errorf("%v is %v high in a tall row", q, r.H)
		}
	}
}

func p137list(n int) *ListBoxWidget[litem, string] {
	items := make([]litem, n)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	return newListBox().SetItems(items)
}

// A list added to a row virtualizes too, and two in a split.
func TestListsAddedToARowOrSplitBuildOnlyTheRowsInView(t *testing.T) {
	for name, c := range map[string]func() Control{
		"HBox.Add":         func() Control { return VBox(4).Pack(HBox(4).Add(p137list(300)), true, true) },
		"HBox.Add(Expand)": func() Control { return VBox(4).Pack(HBox(4).Add(p137list(300).Expand()), true, true) },
		"SplitContainer x2": func() Control {
			return VBox(4).Pack(HBox(4).Pack(SplitContainer(p137list(300), p137list(300)), true, true), true, true)
		},
	} {
		f := NewForm("t", 800, 460)
		f.SetContent(c())
		tt := TestForm(t, f)
		tt.Settle()
		if got := len(tt.FindAll(ByRole(ui.RoleListItem))); got < 10 || got > 80 {
			t.Errorf("%s: %d rows built", name, got)
		}
	}
}

// A message dialog's text stays beside its icon, not at the top of the row.
func TestMessageTextStaysBesideItsIcon(t *testing.T) {
	done := runAsync(func() bool { Alert(nil, Info, "Beside", "msgtext"); return true })
	fw := waitFake(t, "Beside")
	var y float64
	for _, n := range accessNodes(fw) {
		if n.Label == "msgtext" {
			y = n.Bounds.Y
		}
	}
	if y < 60 { // centered in the row is ~84, at its top ~41
		t.Errorf("message text at y=%v: at the top of the row", y)
	}
	press(t, fw, "OK")
	received(t, done)
}

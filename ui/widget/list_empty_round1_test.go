package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Review round 1 of P2e: a ListBox with an empty state keeps on its list what
// render gives a control (the keyboard focus, a label's mnemonic, OnFocusOut,
// its name, description and disabled state), as a ListBox without one does,
// though it builds in a box around the list; and a ListView keeps its icon
// grid's box after SetEmptyState(nil).

// stateBox is a ListBox of three items, or none, with an empty state.
func stateBox(empty bool) *ListBoxWidget[litem, string] {
	lb := newListBox().SetItems(followItems(1, 3)).SetEmptyState(redState())
	if empty {
		lb.Clear()
	}
	return lb
}

// listAccess is what assistive technology reads of the list node.
func listAccess(t *testing.T, tt *ui.Tester) ui.AccessInfo {
	t.Helper()
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleList {
			return a
		}
	}
	t.Fatal("no list node")
	return ui.AccessInfo{}
}

func TestListBoxEmptyStateFocus(t *testing.T) {
	calmOff()
	for _, empty := range []bool{false, true} {
		lb := stateBox(empty).Expand()
		tt := tester(t, lb)
		lb.Focus()
		tt.Frame()
		tt.Frame()
		if !focusInList(tt) {
			t.Errorf("empty=%v: Focus() does not focus the list", empty)
		}
	}
}

func TestListBoxEmptyStateMnemonic(t *testing.T) {
	calmOff()
	withMac(t, false)
	for _, empty := range []bool{false, true} {
		f := NewForm("m", 400, 300)
		f.SetContent(Fields().Row("&Name", TextBox()).Row("&Files", stateBox(empty).SetSize(300, 150)))
		tt := TestForm(t, f)
		tt.Settle()
		tt.Key(ui.Alt, ui.KeyF)
		tt.Settle()
		if !focusInList(tt.Tester) {
			t.Errorf("empty=%v: Alt+F does not focus the list", empty)
		}
	}
}

// Review round 2: two rows share the access key F. Alt+F focuses the list box,
// and Alt+F again moves on to the next: the cycle asks the list, not the box
// around it, whether it has the focus. With two list boxes a third press comes
// back to the first. (After a TextBox it does not, with or without an empty
// state: the TextBox's element is not the one focused, which is older than
// P2e.)
func TestListBoxEmptyStateSharedMnemonic(t *testing.T) {
	calmOff()
	withMac(t, false)
	t.Run("then a text box", func(t *testing.T) {
		f := NewForm("m", 400, 300)
		f.SetContent(Fields().Row("&Files", stateBox(false).SetSize(300, 150)).Row("&Find", TextBox()))
		tt := TestForm(t, f)
		tt.Settle()
		for i, want := range []string{"list", "text box"} {
			tt.Key(ui.Alt, ui.KeyF)
			tt.Settle()
			inList, inText := focusInList(tt.Tester), tt.Find(ByRole(ui.RoleTextField)).Focused()
			if (want == "list") != inList || (want == "text box") != inText {
				t.Fatalf("Alt+F #%d: the list focused %v, the text box %v; want the %s", i+1, inList, inText, want)
			}
		}
	})
	t.Run("two list boxes", func(t *testing.T) {
		f := NewForm("m", 400, 400)
		f.SetContent(Fields().Row("&Files", stateBox(false).SetSize(300, 100)).
			Row("&Find", newListBox().SetItems(followItems(10, 3)).SetEmptyState(redState()).SetSize(300, 100)))
		tt := TestForm(t, f)
		tt.Settle()
		for i, want := range []string{"Files", "Find", "Files"} {
			tt.Key(ui.Alt, ui.KeyF)
			tt.Settle()
			var got []string
			for _, n := range tt.Tester.Nodes() {
				if n.Focused {
					got = append(got, n.Label)
				}
			}
			if len(got) != 1 || got[0] != want {
				t.Fatalf("Alt+F #%d focused %q, want the list %q", i+1, got, want)
			}
		}
	})
}

func TestListBoxEmptyStateOnFocusOut(t *testing.T) {
	calmOff()
	left := make(chan struct{}, 4)
	lb := stateBox(false).SetSize(300, 150).OnFocusOut(func() { left <- struct{}{} })
	tt := tester(t, VBox(0).Add(lb, TextBox()))
	click(t, tt, "r0002")
	if !focusInList(tt) {
		t.Fatal("clicking an item does not focus the list")
	}
	select {
	case <-left:
		t.Fatal("OnFocusOut ran while the list had the focus")
	default:
	}
	focus(tt, 1) // Tab, from the list to the text box
	received(t, left)
}

func TestListBoxEmptyStateAccessibleName(t *testing.T) {
	calmOff()
	for _, empty := range []bool{false, true} {
		lb := stateBox(empty).SetSize(300, 150).SetAccessibleName("Tệp").SetTooltip("Các tệp đã mở")
		tt := tester(t, lb)
		if a := listAccess(t, tt); a.Name != "Tệp" || a.Description != "Các tệp đã mở" {
			t.Errorf("empty=%v: the list reads %q, %q; want the name %q and the tooltip %q", empty, a.Name, a.Description, "Tệp", "Các tệp đã mở")
		}
	}
}

func TestListBoxEmptyStateFieldsName(t *testing.T) {
	calmOff()
	f := Fields().Row("Tệp", stateBox(false).SetSize(300, 150)).Describe("Các tệp đã mở")
	tt := tester(t, f)
	if a := listAccess(t, tt); a.Name != "Tệp" || a.Description != "Các tệp đã mở" {
		t.Errorf("the list reads %q, %q; want its row's label and description", a.Name, a.Description)
	}
}

func TestListBoxEmptyStateDisabled(t *testing.T) {
	calmOff()
	for _, empty := range []bool{false, true} {
		lb := stateBox(empty).SetSize(300, 150).SetDisabled(true)
		tt := tester(t, lb)
		if !listNode(t, tt).Disabled {
			t.Errorf("empty=%v: the disabled list box's list is not disabled", empty)
		}
	}
}

// The Focused variant of the list box's own style shows while the list has
// the focus, though the style lands on the box around it.
func TestListBoxEmptyStateFocusedStyle(t *testing.T) {
	calmOff()
	green := ui.RGB(0, 160, 0)
	lb := stateBox(false).SetSize(300, 150).Apply(NewStyle().Focused(NewStyle().Background(Fixed(green))))
	tt := tester(t, VBox(0).Add(lb, TextBox()).Padding(4))
	under := func() ui.Color {
		box := listBox(t, tt)
		return devicePx(tt.Image(), box.X+box.W/2, int(box.Y+box.H)-4, 1)
	}
	if got := under(); sameColor(got, green) {
		t.Fatal("the Focused style shows before the list has the focus")
	}
	click(t, tt, "r0002")
	tt.Frame()
	if got := under(); !sameColor(got, green) {
		t.Fatalf("under the rows of the focused list is %v, want the Focused style's %v", got, green)
	}
}

// Finding 2: a ListView's icon grid keeps its box after SetEmptyState(nil), as
// a ListBox does: the grid does not move, and keeps the focus.
func TestListViewEmptyStateNilKeepsTheGridsBox(t *testing.T) {
	calmOff()
	for _, m := range allModes[1:] {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode).SetEmptyState(redState()).Expand()
			tt := tester(t, lv)
			click(t, tt, "Banana")
			lv.SetEmptyState(nil)
			tt.Frame()
			if !focusInList(tt) {
				t.Fatal("the view lost the focus at SetEmptyState(nil): its grid moved out of its box")
			}
		})
	}
}

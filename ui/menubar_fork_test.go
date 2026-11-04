package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// barMenuApp is a menu bar of two menus on MenuBase, drawn with PopupMenu,
// that goes from one menu to the next as Across asks, opening it with
// OpenFirst in the next pass; Field had the focus before.
type barMenuApp struct {
	open   [2]bool
	first  int // the menu to open on its first item, -1 for none
	across []int
	chosen []string
}

func (a *barMenuApp) view(c *context) {
	coreColumn(c).Fill().Padding(20).Gap(10).AlignItems(Start).Children(func() {
		coreButton(c, "Field")
		coreRow(c).Role(RoleMenuBar).Children(func() {
			for i, name := range []string{"File", "Edit"} {
				m := coreMenuBase(c, &a.open[i])
				m.Trigger.Unfocusable().KeepFocus().Role(RoleMenuItem).Label(name).Padding(4).Children(func() { coreText(c, name) })
				if a.first == i {
					a.first = -1
					m.OpenFirst()
				}
				m.PopupMenu(func(m *Menu) {
					if m.Item(name + " one").Chosen() {
						a.chosen = append(a.chosen, name+" one")
					}
					m.Submenu(name+" more", func(m *Menu) {
						if m.Item(name + " deep").Chosen() {
							a.chosen = append(a.chosen, name+" deep")
						}
					})
					m.Item(name+" last").Shortcut(Ctrl, KeyS)
				})
				if d := m.Across(); d != 0 {
					a.across = append(a.across, d)
					a.open[i] = false
					a.first = (i + d + 2) % 2
				}
			}
		})
	})
}

func TestMenuPartsPopupMenuDrawsItemsAndReportsTheChoice(t *testing.T) {
	a := &barMenuApp{first: -1}
	tt := coreNewTester(a.view, 500, 400)
	tt.Click("File")
	if !a.open[0] || !tt.HasText("File one") || !tt.HasText(shortcutText("Ctrl+S")) {
		t.Fatalf("a click on File: open %v, texts %q", a.open, tt.Texts())
	}
	if err := tt.Click("File one"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.chosen, []string{"File one"}) || a.open[0] || tt.HasText("File one") {
		t.Errorf("chose %q, open %v", a.chosen, a.open)
	}
}

func TestMenuPartsOpenFirstStartsOnTheFirstItem(t *testing.T) {
	a := &barMenuApp{first: 1}
	tt := coreNewTester(a.view, 500, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	if !a.open[1] {
		t.Fatal("OpenFirst did not open the menu")
	}
	wantFocus(t, tt, "Edit one")
	tt.Key(0, KeyEnter)
	if !slices.Equal(a.chosen, []string{"Edit one"}) || a.open[1] {
		t.Errorf("Enter: chose %q, open %v", a.chosen, a.open)
	}
}

func TestMenuPartsAcrossAsksForTheMenuBeforeOrAfter(t *testing.T) {
	a := &barMenuApp{first: 0}
	tt := coreNewTester(a.view, 500, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	wantFocus(t, tt, "File one")
	// Right on an item opening no submenu: the menu after.
	tt.Key(0, KeyRight)
	if !slices.Equal(a.across, []int{1}) || a.open[0] || !a.open[1] {
		t.Fatalf("Right: across %v, open %v", a.across, a.open)
	}
	wantFocus(t, tt, "Edit one")
	// Right on a submenu item opens it, and asks for nothing.
	tt.Key(0, KeyDown)
	tt.Key(0, KeyRight)
	wantFocus(t, tt, "Edit deep")
	// Left in the submenu closes it; Right on its plain item goes on.
	tt.Key(0, KeyLeft)
	wantFocus(t, tt, "Edit more")
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	if !slices.Equal(a.across, []int{1, 1}) || !a.open[0] {
		t.Fatalf("Right in a submenu: across %v, open %v", a.across, a.open)
	}
	// Left in the menu: the one before.
	tt.Key(0, KeyLeft)
	if !slices.Equal(a.across, []int{1, 1, -1}) || !a.open[1] {
		t.Fatalf("Left: across %v, open %v", a.across, a.open)
	}
	// Right to left, the keys swap.
	tt.SetDirection(RightToLeft)
	tt.Frame()
	tt.Key(0, KeyLeft)
	if !slices.Equal(a.across, []int{1, 1, -1, 1}) {
		t.Errorf("Left right to left: across %v", a.across)
	}
}

// The focus goes back to where it was before the first menu opened, past
// the menus the bar went through, whose panels had it in between.
func TestFocusGoesBackPastAMenuThatClosedFirst(t *testing.T) {
	a := &barMenuApp{first: -1}
	tt := coreNewTester(a.view, 500, 400)
	tt.Click("Field")
	if !tt.Focused("Field") {
		t.Fatal("Field did not take the focus")
	}
	tt.Click("File")
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	if !a.open[1] || len(a.across) != 3 {
		t.Fatalf("open %v, across %v", a.open, a.across)
	}
	tt.Key(0, KeyEscape)
	if a.open[0] || a.open[1] {
		t.Fatalf("Escape: open %v", a.open)
	}
	if !tt.Focused("Field") {
		t.Error("the focus did not go back to Field")
	}
}

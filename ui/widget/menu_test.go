package widget

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
)

func TestMenuButtonChoosesAnItem(t *testing.T) {
	chosen := make(chan string, 1)
	b := MenuButton("Sort by").SetMenu(
		MenuItem{Text: "Name", OnClick: func() { chosen <- "Name" }},
		MenuItem{},
		MenuItem{Text: "Date", Checked: true, OnClick: func() { chosen <- "Date" }},
	)
	tt := tester(t, b)
	r := find(t, tt, "Sort by")
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if got := tt.Menu(); len(got) != 3 || got[1] != "-" {
		t.Fatalf("menu %q, want Name, -, Date", got)
	}
	tt.Release(r.X+r.W/2, r.Y+r.H/2)
	if err := tt.ChooseMenuItem("Date"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s := wait(t, chosen); s != "Date" {
		t.Fatalf("chose %q", s)
	}
}

func TestContextMenuOnALabelWithASubmenu(t *testing.T) {
	chosen := make(chan string, 1)
	l := Label("Report.pdf").ContextMenu(
		MenuItem{Text: "Open"},
		MenuItem{Text: "More", Children: []MenuItem{{Text: "Delete", OnClick: func() { chosen <- "Delete" }}}},
	)
	tt := tester(t, l)
	if err := tt.RightClick("Report.pdf"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("More", "Delete"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	wait(t, chosen)
}

// Review Focus 5: a menu item's handler opens a Confirm.
func TestMenuItemOpensAConfirm(t *testing.T) {
	answer := make(chan bool, 1)
	l := Label("Row").ContextMenu(MenuItem{Text: "Delete", OnClick: func() {
		answer <- Confirm(nil, Question, "Really delete?", "")
	}})
	tt := tester(t, l)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem("Delete"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	press(t, waitFake(t, "Really delete?"), "OK")
	if !wait(t, answer) {
		t.Fatal("Confirm in a menu handler returned false")
	}
}

func TestSetMenuBarClicks(t *testing.T) {
	opened := make(chan struct{}, 1)
	w := NewWindow("Menus", 300, 200).SetMenuBar(
		MenuItem{Text: "File", Children: []MenuItem{{Text: "Open", OnClick: func() { opened <- struct{}{} }}}},
	)
	w.Show()
	defer w.Close()
	fw := lastFake(t)
	var id int
	mygo.RunOnMain(func() {
		var find func(m *platform.Menu)
		find = func(m *platform.Menu) {
			if m == nil {
				return
			}
			for _, it := range m.Items {
				if it.Label == "Open" {
					id = it.ID
				}
				find(it.Submenu)
			}
		}
		find(fw.Menu())
		if id != 0 {
			fw.ClickMenuItem(id)
		}
	})
	if id == 0 {
		t.Fatal("no Open item in the window's menu bar")
	}
	wait(t, opened)
}

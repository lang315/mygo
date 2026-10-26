package widget

import (
	"bytes"
	"image"
	pngpkg "image/png"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

func TestNativeMenuIcons(t *testing.T) {
	check := ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M20 6 9 17l-5-5"/></svg>`))
	var png bytes.Buffer
	if err := pngpkg.Encode(&png, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	items := nativeMenu([]MenuItem{
		{Text: "Svg", Icon: check},
		{Text: "Png", IconPNG: png.Bytes()},
		{Text: "Both", Icon: check, IconPNG: png.Bytes()},
		{Text: "Sub", Icon: check, Children: []MenuItem{{Text: "In"}}},
		{Text: "Bad", IconPNG: []byte("not a png")},
	})
	svg := svgMenuIcon(check, IsDark())
	if items[0].Icon == nil || items[0].Icon != svg {
		t.Errorf("Svg: icon %p, want the cached icon of the SVG %p", items[0].Icon, svg)
	}
	if items[1].Icon == nil || items[1].Icon == svg {
		t.Errorf("Png: icon %p, want one made of the PNG", items[1].Icon)
	}
	if items[2].Icon == nil || items[2].Icon == svg {
		t.Errorf("Both: the PNG must win, got %p", items[2].Icon)
	}
	if items[3].Icon != svg {
		t.Error("Sub: a submenu item lost its icon")
	}
	if items[4].Icon != nil {
		t.Error("Bad: bytes that are not a PNG made an icon")
	}
	if again := nativeMenu([]MenuItem{{Text: "Svg", Icon: check}}); again[0].Icon != svg {
		t.Error("the icon of an SVG is made again each time a menu is set, not once")
	}
}

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
	if s := received(t, chosen); s != "Date" {
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
	received(t, chosen)
}

// A menu item's handler opens a Confirm.
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
	if !received(t, answer) {
		t.Fatal("Confirm in a menu handler returned false")
	}
}

func TestSetMenuBarClicks(t *testing.T) {
	opened := make(chan struct{}, 1)
	w := NewForm("Menus", 300, 200).SetMenuBar(
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
	received(t, opened)
}

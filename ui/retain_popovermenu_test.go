package ui

import "testing"

// Rows whose context menu MyGo draws with Panel and Item options
// (PopoverMenuOptions): the menu opened from a kept row keeps the
// options' look, as from one built in every frame.
func TestRetainRowsWithAStyledPopoverMenu(t *testing.T) {
	red, blue := RGB(200, 30, 30), RGB(30, 30, 200)
	o := PopoverMenuOptions{
		Panel: func(panel Element) { panel.Background(red) },
		Item:  func(item Element) { item.Background(blue) },
	}
	p := newPair(t, 300, 400, hoverRowsScene(func(c *Context, i int) {
		Row(c).Padding(4).Children(func() { Textf(c, "row %d", i) }).PopoverMenu(func(m *Menu) {
			m.Item("Copy")
			m.Submenu("More", func(m *Menu) { m.Item("Elsewhere") })
		}, o)
	}))
	p.idle(3)
	p.both(func(tt *Tester) {
		if err := tt.RightClick("row 3"); err != nil {
			t.Fatal(err)
		}
	})
	p.frame("open")
	p.frame("open 2")
	if !p.kept.HasText("Copy") {
		t.Fatalf("no menu: texts %q", p.kept.Texts())
	}
	p.both(func(tt *Tester) { tt.Key(0, KeyDown); tt.Key(0, KeyDown); tt.Key(0, KeyRight) })
	p.frame("submenu")
	p.frame("submenu 2")
	if !p.kept.HasText("Elsewhere") {
		t.Fatalf("no submenu: texts %q", p.kept.Texts())
	}
	p.both(func(tt *Tester) { tt.Key(0, KeyEscape); tt.Key(0, KeyEscape) })
	p.frame("closed")
	p.frame("closed 2")
}

//go:build darwin

package darwin

import (
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/internal/platform"
)

// linkMenuExtras links the badge column and KeepOpen items for a test, as
// importing package macmenu does, and unlinks them after it.
func linkMenuExtras(t *testing.T) {
	t.Helper()
	setMenuExtras(t, LinkMenuExtras)
}

// unlinkMenuExtras runs a test as an app that does not import macmenu.
func unlinkMenuExtras(t *testing.T) {
	t.Helper()
	setMenuExtras(t, func() {
		registerMenuExtras, menuExtraMetrics, setMenuExtraTitle, updateMenuExtras = nil, nil, nil, nil
	})
}

func setMenuExtras(t *testing.T, set func()) {
	r, m, s, u := registerMenuExtras, menuExtraMetrics, setMenuExtraTitle, updateMenuExtras
	t.Cleanup(func() {
		registerMenuExtras, menuExtraMetrics, setMenuExtraTitle, updateMenuExtras = r, m, s, u
	})
	set()
}

// countClicks is an app handler that counts the choices of menu items.
type countClicks struct {
	platform.AppHandler
	clicks map[int]int
}

func (c *countClicks) MenuItemClicked(id int) { c.clicks[id]++ }

// TestMenuBadgeWithoutExtras checks that, without macmenu, a badge follows
// the label in the item's title, as the other backends show it, with no
// tab stop.
func TestMenuBadgeWithoutExtras(t *testing.T) {
	load()
	unlinkMenuExtras(t)
	registerMenuTarget()
	withPool(func() {
		b := &Backend{}
		m := &platform.Menu{Items: []*platform.MenuItem{
			{ID: 1, Label: "Inbox", Badge: "3", Visible: true, Enabled: true},
		}}
		owner := newOwner()
		menu := b.buildMenu(m, "", owner)
		defer dropOwner(owner)
		item := send(menu, "itemAtIndex:", 0)
		if got := goString(send(item, "title")); got != "Inbox  3" {
			t.Errorf("title = %q, want %q", got, "Inbox  3")
		}
		if tab := titleTab(item); tab != 0 {
			t.Errorf("badge has a tab stop at %v", tab)
		}

		b.UpdateMenuItem(&platform.MenuItem{ID: 1, Label: "Inbox", Badge: "4", Visible: true, Enabled: true})
		if got := goString(send(item, "title")); got != "Inbox  4" {
			t.Errorf("title after update = %q, want %q", got, "Inbox  4")
		}
		if tab := titleTab(item); tab != 0 {
			t.Errorf("badge has a tab stop at %v after update", tab)
		}
	})
}

// TestMenuKeepOpenWithoutExtras checks that, without macmenu, a KeepOpen
// item is an item NSMenu draws, with no view of MyGo's: choosing it fires
// its click once, through the action that ends the menu's tracking.
func TestMenuKeepOpenWithoutExtras(t *testing.T) {
	load()
	unlinkMenuExtras(t)
	registerMenuTarget()
	h := &countClicks{clicks: map[int]int{}}
	b := &Backend{h: h}
	prev := theBackend
	theBackend = b
	t.Cleanup(func() { theBackend = prev })
	withPool(func() {
		b.menuTarget = send(send(class("MyGoMenuTarget"), "alloc"), "init")
		defer release(b.menuTarget)
		m := &platform.Menu{Items: []*platform.MenuItem{
			{ID: 42, Label: "Node", Badge: "45 ms", KeepOpen: true, Visible: true, Enabled: true},
		}}
		owner := newOwner()
		menu := b.buildMenu(m, "", owner)
		defer dropOwner(owner)
		item := send(menu, "itemAtIndex:", 0)
		if view := send(item, "view"); view != 0 {
			t.Fatal("KeepOpen item has a view without macmenu")
		}
		if send(menu, "delegate") != 0 {
			t.Error("menu has a highlight delegate without macmenu")
		}
		if got := goString(send(item, "title")); got != "Node  45 ms" {
			t.Errorf("title = %q", got)
		}
		if objc.SEL(send(item, "action")) != sel("mygoMenuItemClicked:") || send(item, "target") != b.menuTarget {
			t.Fatal("KeepOpen item does not choose through the menu's action")
		}
		// NSMenu sends the item's action to its target; the test has no
		// NSApp for performActionForItemAtIndex: to send it through.
		send(send(item, "target"), "mygoMenuItemClicked:", uintptr(item))
		if got := h.clicks[42]; got != 1 {
			t.Errorf("clicks = %d, want 1", got)
		}
	})
}

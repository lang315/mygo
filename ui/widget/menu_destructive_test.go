package widget

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
)

// The window's menu bar gets the flag in its native items.
func TestMenuItemDestructiveReachesTheWindowMenuBar(t *testing.T) {
	was := macMenus
	macMenus = false
	t.Cleanup(func() { macMenus = was })
	w := NewForm("Bar", 300, 200).SetMenuBar(MenuItem{Text: "File", Children: []MenuItem{
		{Text: "Open"}, {Text: "Delete", Destructive: true}}})
	w.Show()
	t.Cleanup(w.Close)
	fw := lastFake(t)
	mygo.RunOnMain(func() {})
	if del, open := findItem(fw.Menu(), "Delete"), findItem(fw.Menu(), "Open"); del == nil || !del.Destructive || open == nil || open.Destructive {
		t.Fatalf("Delete %+v, Open %+v: want the flag on Delete only", del, open)
	}
}

// A context menu of a control, shown by the window, has it too.
func TestMenuItemDestructiveReachesAContextMenu(t *testing.T) {
	l := Label("row").ContextMenu(MenuItem{Text: "Rename"}, MenuItem{Text: "Delete 3", Destructive: true})
	showFrame(t, l)
	s := lastFake(t).FakeSurface()
	before := len(fb.Popups())
	mygo.RunOnMain(func() {
		s.Send(platform.SurfaceEvent{Kind: platform.PointerDown, X: 20, Y: 10, Button: 1})
		s.Send(platform.SurfaceEvent{Kind: platform.PointerUp, X: 20, Y: 10, Button: 1})
		s.Frame()
	})
	mygo.RunOnMain(func() {}) // the menu, posted
	popups := fb.Popups()
	if len(popups) != before+1 {
		t.Fatalf("%d context menus shown, want one more than %d", len(popups), before)
	}
	items := popups[len(popups)-1].Menu.Items
	if items[0].Destructive || !items[1].Destructive {
		t.Fatalf("Rename %v, Delete %v: want false and true", items[0].Destructive, items[1].Destructive)
	}
}

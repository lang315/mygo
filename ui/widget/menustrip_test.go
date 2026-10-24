package widget

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
)

// findItem returns the item labelled label in a platform menu, and its
// parents' labels' absence: nil when there is none.
func findItem(m *platform.Menu, label string) *platform.MenuItem {
	if m == nil {
		return nil
	}
	for _, it := range m.Items {
		if it.Label == label {
			return it
		}
		if got := findItem(it.Submenu, label); got != nil {
			return got
		}
	}
	return nil
}

// lastUpdate returns the last update of the item with the given label that
// reached the backend, after the main thread ran what was posted to it.
func lastUpdate(t *testing.T, label string) *platform.MenuItem {
	t.Helper()
	mygo.RunOnMain(func() {}) // updates are posted to the main thread
	var got *platform.MenuItem
	for _, u := range fb.MenuUpdates() {
		if u.Label == label {
			got = u
		}
	}
	return got
}

func windowStrip(t *testing.T, items ...*MenuStripItem) (*MenuStrip, func() *platform.Menu) {
	t.Helper()
	was := macMenus
	macMenus = false
	t.Cleanup(func() { macMenus = was })
	m := NewMenuStrip(items...)
	w := NewForm("Strip", 300, 200).MenuStrip(m)
	w.Show()
	t.Cleanup(w.Close)
	fw := lastFake(t)
	return m, func() *platform.Menu { return fw.Menu() }
}

func TestMenuStripReachesTheWindowMenu(t *testing.T) {
	_, menu := windowStrip(t, StripItem("File", StripItem("Open"), StripSeparator(), StripItem("Quit")))
	if findItem(menu(), "File") == nil || findItem(menu(), "Open") == nil || findItem(menu(), "Quit") == nil {
		t.Fatalf("the window's menu lacks File, Open or Quit: %+v", menu())
	}
}

func TestMenuStripClickIsPostedToTheEventGoroutine(t *testing.T) {
	ran := make(chan bool, 1)
	_, menu := windowStrip(t, StripItem("File", StripItem("Open").OnClick(func() { ran <- onEvents() })))
	id := findItem(menu(), "Open").ID
	mygo.RunOnMain(func() { lastFake(t).ClickMenuItem(id) })
	if !received(t, ran) {
		t.Fatal("OnClick did not run on the event goroutine")
	}
}

func TestMenuStripItemsChangeAtRunTime(t *testing.T) {
	open := StripItem("Open")
	check := StripCheck("Word wrap", false)
	windowStrip(t, StripItem("File", open), StripItem("View", check))

	open.SetEnabled(false)
	if u := lastUpdate(t, "Open"); u == nil || u.Enabled {
		t.Fatalf("SetEnabled(false) reached the backend as %+v", u)
	}
	if open.Enabled() {
		t.Fatal("Enabled() is true")
	}
	check.SetChecked(true)
	if u := lastUpdate(t, "Word wrap"); u == nil || !u.Checked {
		t.Fatalf("SetChecked(true) reached the backend as %+v", u)
	}
	if !check.Checked() {
		t.Fatal("Checked() is false")
	}
	open.SetText("Open…")
	if u := lastUpdate(t, "Open…"); u == nil {
		t.Fatal("SetText did not reach the backend")
	}
	open.SetShortcut("CmdOrCtrl+O")
	if u := lastUpdate(t, "Open…"); u.Accelerator != "CmdOrCtrl+O" {
		t.Fatalf("shortcut %q", u.Accelerator)
	}
}

func TestMenuStripCheckItemTogglesBeforeItsHandler(t *testing.T) {
	seen := make(chan bool, 1)
	var check *MenuStripItem
	check = StripCheck("Word wrap", false).OnClick(func() { seen <- check.Checked() })
	_, menu := windowStrip(t, StripItem("View", check))
	id := findItem(menu(), "Word wrap").ID
	mygo.RunOnMain(func() { lastFake(t).ClickMenuItem(id) })
	if !received(t, seen) {
		t.Fatal("the handler saw the item unchecked: it toggles before the handler")
	}
}

func TestMenuStripOnMacIsTheAppMenu(t *testing.T) {
	was := macMenus
	macMenus = true
	t.Cleanup(func() { macMenus = was })
	m := NewMenuStrip(StripRole(mygo.RoleAppMenu), StripItem("Tools", StripItem("Run")))
	w := NewForm("Mac", 300, 200).MenuStrip(m)
	w.Show()
	t.Cleanup(w.Close)
	mygo.RunOnMain(func() {})
	if findItem(fb.AppMenu(), "Run") == nil {
		t.Fatalf("the application menu lacks Run: %+v", fb.AppMenu())
	}
	if lastFake(t).Menu() != nil && findItem(lastFake(t).Menu(), "Run") != nil {
		t.Fatal("on macOS the window must not get a menu of its own")
	}
}

func TestMenuStripChangedFromAnyGoroutineIsRaceFree(t *testing.T) {
	item := StripItem("Run")
	windowStrip(t, StripItem("Tools", item))
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 100 {
			item.SetEnabled(i%2 == 0)
			item.OnClick(func() {})
		}
	}()
	for range 20 {
		mygo.RunOnMain(func() {})
	}
	<-done
}

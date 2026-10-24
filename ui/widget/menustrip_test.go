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

// onMac runs the test as if on macOS, where the app has one menu bar, with
// a menu of its own that a form's strip stands in for and gives back.
func onMac(t *testing.T) (base *mygo.Menu) {
	t.Helper()
	was, prev := macMenus, mygo.App.Menu()
	macMenus = true
	base = mygo.NewMenu([]*mygo.MenuItem{{Label: "Base"}})
	mygo.App.SetMenu(base)
	t.Cleanup(func() { macMenus = was; mygo.App.SetMenu(prev) })
	return base
}

// appMenu returns the app's menu in the backend, after the main thread ran
// what was posted to it.
func appMenu() *platform.Menu {
	mygo.RunOnMain(func() {})
	return fb.AppMenu()
}

func focusWindow(fw interface{ Focus() }) { mygo.RunOnMain(fw.Focus) }

func shownForm(t *testing.T, title, item string) *Form {
	t.Helper()
	f := NewForm(title, 300, 200).MenuStrip(NewMenuStrip(StripItem(item)))
	f.Show()
	t.Cleanup(f.Close)
	return f
}

// A form never shown does not take the menu bar from the one on screen.
func TestMacAFormNeverShownDoesNotReplaceTheAppMenu(t *testing.T) {
	onMac(t)
	shownForm(t, "A", "AItem")
	settings := NewForm("Settings", 300, 200).MenuStrip(NewMenuStrip(StripItem("SettingsItem")))
	_ = settings
	if m := appMenu(); findItem(m, "AItem") == nil || findItem(m, "SettingsItem") != nil {
		t.Fatalf("the app menu is %+v: A's, not the unshown form's", m)
	}
}

// A closed form's menu does not stay: the app's own comes back.
func TestMacClosingTheFormGivesTheAppMenuBack(t *testing.T) {
	onMac(t)
	a := shownForm(t, "A", "AItem")
	if findItem(appMenu(), "AItem") == nil {
		t.Fatal("A's menu is not the app's while A is shown")
	}
	a.Close()
	if m := appMenu(); findItem(m, "AItem") != nil || findItem(m, "Base") == nil {
		t.Fatalf("after A closed the app menu is %+v, want the app's own", m)
	}
}

// A form that never had a strip, set to nil, takes nothing from the one that
// has: on macOS an empty menu bar loses Cmd+Q and Cmd+C, and the default does
// not come back.
func TestMacNilStripOfAnotherFormLeavesTheAppMenu(t *testing.T) {
	onMac(t)
	shownForm(t, "A", "AItem")
	NewForm("B", 300, 200).MenuStrip(nil)
	if findItem(appMenu(), "AItem") == nil {
		t.Fatal("B's MenuStrip(nil) removed A's menu")
	}
}

// The app menu is the focused form's strip, and a closed form hands it on.
func TestMacTheAppMenuFollowsFocusAndHandsOnWhenTheOwnerCloses(t *testing.T) {
	onMac(t)
	a := shownForm(t, "A", "AItem")
	fa := lastFake(t)
	b := shownForm(t, "B", "BItem")
	fb2 := lastFake(t)
	focusWindow(fb2)
	if m := appMenu(); findItem(m, "BItem") == nil {
		t.Fatalf("B has the focus: app menu %+v", m)
	}
	focusWindow(fa)
	if m := appMenu(); findItem(m, "AItem") == nil {
		t.Fatalf("A has the focus: app menu %+v", m)
	}
	a.Close() // the owner closes: B's, the other form with a strip, takes over
	if m := appMenu(); findItem(m, "BItem") == nil {
		t.Fatalf("A closed: app menu %+v, want B's", m)
	}
	b.Close()
	if m := appMenu(); findItem(m, "Base") == nil {
		t.Fatalf("both closed: app menu %+v, want the app's own", m)
	}
}

// A strip set on the form that owns the menu changes it at once; nil on the
// owner gives it back.
func TestMacTheOwnersStripChangesAndRemovesTheAppMenu(t *testing.T) {
	onMac(t)
	a := shownForm(t, "A", "AItem")
	a.MenuStrip(NewMenuStrip(StripItem("Newer")))
	if m := appMenu(); findItem(m, "Newer") == nil || findItem(m, "AItem") != nil {
		t.Fatalf("app menu %+v after a new strip", m)
	}
	a.MenuStrip(nil)
	if m := appMenu(); findItem(m, "Newer") != nil || findItem(m, "Base") == nil {
		t.Fatalf("app menu %+v after nil, want the app's own", m)
	}
}

// On Windows and Linux the window's bar is the last of SetMenuBar and
// MenuStrip: nil removes a strip, and leaves what SetMenuBar set.
func TestMenuStripNilLeavesTheMenuBarSetMenuBarSet(t *testing.T) {
	was := macMenus
	macMenus = false
	t.Cleanup(func() { macMenus = was })
	w := NewForm("Bar", 300, 200).SetMenuBar(MenuItem{Text: "Plain"})
	w.Show()
	t.Cleanup(w.Close)
	fw := lastFake(t)
	mygo.RunOnMain(func() {})
	w.MenuStrip(nil)
	mygo.RunOnMain(func() {})
	if findItem(fw.Menu(), "Plain") == nil {
		t.Fatalf("MenuStrip(nil) wiped the SetMenuBar menu: %+v", fw.Menu())
	}
	w.MenuStrip(NewMenuStrip(StripItem("Strip")))
	mygo.RunOnMain(func() {})
	if findItem(fw.Menu(), "Strip") == nil || findItem(fw.Menu(), "Plain") != nil {
		t.Fatalf("a strip does not replace the bar: %+v", fw.Menu())
	}
	w.MenuStrip(nil)
	mygo.RunOnMain(func() {})
	if findItem(fw.Menu(), "Strip") != nil {
		t.Fatalf("nil did not remove the strip: %+v", fw.Menu())
	}
	w.SetMenuBar(MenuItem{Text: "Again"}) // the last one set wins
	w.MenuStrip(nil)
	mygo.RunOnMain(func() {})
	if findItem(fw.Menu(), "Again") == nil {
		t.Fatalf("the bar set after the strip is gone: %+v", fw.Menu())
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

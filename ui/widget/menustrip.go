package widget

import (
	"runtime"
	"sync/atomic"

	"github.com/egoist/mygo"
)

// macMenus is whether a window's main menu is the app's, as on macOS;
// tests replace it.
var macMenus = runtime.GOOS == "darwin"

// MenuStrip is the main menu of a window: the app's menu bar on macOS, the
// window's own on Windows and Linux (Form.MenuStrip). It is made of
// MenuStripItems, which the program changes while the app runs: the native
// menu follows.
type MenuStrip struct {
	menu *mygo.Menu
}

// NewMenuStrip makes a main menu of items, the top ones of the bar, each
// with the items of its menu.
func NewMenuStrip(items ...*MenuStripItem) *MenuStrip {
	tmpl := make([]*mygo.MenuItem, len(items))
	for i, it := range items {
		tmpl[i] = it.it
	}
	return &MenuStrip{menu: mygo.NewMenu(tmpl)}
}

// MenuStripItem is an item of a MenuStrip, or a menu it holds.
type MenuStripItem struct {
	it    *mygo.MenuItem
	click atomic.Pointer[func()]
}

// newStripItem makes an item of template t, whose click posts what OnClick
// set.
func newStripItem(t *mygo.MenuItem) *MenuStripItem {
	s := &MenuStripItem{it: t}
	t.Click = func(*mygo.MenuItem, *mygo.Window) {
		if h := s.click.Load(); h != nil {
			Post(*h)
		}
	}
	return s
}

// StripItem makes an item showing text. With children it is a menu that
// opens them, as File with Open and Quit.
func StripItem(text string, children ...*MenuStripItem) *MenuStripItem {
	t := &mygo.MenuItem{Label: text}
	for _, c := range children {
		t.Submenu = append(t.Submenu, c.it)
	}
	return newStripItem(t)
}

// StripCheck makes an item with a check mark, on or off. Choosing it
// toggles the mark before its OnClick runs.
func StripCheck(text string, checked bool) *MenuStripItem {
	return newStripItem(&mygo.MenuItem{Label: text, Type: mygo.MenuItemCheckbox, Checked: checked})
}

// StripSeparator makes a line between items.
func StripSeparator() *MenuStripItem { return &MenuStripItem{it: mygo.Separator()} }

// StripRole makes an item with the behavior, label and shortcut of a role,
// as mygo.RoleAppMenu, RoleEditMenu or RoleQuit give the system's own. On
// macOS a menu that is not the default one should hold them, for the app
// menu and the Edit menu.
func StripRole(role mygo.MenuRole) *MenuStripItem {
	return newStripItem(&mygo.MenuItem{Role: role})
}

// OnClick runs fn on the event goroutine when the user chooses the item.
func (s *MenuStripItem) OnClick(fn func()) *MenuStripItem {
	s.click.Store(&fn)
	return s
}

// SetText changes the item's text.
func (s *MenuStripItem) SetText(text string) *MenuStripItem { s.it.SetLabel(text); return s }

// SetShortcut sets the item's keyboard shortcut, such as "CmdOrCtrl+Shift+N".
func (s *MenuStripItem) SetShortcut(shortcut string) *MenuStripItem {
	s.it.SetAccelerator(shortcut)
	return s
}

// SetEnabled enables or disables the item.
func (s *MenuStripItem) SetEnabled(on bool) *MenuStripItem { s.it.SetEnabled(on); return s }

// Enabled reports whether the item is enabled.
func (s *MenuStripItem) Enabled() bool { return s.it.IsEnabled() }

// SetDestructive marks the item as one that destroys something, as
// MenuItem.Destructive does; where mygo draws the menu it shows in the
// danger color.
func (s *MenuStripItem) SetDestructive(on bool) *MenuStripItem { s.it.SetDestructive(on); return s }

// Destructive reports whether the item is marked destructive.
func (s *MenuStripItem) Destructive() bool { return s.it.IsDestructive() }

// SetVisible shows or hides the item.
func (s *MenuStripItem) SetVisible(on bool) *MenuStripItem { s.it.SetVisible(on); return s }

// SetChecked checks or unchecks a StripCheck item.
func (s *MenuStripItem) SetChecked(on bool) *MenuStripItem { s.it.SetChecked(on); return s }

// Checked reports whether a StripCheck item is checked.
func (s *MenuStripItem) Checked() bool { return s.it.IsChecked() }

// MenuStrip sets the window's main menu. On macOS the menu bar is the
// app's, and it is the strip of the form that has the focus: a form's strip
// takes it when the form is shown with none taken, and when its window gets
// the focus; when that form closes, the next form with a strip takes it, or
// the app's own menu comes back. A form not shown takes nothing, and nil on
// a form that does not hold the menu bar changes nothing. On Windows and
// Linux it is the window's own menu bar, above its content, the last set of
// SetMenuBar and MenuStrip; nil removes a strip and leaves a bar SetMenuBar
// set. Items of the strip change the native menu as the program sets them.
func (w *Form) MenuStrip(m *MenuStrip) *Form {
	if macMenus {
		w.appMenuStrip(m)
		return w
	}
	var had bool
	w.change(func() {
		had = w.strip != nil
		w.strip = m
		if m != nil {
			w.menu = nil // the strip replaces the bar SetMenuBar set
		}
	})
	if m != nil || had {
		w.do(func(n *mygo.Window) { n.SetMenu(m.native()) })
	}
	return w
}

var (
	appMenuOwner *Form      // the form whose strip is the app's menu, under mu
	appMenuBase  *mygo.Menu // the app's own menu, which a strip stands in for
	appMenuTaken bool       // appMenuBase is the menu before a strip took over
	appMenuGen   int        // changes with every change of the owner or its strip
)

// appMenuStrip is MenuStrip on macOS.
func (w *Form) appMenuStrip(m *MenuStrip) {
	var owns bool
	w.change(func() {
		w.strip = m
		owns = appMenuOwner == w
		appMenuGen++
	})
	switch {
	case owns && m == nil:
		releaseAppMenu(w)
	case owns:
		syncAppMenu()
	case m != nil:
		if n := w.Native(); n != nil && (n.IsFocused() || !appMenuHeld()) {
			claimAppMenu(w)
		}
	}
}

// appMenuHeld reports whether a form holds the app's menu.
func appMenuHeld() bool { mu.Lock(); defer mu.Unlock(); return appMenuOwner != nil }

// claimAppMenu gives the app's menu to w's strip, if it has one.
func claimAppMenu(w *Form) {
	mu.Lock()
	ok := w.strip != nil
	if ok {
		appMenuOwner = w
		appMenuGen++
	}
	mu.Unlock()
	if ok {
		syncAppMenu()
	}
}

// claimAppMenuIfFree is claimAppMenu while no form holds the menu.
func claimAppMenuIfFree(w *Form) {
	if !appMenuHeld() {
		claimAppMenu(w)
	}
}

// releaseAppMenu takes the app's menu from w if it holds it: to another form
// that is shown with a strip, else back to the app's own.
func releaseAppMenu(w *Form) {
	mu.Lock()
	if appMenuOwner != w {
		mu.Unlock()
		return
	}
	appMenuOwner = nil
	for _, f := range shown {
		if f != w && f.strip != nil {
			appMenuOwner = f
			break
		}
	}
	appMenuGen++
	mu.Unlock()
	syncAppMenu()
}

// syncAppMenu sets the app's menu to the owner's strip, or gives back the
// menu the app had. It calls into mygo without the lock, which a view on the
// main thread takes; if the owner changed meanwhile, it goes round again.
func syncAppMenu() {
	for {
		mu.Lock()
		gen := appMenuGen
		var want *mygo.Menu
		switch {
		case appMenuOwner != nil && appMenuOwner.strip != nil:
			if !appMenuTaken {
				appMenuBase, appMenuTaken = mygo.App.Menu(), true
			}
			want = appMenuOwner.strip.menu
		case appMenuTaken:
			want, appMenuTaken = appMenuBase, false
		default:
			mu.Unlock()
			return
		}
		mu.Unlock()
		mygo.App.SetMenu(want)
		mu.Lock()
		done := gen == appMenuGen
		mu.Unlock()
		if done {
			return
		}
	}
}

// native returns the mygo menu of m, nil for no strip.
func (m *MenuStrip) native() *mygo.Menu {
	if m == nil {
		return nil
	}
	return m.menu
}

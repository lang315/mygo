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

// SetVisible shows or hides the item.
func (s *MenuStripItem) SetVisible(on bool) *MenuStripItem { s.it.SetVisible(on); return s }

// SetChecked checks or unchecks a StripCheck item.
func (s *MenuStripItem) SetChecked(on bool) *MenuStripItem { s.it.SetChecked(on); return s }

// Checked reports whether a StripCheck item is checked.
func (s *MenuStripItem) Checked() bool { return s.it.IsChecked() }

// MenuStrip sets the window's main menu: on macOS the app's menu bar, which
// every window shares, so the last one set is the one shown; on Windows and
// Linux the window's own menu bar, above its content. nil removes it. Items
// of the strip change the native menu as the program sets them.
func (w *Form) MenuStrip(m *MenuStrip) *Form {
	w.change(func() { w.strip = m })
	if macMenus {
		mygo.App.SetMenu(m.native())
	} else {
		w.do(func(n *mygo.Window) { n.SetMenu(m.native()) })
	}
	return w
}

// native returns the mygo menu of m, nil for no strip.
func (m *MenuStrip) native() *mygo.Menu {
	if m == nil {
		return nil
	}
	return m.menu
}

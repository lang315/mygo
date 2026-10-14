package widget

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// MenuItem is an item of a menu: Text, or a separator when Text is "".
// OnClick runs on the event goroutine when the user chooses it; an item
// with Children opens them as a submenu. Checked shows a check mark;
// Disabled greys the item out. Items have no icons (spec R4).
type MenuItem struct {
	Text     string
	OnClick  func()
	Children []MenuItem
	Checked  bool
	Disabled bool
}

// buildMenu adds items to a ui menu, posting the OnClick of the item
// chosen.
func buildMenu(m *ui.Menu, items []MenuItem) {
	for _, it := range items {
		switch {
		case it.Text == "":
			m.Separator()
		case len(it.Children) > 0:
			children := it.Children
			m.Submenu(it.Text, func(m *ui.Menu) { buildMenu(m, children) }).Disabled(it.Disabled)
		default:
			if m.Item(it.Text).Checked(it.Checked).Disabled(it.Disabled).Chosen() && it.OnClick != nil {
				Post(it.OnClick)
			}
		}
	}
}

// nativeMenu makes mygo menu items of items, for a window's menu bar.
func nativeMenu(items []MenuItem) []*mygo.MenuItem {
	out := make([]*mygo.MenuItem, 0, len(items))
	for _, it := range items {
		if it.Text == "" {
			out = append(out, mygo.Separator())
			continue
		}
		mi := &mygo.MenuItem{Label: it.Text, Disabled: it.Disabled}
		switch {
		case len(it.Children) > 0:
			mi.Type, mi.Submenu = mygo.MenuItemSubmenu, nativeMenu(it.Children)
		case it.Checked:
			mi.Type, mi.Checked = mygo.MenuItemCheckbox, true
		}
		if h := it.OnClick; h != nil {
			mi.Click = func(*mygo.MenuItem, *mygo.Window) { Post(h) }
		}
		out = append(out, mi)
	}
	return out
}

// MenuButtonWidget is a button that opens a menu.
type MenuButtonWidget struct {
	common[MenuButtonWidget]
	text  string
	items []MenuItem
}

// MenuButton makes a button showing text and an arrow, opening a menu.
func MenuButton(text string) *MenuButtonWidget {
	b := &MenuButtonWidget{text: text}
	b.init(b)
	return b
}

func (b *MenuButtonWidget) SetText(s string) *MenuButtonWidget {
	change(func() { b.text = s })
	return b
}

// SetMenu sets the items of the menu; set it again when they change.
func (b *MenuButtonWidget) SetMenu(items ...MenuItem) *MenuButtonWidget {
	change(func() { b.items = items })
	return b
}

func (b *MenuButtonWidget) build(c *ui.Context) ui.Element {
	items := b.items // a copy of the header: SetMenu replaces, never edits
	return ui.MenuButton(c, b.text, func(m *ui.Menu) { buildMenu(m, items) })
}

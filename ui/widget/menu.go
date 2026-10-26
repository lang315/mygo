package widget

import (
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// MenuItem is an item of a menu: Text, or a separator when Text is "".
// OnClick runs on the event goroutine when the user chooses it; an item
// with Children opens them as a submenu. Checked shows a check mark;
// Disabled greys the item out. Icon shows an SVG before the text, in the
// color of the text when it paints with currentColor only; IconPNG shows a
// PNG instead (32×32 pixels, shown at 16×16 points). Destructive marks an
// item that destroys, as Delete: it shows in the danger color and semibold
// where mygo draws the menu itself (Windows 11, the web), and as a normal
// item in native menus (macOS, Linux, Windows 10), which have no role for it.
type MenuItem struct {
	Text     string
	OnClick  func()
	Children []MenuItem
	Checked  bool
	Disabled bool
	Icon     *ui.SVG
	IconPNG  []byte

	Destructive bool
}

// buildMenu adds items to a ui menu, posting the OnClick of the item
// chosen.
func buildMenu(m *ui.Menu, items []MenuItem) {
	for _, it := range items {
		switch {
		case it.Text == "":
			m.Separator()
		case len(it.Children) > 0:
			m.Submenu(it.Text, func(m *ui.Menu) { buildMenu(m, it.Children) }).Disabled(it.Disabled).Icon(it.Icon).IconPNG(it.IconPNG).Destructive(it.Destructive)
		default:
			if m.Item(it.Text).Checked(it.Checked).Disabled(it.Disabled).Icon(it.Icon).IconPNG(it.IconPNG).Destructive(it.Destructive).Chosen() && it.OnClick != nil {
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
		mi := &mygo.MenuItem{Label: it.Text, Disabled: it.Disabled, Destructive: it.Destructive}
		switch {
		case len(it.Children) > 0:
			mi.Type, mi.Submenu = mygo.MenuItemSubmenu, nativeMenu(it.Children)
		case it.Checked:
			mi.Type, mi.Checked = mygo.MenuItemCheckbox, true
		}
		switch {
		case it.IconPNG != nil:
			mi.Icon, _ = mygo.NewMenuIcon(it.IconPNG, false)
		case it.Icon != nil:
			mi.Icon = svgMenuIcon(it.Icon, IsDark())
		}
		if h := it.OnClick; h != nil {
			mi.Click = func(*mygo.MenuItem, *mygo.Window) { Post(h) }
		}
		out = append(out, mi)
	}
	return out
}

// svgIcons holds the menu icons made of SVGs, by SVG and theme: an icon is
// drawn and decoded once, not each time a menu is set.
var svgIcons sync.Map // svgIcon -> *mygo.MenuIcon

type svgIcon struct {
	svg  *ui.SVG
	dark bool
}

// svgMenuIcon returns the menu icon of s in the color of the text of the
// light or the dark theme, nil if it cannot be made.
// shortcut: entries stay as long as the program, one per SVG and theme used in a menu bar; an app that makes SVGs on the fly in menus should drop it.
func svgMenuIcon(s *ui.SVG, dark bool) *mygo.MenuIcon {
	key := svgIcon{s, dark}
	if ic, ok := svgIcons.Load(key); ok {
		return ic.(*mygo.MenuIcon)
	}
	text := ui.LightTheme().Text
	if dark {
		text = ui.DarkTheme().Text
	}
	png, template := ui.MenuIconPNG(s, text)
	ic, err := mygo.NewMenuIcon(png, template)
	if err != nil {
		return nil
	}
	svgIcons.Store(key, ic)
	return ic
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
	b.change(func() { b.text = s })
	return b
}

// SetMenu sets the items of the menu; set it again when they change.
func (b *MenuButtonWidget) SetMenu(items ...MenuItem) *MenuButtonWidget {
	b.change(func() { b.items = items })
	return b
}

func (b *MenuButtonWidget) build(c *ui.Context) ui.Element {
	return ui.MenuButton(c, b.text, func(m *ui.Menu) { buildMenu(m, b.items) })
}

package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// MenuBarWidget is a row of menus that mygo draws in the window's content,
// as WinForms' MenuStrip docked in a form: put it where it goes, as under
// a HeaderBar, which the system's menu bar (Form.SetMenuBar, MenuStrip)
// cannot do.
type MenuBarWidget struct {
	common[MenuBarWidget]
	menus []MenuItem
	open  []bool // by menu: whether it shows
	// first is the menu to open on its first item in the next build, as
	// its access key or the arrows ask, -1 for none; hot is the title the
	// pointer was on in the last build, -1 for none.
	first, hot int
}

// barKey is the access key of a menu bar's title, which opens its menu.
type barKey struct {
	key  rune
	open func()
}

// MenuBar makes a menu bar of menus: each MenuItem is a title, as
// "&File", and its Children the items of its menu, which have what
// MenuItem has (submenus, separators, check marks, shortcuts, icons,
// Disabled, Destructive). A click on a title opens its menu; while one
// shows, the pointer on another title opens that one instead, and Left
// and Right go to the menu before and after, past those disabled. Alt and
// the letter after "&" open a menu on its first item, before the
// controls' access keys, and the letters are underlined while Alt is held,
// except on macOS, where access keys are off (see mnemonics). Escape
// closes the menu, and the focus goes back to where it was. Assistive
// technology sees a menu bar of menu items, and the menus as menus.
func MenuBar(menus ...MenuItem) *MenuBarWidget {
	b := &MenuBarWidget{menus: menus, first: -1, hot: -1}
	b.init(b)
	return b
}

// SetMenus sets the menus of the bar; set them again when their items
// change.
func (b *MenuBarWidget) SetMenus(menus ...MenuItem) *MenuBarWidget {
	b.change(func() { b.menus, b.open, b.first = menus, nil, -1 })
	return b
}

func (b *MenuBarWidget) fillsAcross() bool { return true }

// opens reports whether menu i opens: enabled, with an item that is not a
// separator.
func (b *MenuBarWidget) opens(i int) bool {
	return !b.menus[i].Disabled && slices.ContainsFunc(b.menus[i].Children, func(c MenuItem) bool { return c.Text != "" })
}

// next returns the menu that opens after i, by d (1 or -1), round the
// bar; i itself when no other opens.
func (b *MenuBarWidget) next(i, d int) int {
	n := len(b.menus)
	for k := 1; k < n; k++ {
		if j := ((i+d*k)%n + n) % n; b.opens(j) {
			return j
		}
	}
	return i
}

func (b *MenuBarWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	if len(b.open) != len(b.menus) {
		b.open = make([]bool, len(b.menus))
	}
	was := slices.Index(b.open, true)
	alt := !macMenus && buildingForm != nil && c.Modifiers()&ui.Alt != 0
	hot := -1
	bar := ui.Row(c).Role(ui.RoleMenuBar).Padding(t.Space(0.5), t.Space(1))
	bar.Children(func() {
		for i, it := range b.menus {
			label, key, at := parseMnemonic(it.Text)
			m := ui.MenuBase(c, &b.open[i])
			// The focus stays where it is, to go back to as the menu closes,
			// and Tab passes the titles by, as the system's bar.
			title := m.Trigger.Unfocusable().KeepFocus().Role(ui.RoleMenuItem).Label(label).
				Padding(t.Space(1), t.Space(2.5)).Radius(t.Radius).Disabled(!b.opens(i))
			if !b.opens(i) {
				title.TextColor(t.TextMuted)
			} else if b.open[i] || title.Hovered() {
				title.Background(t.SurfaceHover)
			}
			title.Children(func() {
				if alt && key != 0 {
					ui.RichText(c, mnemonicSpans(label, at)...)
				} else {
					ui.Text(c, label).SingleLine()
				}
			})
			if !b.opens(i) {
				continue
			}
			if title.Hovered() {
				hot = i
			}
			if key != 0 && !macMenus && buildingForm != nil {
				buildingForm.barKeys = append(buildingForm.barKeys, barKey{key, func() { b.first = i }})
			}
			if b.first == i {
				b.first = -1
				for j := range b.open {
					b.open[j] = false
				}
				m.OpenFirst()
			}
			m.PopupMenu(func(m *ui.Menu) { addItems(m, it.Children, true) })
			if d := m.Across(); d != 0 {
				if j := b.next(i, d); j != i {
					b.open[i], b.first = false, j
				}
			}
		}
	})
	if hot >= 0 && hot != b.hot && was >= 0 && was != hot && b.open[was] {
		// The pointer went onto another title while a menu shows: that
		// one's menu shows instead, in the next build.
		Post(func() {
			b.change(func() {
				if hot < len(b.open) && was < len(b.open) && b.open[was] { // as it was: SetMenus or Escape came first
					b.open[was], b.open[hot] = false, true
				}
			})
		})
	}
	b.hot = hot
	return bar
}

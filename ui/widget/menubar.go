package widget

import (
	"slices"

	"github.com/egoist/mygo/internal/keyhold"
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
	// mode is the title that has the keyboard focus while F10 (or Alt alone
	// on Windows) put the bar in its keyboard mode, -1 outside it; focus is
	// set while the title is to take the focus in the next build, inMenu
	// once a menu opened in the mode, back is the control that had the
	// focus before, to give it back, and form the form whose mode it is.
	// release, set once the mode ended, has the next build take the focus
	// off a title, which no longer takes it, for one frame ignoring input.
	mode                   int
	focus, inMenu, release bool
	back                   Control
	form                   *Form
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
// closes the menu, and the focus goes back to where it was. F10, and on
// Windows Alt alone, take the keyboard to the first bar of a form that has
// no bar of the system's, as the system's bar: its first title takes the
// focus, the arrows go along the bar, a letter or Down opens a menu, and
// Escape, F10 or Alt alone leave. Assistive technology sees a menu bar of
// menu items, and the menus as menus.
func MenuBar(menus ...MenuItem) *MenuBarWidget {
	b := &MenuBarWidget{menus: menus, first: -1, hot: -1, mode: -1}
	b.init(b)
	keyhold.Enable() // the keys typed after Alt alone or F10 reach the bar
	return b
}

// SetMenus sets the menus of the bar; set them again when their items
// change.
func (b *MenuBarWidget) SetMenus(menus ...MenuItem) *MenuBarWidget {
	b.change(func() {
		b.menus, b.open, b.first = menus, nil, -1
		b.leaveMode(true)
	})
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

// enterMode puts the bar in form w's keyboard mode, its first title that
// opens a menu taking the focus from back, the control that has it. Under
// mu.
func (b *MenuBarWidget) enterMode(w *Form, back Control) {
	for i := range b.menus {
		if b.opens(i) {
			b.mode, b.focus, b.inMenu, b.back, b.form = i, true, false, back, w
			w.modeBar = b
			return
		}
	}
}

// leaveMode ends the keyboard mode. The focus leaves the title, which no
// longer takes it: to the control that had it before when restore is set,
// else it stays where it went, or goes nowhere. Under mu.
func (b *MenuBarWidget) leaveMode(restore bool) {
	if b.mode < 0 {
		return
	}
	if restore && b.back != nil {
		focusLater(b.back)
	}
	Post(func() { b.change(func() { b.release = true }) })
	if b.form != nil && b.form.modeBar == b {
		b.form.modeBar = nil
	}
	b.mode, b.focus, b.inMenu, b.back, b.form = -1, false, false, nil, nil
}

// modeKeys answers the keys of the keyboard mode while title i has the
// focus: the arrows along the bar (the other way round right to left), a
// menu's letter, Down, Up, Enter and Space to open it, and Escape to
// leave; the form answers F10 and Alt alone (answerMnemonics).
func (b *MenuBarWidget) modeKeys(title ui.Element, i int, rtl bool) {
	back, ahead := ui.KeyLeft, ui.KeyRight
	if rtl {
		back, ahead = ahead, back
	}
	switch {
	case title.Shortcut(0, back):
		b.mode, b.focus = b.next(i, -1), true
	case title.Shortcut(0, ahead):
		b.mode, b.focus = b.next(i, 1), true
	case title.Shortcut(0, ui.KeyDown), title.Shortcut(0, ui.KeyUp), title.Shortcut(0, ui.KeyEnter), title.Shortcut(0, ui.KeySpace):
		b.first = i
	case title.Shortcut(0, ui.KeyEscape):
		b.leaveMode(true)
	}
	for j, it := range b.menus {
		if _, key, _ := parseMnemonic(it.Text); key != 0 && b.opens(j) {
			if k, ok := mnemonicKey(key); ok && title.Shortcut(0, k) {
				b.first = j
			}
		}
	}
}

func (b *MenuBarWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	if len(b.open) != len(b.menus) {
		b.open = make([]bool, len(b.menus))
	}
	if buildingForm == nil || macMenus {
		b.mode = -1 // no mode outside a form, nor on macOS, as access keys
	} else if buildingForm.menuBar == nil {
		buildingForm.menuBar = b // the form's F10 (answerMnemonics)
	}
	was := slices.Index(b.open, true)
	switch {
	case b.mode < 0:
	case was >= 0 || b.first >= 0:
		b.inMenu = true
	case b.inMenu:
		b.leaveMode(true) // the menu opened in the mode closed
	}
	alt := !macMenus && buildingForm != nil && (c.Modifiers()&ui.Alt != 0 || b.mode >= 0)
	rtl := c.Direction() == ui.RightToLeft
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
			} else if b.open[i] || title.Hovered() || b.mode == i {
				title.Background(t.SurfaceHover)
			}
			if b.release && b.mode < 0 && title.FocusWithin() {
				// The mode ended with the focus still here, nothing having
				// had it before: an element ignoring input loses it.
				title.IgnoreInput(true)
			}
			if b.mode == i && b.opens(i) {
				title.Focusable()
				if b.focus {
					b.focus = false
					title.Focus()
				}
				b.modeKeys(title, i, rtl)
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
	b.release = false
	return bar
}

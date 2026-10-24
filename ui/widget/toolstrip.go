package widget

import "github.com/egoist/mygo/ui"

// ToolStripWidget is a row of tools, for the top of a form (Form.ToolStrip):
// ToolButtons, Separators, Springs, ComboBoxes and DropDownButtons. It is
// ui.Toolbar underneath: its buttons take no face of their own until hovered,
// it is one stop of Tab with the arrow keys moving among its items, and what
// does not fit its width goes, from the last item, into a menu at its end,
// where choosing an item does what clicking it does: a ComboBox goes there
// under its chosen text, and a DropDownButton under its own, and choosing
// either opens its menu. ToolStrip adds no overflow of its own.
type ToolStripWidget struct {
	common[ToolStripWidget]
	items []Control
}

// ToolStrip makes a tool bar of items.
func ToolStrip(items ...Control) *ToolStripWidget {
	s := &ToolStripWidget{items: items}
	s.init(s)
	return s
}

// Add adds items after the others.
func (s *ToolStripWidget) Add(items ...Control) *ToolStripWidget {
	s.change(func() { s.items = append(s.items, items...) })
	return s
}

func (s *ToolStripWidget) build(c *ui.Context) ui.Element {
	return ui.Toolbar(c, func() {
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		for _, it := range s.items {
			render(c, it)
		}
	})
}

// ToolButtonWidget is a button of a tool strip: an icon, text, or both,
// and a tooltip (SetTooltip). SetToggle makes it a toggle, which keeps
// being checked or not.
type ToolButtonWidget struct {
	common[ToolButtonWidget]
	text     string
	icon     *ui.SVG
	toggle   bool
	on       mirror[bool]
	onClick  func()
	onChange func()
	// destructive is Destructive's, for the overflow menu.
	destructive bool
}

// ToolButton makes a tool button showing text; "" with SetIcon is an
// icon-only button, which SetTooltip names.
func ToolButton(text string) *ToolButtonWidget {
	b := &ToolButtonWidget{text: text}
	b.init(b)
	return b
}

// SetText changes the text.
func (b *ToolButtonWidget) SetText(s string) *ToolButtonWidget {
	b.change(func() { b.text = s })
	return b
}

// SetIcon shows an icon before the text, in the color of the text; nil
// removes it.
func (b *ToolButtonWidget) SetIcon(icon *ui.SVG) *ToolButtonWidget {
	b.change(func() { b.icon = icon })
	return b
}

// Suggested draws the button as the main action, as Button's Suggested
// does: the accent for its background, with the accent's text, and the
// accent's hover and pressed roles. It is the button's own style, which
// wins over applied ones. In the overflow menu it is a normal item: menus
// have no suggested role.
func (b *ToolButtonWidget) Suggested() *ToolButtonWidget {
	b.change(func() { b.destructive = false })
	return b.own(suggestedStyle)
}

// Destructive draws the button as an action that destroys, as Button's
// Destructive does: the danger color for its background, with the accent's
// text, fading to 0.9 opacity when hovered and 0.8 pressed. It is the
// button's own style. In the overflow menu it is a destructive item
// (MenuItem.Destructive), which only Windows 11 and the web draw apart.
func (b *ToolButtonWidget) Destructive() *ToolButtonWidget {
	b.change(func() { b.destructive = true })
	return b.own(destructiveStyle)
}

// SetToggle makes the button a toggle: a click checks it or unchecks it,
// and OnChange runs.
func (b *ToolButtonWidget) SetToggle(on bool) *ToolButtonWidget {
	b.change(func() { b.toggle = on })
	return b
}

// Checked reports whether a toggle is checked.
func (b *ToolButtonWidget) Checked() bool { mu.Lock(); defer mu.Unlock(); return b.on.value }

// SetChecked checks or unchecks a toggle. It does not run OnChange: the
// program knows what it set.
func (b *ToolButtonWidget) SetChecked(v bool) *ToolButtonWidget {
	b.change(func() { b.on.set(v) })
	return b
}

// OnClick runs fn on the event goroutine when the user clicks a button that
// is not a toggle.
func (b *ToolButtonWidget) OnClick(fn func()) *ToolButtonWidget {
	b.change(func() { b.onClick = fn })
	return b
}

// OnChange runs fn on the event goroutine after the user checks or
// unchecks a toggle.
func (b *ToolButtonWidget) OnChange(fn func()) *ToolButtonWidget {
	b.change(func() { b.onChange = fn })
	return b
}

func (b *ToolButtonWidget) build(c *ui.Context) ui.Element {
	label := b.text
	if b.icon != nil {
		label = ""
	}
	var e ui.Element
	if b.toggle {
		p, changed := b.on.bind()
		e = ui.Toggle(c, p, label).OnChange(onChange(&b.on, b.onChange))
		if changed && b.onChange != nil {
			Post(b.onChange)
		}
	} else {
		e = ui.Button(c, label)
		if h := b.onClick; h != nil {
			e.OnClick(func() { Post(h) })
		}
	}
	e.Destructive(b.destructive)
	if b.icon != nil {
		icon, text := b.icon, b.text
		name := text
		if name == "" {
			name = b.tooltip // an icon-only button is named by its tooltip
		}
		e.Label(name).Children(func() {
			ui.Row(c).Gap(c.Theme().Space(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icon)
				if text != "" {
					ui.Text(c, text)
				}
			})
		})
	}
	return e
}

// DropDownButton makes a tool strip button showing text and an arrow, which
// opens a menu of items.
func DropDownButton(text string, items ...MenuItem) *MenuButtonWidget {
	return MenuButton(text).SetMenu(items...)
}

// ToolStrip docks t at the top of the form, below its header bar and the
// window's menu bar, and above its content; nil removes it.
func (w *Form) ToolStrip(t *ToolStripWidget) *Form {
	w.change(func() { w.tools = t })
	return w
}

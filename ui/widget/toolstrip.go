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
	tk, scale := skinOf(c)
	bar := ui.Toolbar(c, func() {
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		for _, it := range s.items {
			e := render(c, it)
			if _, sep := it.(*SeparatorWidget); sep && tk != nil {
				// Across a separator, ToolbarGap + ToolbarGroupGap from control to
				// control, its line of 1 in the middle.
				e.MarginX(((tk.ToolbarGroupGap-tk.ToolbarGap)*scale - 1) / 2)
			}
		}
	})
	if tk != nil {
		t := c.Theme()
		line := tk.LineStrong
		if c.Preferences().HighContrast {
			line = line.Mix(tk.Text, 0.45)
		}
		bar.Background(tk.Chrome).Padding(tk.ToolbarPadTop*scale, t.Space(1), tk.ToolbarPadBottom*scale, t.Space(1)).
			Gap(tk.ToolbarGap*scale).BorderWidth(0, 0, 1, 0).BorderColor(line)
	}
	return bar
}

// ToolButtonWidget is a button of a tool strip: an icon, text, or both,
// and a tooltip (SetTooltip). SetToggle makes it a toggle, which keeps
// being checked or not.
type ToolButtonWidget struct {
	common[ToolButtonWidget]
	text     string
	icon     *ui.SVG
	png      *ui.Bitmap // SetIconPNG's, drawn in its own colors
	look     lookKind
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
// removes it. It replaces a PNG icon.
func (b *ToolButtonWidget) SetIcon(icon *ui.SVG) *ToolButtonWidget {
	b.change(func() { b.icon, b.png = icon, nil })
	return b
}

// SetIconPNG shows the image in data before the text, in its own colors, as
// Button's SetIconPNG does; nil, or bytes that are no image, remove it (the
// tooltip still names a button with no text). It replaces an SVG icon.
func (b *ToolButtonWidget) SetIconPNG(data []byte) *ToolButtonWidget {
	bmp := decodeImage(data) // outside mu: decoding takes a while
	b.change(func() { b.png, b.icon, bitmapIcon = bmp, nil, drawBitmapIcon })
	return b
}

// Suggested draws the button as the main action, as Button's Suggested
// does: the accent for its background, with the accent's text, and the
// accent's hover and pressed roles. It is the button's own style, which
// wins over applied ones. A tool button draws no border of its own, so
// there is no ring to hide. In the overflow menu it is a normal item: menus
// have no suggested role. It replaces a Destructive given before.
func (b *ToolButtonWidget) Suggested() *ToolButtonWidget {
	b.change(func() { b.destructive = false })
	return b.own(b.look.to(suggestedLook, suggestedStyle))
}

// Destructive draws the button as an action that destroys, as Button's
// Destructive does: the danger color for its background (darkened where the
// accent's text would be under 4.5:1 on it), with the accent's text, darker
// by 8% when hovered and 16% pressed. It is the button's own style. In the
// overflow menu it is a destructive item (MenuItem.Destructive), which only
// Windows 11 and the web draw apart. It replaces a Suggested given before.
func (b *ToolButtonWidget) Destructive() *ToolButtonWidget {
	b.change(func() { b.destructive = true })
	return b.own(b.look.to(destructiveLook, destructiveStyle))
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
	if b.icon != nil || b.png != nil {
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
	if b.icon != nil || b.png != nil {
		side := float32(0)
		if tk, scale := skinOf(c); tk != nil {
			side = tk.ToolbarIcon * scale
		}
		iconFace(c, e, b.icon, b.png, b.text, b.tooltip, true, side)
	} else if b.text == "" && b.tooltip != "" {
		e.Label(b.tooltip) // a button with no text is named by its tooltip
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

func (s *ToolStripWidget) children() []Control { return s.items }

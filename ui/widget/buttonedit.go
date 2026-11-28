package widget

import (
	"slices"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// EditButtonKind is what a button of a ButtonEdit shows.
type EditButtonKind int

const (
	EditEllipsis EditButtonKind = iota // …, to browse or open a dialog
	EditClear                          // ×: clears the text; shows while there is any
	EditSearch                         // a magnifying glass
	EditPlus                           // +
	EditMinus                          // −
	EditDropDown                       // a chevron down
	EditGlyph                          // the button's Image
)

// EditButton is a button inside a ButtonEdit's frame. Image is drawn only
// by an EditGlyph. Tooltip shows while the pointer rests on it, and names
// it for assistive technology. OnClick is posted to the event goroutine,
// as Button's OnClick is.
type EditButton struct {
	Kind    EditButtonKind
	Image   *ui.Bitmap
	Tooltip string
	OnClick func()
}

// ButtonEditWidget is a one-line text field with buttons at its end,
// inside its frame (DevExpress's ButtonEdit). The buttons are no stops of
// Tab and leave the keyboard in the text, as DevExpress's do; Ctrl+Enter
// clicks the first Ellipsis or DropDown button shown.
type ButtonEditWidget struct {
	common[ButtonEditWidget]
	text        mirror[string]
	placeholder string
	readOnly    bool
	onChange    func()
	buttons     []EditButton
	hidden      []bool // SetButtonVisible's
	// Made once, so a frame at rest allocates nothing: the build's children,
	// ui's OnChange callback, each button's click and face. icons are the
	// buttons' SVGs, parsed in the first build.
	edit     func()
	children func()
	clicks   []func()
	faces    []func()
	icons    []*ui.SVG
	c        *ui.Context // the build's, under mu while building
}

// ButtonEdit makes an empty field with buttons at its end, in order.
func ButtonEdit(buttons ...EditButton) *ButtonEditWidget {
	b := &ButtonEditWidget{buttons: slices.Clone(buttons), hidden: make([]bool, len(buttons))}
	b.clicks, b.faces = make([]func(), len(buttons)), make([]func(), len(buttons))
	for i := range b.buttons {
		b.clicks[i] = func() { mu.Lock(); b.press(i); mu.Unlock() }
		b.faces[i] = func() { b.face(i) }
	}
	b.children = b.buildChildren
	b.init(b)
	b.lookAs(skinhook.KindField) // typed into, as a TextBox
	return b
}

// Text returns the text.
func (b *ButtonEditWidget) Text() string { mu.Lock(); defer mu.Unlock(); return b.text.value }

// SetText sets the text, without running OnChange.
func (b *ButtonEditWidget) SetText(s string) *ButtonEditWidget {
	b.change(func() { b.text.set(s) })
	return b
}

// SetPlaceholder sets the text shown while the field is empty.
func (b *ButtonEditWidget) SetPlaceholder(s string) *ButtonEditWidget {
	b.change(func() { b.placeholder = s })
	return b
}

// SetReadOnly keeps the user from typing in the field, or lets them again.
// The buttons still work, Clear included.
func (b *ButtonEditWidget) SetReadOnly(on bool) *ButtonEditWidget {
	b.change(func() { b.readOnly = on })
	return b
}

// OnChange runs fn, on the event goroutine, after each edit of the user,
// Clear included. A setter does not run it.
func (b *ButtonEditWidget) OnChange(fn func()) *ButtonEditWidget {
	b.change(func() { b.onChange = fn })
	return b
}

// SetButtonVisible shows or hides the button of index i; a hidden one
// takes no room. An index out of range is ignored.
func (b *ButtonEditWidget) SetButtonVisible(i int, on bool) *ButtonEditWidget {
	b.change(func() {
		if i >= 0 && i < len(b.hidden) {
			b.hidden[i] = !on
		}
	})
	return b
}

// editShapes are the Lucide icons of the kinds (ISC license); DropDown's
// is chevronDown, Glyph has none.
var editShapes = [...]string{
	EditEllipsis: `<circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>`,
	EditClear:    `<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`,
	EditSearch:   `<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>`,
	EditPlus:     `<path d="M5 12h14"/><path d="M12 5v14"/>`,
	EditMinus:    `<path d="M5 12h14"/>`,
	EditDropDown: "",
}

// editIcons are the kinds' icons, parsed when a ButtonEdit first shows
// one: parsed at the start of the program, they would cost every app.
// Under mu.
var editIcons [len(editShapes)]*ui.SVG

// editIcon returns the icon of kind k, nil for none. Under mu.
func editIcon(k EditButtonKind) *ui.SVG {
	switch {
	case k == EditDropDown:
		return chevronDown
	case k < 0 || int(k) >= len(editShapes):
		return nil
	case editIcons[k] == nil:
		editIcons[k] = lucide(editShapes[k])
	}
	return editIcons[k]
}

// name is what names button i: its tooltip, or its kind's name.
func (b *ButtonEditWidget) name(i int) string {
	k := &b.buttons[i]
	if k.Tooltip != "" {
		return k.Tooltip
	}
	switch k.Kind {
	case EditEllipsis:
		return l10n.More()
	case EditClear:
		return l10n.Clear()
	case EditSearch:
		return l10n.Search()
	case EditDropDown:
		return l10n.Open()
	case EditPlus:
		return "+"
	case EditMinus:
		return "−"
	}
	return ""
}

// press clicks button i: Clear empties the text, as an edit of the user,
// then its OnClick runs. Under mu.
func (b *ButtonEditWidget) press(i int) {
	k := b.buttons[i]
	if k.Kind != EditClear || b.text.value == "" {
		if k.OnClick != nil {
			Post(k.OnClick)
		}
		return
	}
	b.text.set("")
	h, write, click := b.onChange, b.bound.write, k.OnClick
	Post(func() {
		if h != nil {
			h()
		}
		if write != nil {
			write("")
		}
		if click != nil {
			click()
		}
	})
}

// dropButton is the index of the first Ellipsis or DropDown button shown,
// -1 for none. Under mu.
func (b *ButtonEditWidget) dropButton() int {
	for i, k := range b.buttons {
		if !b.hidden[i] && (k.Kind == EditEllipsis || k.Kind == EditDropDown) {
			return i
		}
	}
	return -1
}

func (b *ButtonEditWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	if b.icons == nil {
		b.icons = make([]*ui.SVG, len(b.buttons))
		for i, k := range b.buttons {
			b.icons[i] = editIcon(k.Kind)
		}
	}
	b.c = c
	frame := editFrame(c, b.children)
	b.c = nil
	b.edgeEditFrame(t, frame)
	return frame
}

// editFrame is the frame of a field with buttons inside it, ButtonEdit's
// and TimeSpanEdit's, as high as a TextBox; children builds the text and
// the buttons.
func editFrame(c *ui.Context, children func()) ui.Element {
	t := c.Theme()
	frame := ui.Row(c).Radius(t.Radius).Border(1, t.Outline()).Background(t.FieldFill()).
		AlignItems(ui.Stretch).MinWidth(t.Space(35))
	if h := t.Metrics.ControlHeight; h > 0 {
		frame.MinHeight(h)
	}
	frame.Children(children)
	return frame
}

// edgeEditFrame gives an editFrame the edge of a text input's, the accent
// while the text (inner) has the focus, and keeps the buttons off the error
// mark. Under mu, building.
func (b *core) edgeEditFrame(t *ui.Theme, frame ui.Element) {
	switch {
	case b.inner.Focused():
		frame.Border(1, t.Accent)
	case frame.Hovered():
		frame.Border(1, t.Outline().Mix(t.Text, 0.25))
	}
	if b.errMsg != "" {
		b.post = errorMarkRoom // the buttons stop before the mark
	}
}

// editButtonBase is a button inside an editFrame: no stop of Tab, and a
// click leaves the focus in the text.
func editButtonBase(c *ui.Context) ui.Element {
	t := c.Theme()
	e := ui.ButtonBase(c).Unfocusable().KeepFocus().Role(ui.RoleButton).
		Width(t.Space(6)).Shrink(0).Margin(t.Space(0.5)).Radius(t.Radius).
		AlignItems(ui.Center).Justify(ui.Center)
	switch {
	case e.Pressed():
		e.Background(t.SurfacePressed)
	case e.Hovered():
		e.Background(t.SurfaceHover)
	}
	return e
}

// errorMarkRoom pads a field's frame at its end for an ErrorProvider's
// mark, which sits there, so that what the field shows at its end stops
// before it.
func errorMarkRoom(c *ui.Context, el ui.Element) {
	side := 1
	if c.Direction() == ui.RightToLeft {
		side = 3
	}
	el.PaddingEnd(el.Paddings()[side] + errorMarkSize + errorMarkInset + 2)
}

// buildChildren builds the text and the buttons. Under mu, building.
func (b *ButtonEditWidget) buildChildren() {
	c, t := b.c, b.c.Theme()
	p, changed := b.text.bind()
	in := ui.TextInputBase(c, p).Grow(1).MinWidth(0).Padding(t.Space(1.5), t.Space(2.5))
	b.inner = in
	if b.placeholder != "" {
		in.Placeholder(b.placeholder)
		if tk, _ := skinOf(c); tk != nil {
			in.PlaceholderColor(tk.TextPlaceholder)
		}
	}
	in.ReadOnly(b.readOnly)
	in.OnChange(editedOnce(&b.edit, &b.text, &b.bound, &b.onChange))
	if changed && b.onChange != nil {
		Post(b.onChange)
	}
	if h := formAccept; h != nil {
		in.OnSubmit(h) // Enter in the line clicks the form's accept button
	}
	if i := b.dropButton(); i >= 0 && in.Shortcut(ui.Ctrl, ui.KeyEnter) {
		b.press(i)
	}
	for i := range b.buttons {
		if b.hidden[i] || b.buttons[i].Kind == EditClear && *p == "" {
			continue
		}
		e := editButtonBase(c.Key(editKey(i)))
		e.Label(b.name(i))
		if tip := b.buttons[i].Tooltip; tip != "" {
			e.Tooltip(tip)
		}
		e.OnClick(b.clicks[i]).Children(b.faces[i])
	}
}

// editKey keys a ButtonEdit's button by its index, so that it keeps its
// state while others hide.
type editKey int

// face draws button i's icon, or its Image. Under mu, building.
func (b *ButtonEditWidget) face(i int) {
	c, k := b.c, &b.buttons[i]
	if k.Kind == EditGlyph {
		side := iconSide(c) // a nil Image draws nothing
		ui.Image(c, k.Image).Width(side).Height(side)
		return
	}
	if svg := b.icons[i]; svg != nil {
		t := c.Theme()
		ui.Icon(c, svg).FontSize(t.Space(3.5)).TextColor(t.TextMuted)
	}
}

func (b *ButtonEditWidget) bindValue() string  { return b.Text() }
func (b *ButtonEditWidget) bindApply(v string) { b.SetText(v) }

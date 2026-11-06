package widget

import (
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// Kind is what a message dialog is about, shown by its icon.
type Kind uint8

const (
	Info Kind = iota
	Warning
	Question
	Error
)

// kindIcons are the message dialogs' icons: filled, in their own colors,
// the same on every platform and theme, drawn in a 32 by 32 box.
var kindIcons = [...]*ui.SVG{
	Info:     filledIcon(`<circle cx="16" cy="16" r="15" fill="#1976d2"/><circle cx="16" cy="9.5" r="2.2" fill="#fff"/><path d="M16 14.5v9" stroke="#fff" stroke-width="3.4" stroke-linecap="round" fill="none"/>`),
	Warning:  filledIcon(`<path d="M16 3 30 28H2z" fill="#fbc02d" stroke="#fbc02d" stroke-width="3" stroke-linejoin="round"/><path d="M16 11.5v8" stroke="#000" stroke-width="3" stroke-linecap="round" fill="none"/><circle cx="16" cy="24" r="1.9"/>`),
	Question: filledIcon(`<circle cx="16" cy="16" r="15" fill="#1976d2"/><path d="M11.5 12.5a4.5 4.5 0 1 1 6.5 4c-1.6.9-2 1.9-2 3.3" stroke="#fff" stroke-width="3" stroke-linecap="round" fill="none"/><circle cx="16" cy="24.5" r="2" fill="#fff"/>`),
	Error:    filledIcon(`<circle cx="16" cy="16" r="15" fill="#d32f2f"/><path d="m10.5 10.5 11 11m0-11-11 11" stroke="#fff" stroke-width="3.4" stroke-linecap="round" fill="none"/>`),
}

func filledIcon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">` + shapes + `</svg>`))
}

// lucide wraps the shapes of a Lucide icon in its svg element.
func lucide(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// IconWidget shows an SVG icon in the color of the text around it.
type IconWidget struct {
	common[IconWidget]
	svg  *ui.SVG
	size float32
}

// Icon makes an icon of svg, size DIPs high; 0 is the text's size. Color
// it with TextColor. There are no named icons: embed the SVGs you need.
func Icon(svg *ui.SVG, size float32) *IconWidget {
	i := &IconWidget{svg: svg, size: size}
	i.init(i)
	return i
}

func (i *IconWidget) build(c *ui.Context) ui.Element {
	e := ui.Icon(c, i.svg)
	if i.size > 0 {
		e.FontSize(i.size)
	}
	return e
}

// kindIconWidget is a message dialog's icon, in its kind's own colors.
type kindIconWidget struct {
	common[kindIconWidget]
	kind Kind
}

func kindIcon(k Kind) *kindIconWidget {
	i := &kindIconWidget{kind: k}
	i.init(i)
	return i
}

func (i *kindIconWidget) build(c *ui.Context) ui.Element {
	return ui.Image(c, kindIcons[i.kind]).Size(32, 32)
}

// message runs a dialog over parent showing text, input under it if any,
// and buttons at the bottom, the last the default. It returns the index of
// the button chosen, -1 when the dialog closed otherwise. wire, if any,
// gets the default button and a function choosing it.
func message(parent *Form, kind Kind, title, text string, input Control, wire func(ok *ButtonWidget, choose func()), buttons ...string) int {
	d, chosen := messageDialog(parent, kind, title, text, input, wire, buttons...)
	d.Run()
	return chosen()
}

// messageDialog makes message's dialog, and the function that tells which
// button was chosen once it closed.
func messageDialog(parent *Form, kind Kind, title, text string, input Control, wire func(ok *ButtonWidget, choose func()), buttons ...string) (*Dialog, func() int) {
	chosen := -1
	// The first answer wins: a second, queued while the first ran, must
	// not change what Run's caller reads.
	answer := func(i int) {
		if chosen < 0 {
			chosen = i
		}
	}
	var d *Dialog
	bar := ButtonBox()
	var ok *ButtonWidget
	for i, s := range buttons {
		ok = Button(s).OnClick(func() { answer(i); d.Close() })
		bar.Add(ok)
	}
	ok.Suggested()
	// The text keeps the icon's height at least, centred in it: one line sits
	// beside the icon's middle, more lines leave the icon at the top.
	body := VBox(8).Apply(Style{}.MinHeight(32))
	if text != "" {
		body.Pack(Label(text).Align(Start), input == nil, false)
	}
	if input != nil {
		body.Add(input)
	} else {
		ok.Focus() // Enter or Space answers with the default
	}
	content := VBox(12).
		Pack(HBox(12).Add(kindIcon(kind).VAlign(Start)).Pack(body, true, true), true, true).
		Add(bar)
	d = NewDialog(parent, title, "", 420, 0, content)
	d.title.SetVisible(false) // the window's caption shows it
	d.change(func() { d.fixedFrame = true })
	d.SetResizable(false)
	if wire != nil {
		last := len(buttons) - 1
		wire(ok, func() { answer(last); d.Close() })
	}
	return d, func() int { return chosen }
}

// Alert shows text in a dialog over parent (nil for none) and returns when
// the user dismisses it. It blocks as Dialog.Run does.
func Alert(parent *Form, kind Kind, title, text string) {
	message(parent, kind, title, text, nil, nil, l10n.Ok())
}

// Confirm asks a question in a dialog over parent and reports whether the
// user chose Yes; No, which is the default, and closing the dialog say no.
func Confirm(parent *Form, kind Kind, title, text string) bool {
	return message(parent, kind, title, text, nil, nil, l10n.Yes(), l10n.No()) == 0
}

// InputString asks for a line of text, label above an entry holding value,
// and returns it with ok true for OK or Enter.
func InputString(parent *Form, kind Kind, title, label, value string) (s string, ok bool) {
	e := TextBox().SetText(value).Focus()
	n := message(parent, kind, title, label, e, func(_ *ButtonWidget, choose func()) { e.OnEnter(choose) }, l10n.Cancel(), l10n.Ok())
	return e.Text(), n == 1
}

// InputInt asks for a whole number from min to max, stepping by step,
// starting at value.
func InputInt(parent *Form, kind Kind, title, label string, min, max, step, value int) (n int, ok bool) {
	s := NumericUpDown(float64(min), float64(max), float64(step)).SetValue(float64(value)).Focus()
	i := message(parent, kind, title, label, s, func(_ *ButtonWidget, choose func()) { s.OnEnter(choose) }, l10n.Cancel(), l10n.Ok())
	return s.Int(), i == 1
}

// InputSelect asks to choose one of opts and returns its Value. OK stays
// disabled until a choice is made.
func InputSelect(parent *Form, kind Kind, title, label string, opts ...Option) (value string, ok bool) {
	c := ComboBox(opts...)
	n := message(parent, kind, title, label, c, func(okButton *ButtonWidget, _ func()) {
		okButton.SetDisabled(true)
		c.OnChange(func() { okButton.SetDisabled(c.Value() == "") })
	}, l10n.Cancel(), l10n.Ok())
	return c.Value(), n == 1
}

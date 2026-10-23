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

// kindIcons are Lucide's info, triangle-alert, circle-help and circle-x
// (ISC license).
var kindIcons = [...]*ui.SVG{
	Info:     lucide(`<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>`),
	Warning:  lucide(`<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>`),
	Question: lucide(`<circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><path d="M12 17h.01"/>`),
	Error:    lucide(`<circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/>`),
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

// kindIconWidget is a message dialog's icon, in its kind's theme color.
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
	t := c.Theme()
	col := t.Accent
	switch i.kind {
	case Warning:
		col = t.Warning
	case Error:
		col = t.Danger
	}
	return ui.Icon(c, kindIcons[i.kind]).FontSize(32).TextColor(col)
}

// message runs a dialog over parent showing text, input under it if any,
// and buttons at the bottom, the last the default. It returns the index of
// the button chosen, -1 when the dialog closed otherwise. wire, if any,
// gets the default button and a function choosing it.
func message(parent *Window, kind Kind, title, text string, input Control, wire func(ok *ButtonWidget, choose func()), buttons ...string) int {
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
	body := VBox(8)
	if text != "" {
		body.Add(Label(text))
	}
	height := 200
	if input != nil {
		body.Add(input)
		height = 240
	} else {
		ok.Focus() // Enter or Space answers with the default
	}
	content := VBox(12).
		Pack(HBox(12).Add(kindIcon(kind)).Pack(body, true, true), true, true).
		Add(bar)
	d = NewDialog(parent, title, "", 420, height, content)
	if wire != nil {
		last := len(buttons) - 1
		wire(ok, func() { answer(last); d.Close() })
	}
	d.Run()
	return chosen
}

// Alert shows text in a dialog over parent (nil for none) and returns when
// the user dismisses it. It blocks as Dialog.Run does.
func Alert(parent *Window, kind Kind, title, text string) {
	message(parent, kind, title, text, nil, nil, l10n.Ok())
}

// Confirm asks a question in a dialog over parent and reports whether the
// user chose OK; Cancel and closing the dialog say no.
func Confirm(parent *Window, kind Kind, title, text string) bool {
	return message(parent, kind, title, text, nil, nil, l10n.Cancel(), l10n.Ok()) == 1
}

// InputString asks for a line of text, label above an entry holding value,
// and returns it with ok true for OK or Enter.
func InputString(parent *Window, kind Kind, title, label, value string) (s string, ok bool) {
	e := Entry().SetText(value).Focus()
	n := message(parent, kind, title, label, e, func(_ *ButtonWidget, choose func()) { e.OnEnter(choose) }, l10n.Cancel(), l10n.Ok())
	return e.Text(), n == 1
}

// InputInt asks for a whole number from min to max, stepping by step,
// starting at value.
func InputInt(parent *Window, kind Kind, title, label string, min, max, step, value int) (n int, ok bool) {
	s := Spin(float64(min), float64(max), float64(step)).SetValue(float64(value)).Focus()
	i := message(parent, kind, title, label, s, func(_ *ButtonWidget, choose func()) { s.OnEnter(choose) }, l10n.Cancel(), l10n.Ok())
	return s.Int(), i == 1
}

// InputSelect asks to choose one of opts and returns its Value. OK stays
// disabled until a choice is made.
func InputSelect(parent *Window, kind Kind, title, label string, opts ...Option) (value string, ok bool) {
	c := Combo(opts...)
	n := message(parent, kind, title, label, c, func(okButton *ButtonWidget, _ func()) {
		okButton.SetDisabled(true)
		c.OnChange(func() { okButton.SetDisabled(c.Value() == "") })
	}, l10n.Cancel(), l10n.Ok())
	return c.Value(), n == 1
}

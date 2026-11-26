package widget

import "github.com/egoist/mygo/ui"

// CheckButtonWidget is a button that stays pressed while it is checked, as
// Bold in an editor's toolbar: a click presses it or lets it go, as does
// Space while it has the keyboard focus. It is ui.Toggle underneath, as
// DevExpress's CheckButton.
type CheckButtonWidget struct {
	edit func() // editedOnce's
	common[CheckButtonWidget]
	text     string
	icon     *ui.SVG
	on       mirror[bool]
	onChange func()
}

// CheckButton makes a check button showing text, not checked.
func CheckButton(text string) *CheckButtonWidget {
	b := &CheckButtonWidget{text: text}
	b.init(b)
	return b
}

// Checked reports whether the button is pressed.
func (b *CheckButtonWidget) Checked() bool { mu.Lock(); defer mu.Unlock(); return b.on.value }

// SetChecked presses the button or lets it go, without running OnChange.
func (b *CheckButtonWidget) SetChecked(on bool) *CheckButtonWidget {
	b.change(func() { b.on.set(on) })
	return b
}

// SetIcon shows an icon before the text, in the color of the text, as
// ButtonWidget's; nil removes it. A button with no text is named by its
// tooltip.
func (b *CheckButtonWidget) SetIcon(s *ui.SVG) *CheckButtonWidget {
	b.change(func() { b.icon = s })
	return b
}

// OnChange runs fn, on the event goroutine, after the user pressed the
// button or let it go. A setter does not run it.
func (b *CheckButtonWidget) OnChange(fn func()) *CheckButtonWidget {
	b.change(func() { b.onChange = fn })
	return b
}

func (b *CheckButtonWidget) build(c *ui.Context) ui.Element {
	p, changed := b.on.bind()
	if changed && b.onChange != nil {
		Post(b.onChange)
	}
	text := b.text
	if b.icon != nil {
		text = ""
	}
	e := ui.Toggle(c, p, text).OnChange(editedOnce(&b.edit, &b.on, &b.bound, &b.onChange))
	if b.icon != nil {
		iconFace(c, e, b.icon, nil, b.text, b.tooltip, false, 0)
	}
	skinChecked(c, e, *p)
	b.selected = *p
	return e
}

func (b *CheckButtonWidget) bindValue() bool  { return b.Checked() }
func (b *CheckButtonWidget) bindApply(v bool) { b.SetChecked(v) }

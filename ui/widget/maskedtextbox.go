package widget

import (
	"github.com/egoist/mygo/ui"
)

// MaskedTextBoxWidget is a one-line text input that takes only what a mask
// allows, as WinForms' MaskedTextBox: "000-0000" is seven digits with a
// dash, shown as the template "___-____" the user fills. See
// docs/ui/widget/maskedtextbox.md for the mask characters.
type MaskedTextBoxWidget struct {
	common[MaskedTextBoxWidget]
	text       mirror[string] // the template as shown
	spec       string
	prompt     rune
	mk         *mask
	onChange   func()
	onRejected func(r rune)
}

// MaskedTextBox makes a masked input empty, with the prompt '_'.
func MaskedTextBox(spec string) *MaskedTextBoxWidget {
	m := &MaskedTextBoxWidget{spec: spec, prompt: '_'}
	m.mk = newMask(spec, m.prompt)
	m.text.set(m.mk.display())
	m.init(m)
	return m
}

// SetMask changes the mask and empties the box.
func (m *MaskedTextBoxWidget) SetMask(spec string) *MaskedTextBoxWidget {
	m.change(func() {
		m.spec = spec
		m.mk = newMask(spec, m.prompt)
		m.text.set(m.mk.display())
	})
	return m
}

// Mask returns the mask.
func (m *MaskedTextBoxWidget) Mask() string { mu.Lock(); defer mu.Unlock(); return m.spec }

// SetPromptChar changes the character that stands for a slot not filled
// (the default is '_'). It must not be a character the mask takes.
func (m *MaskedTextBoxWidget) SetPromptChar(r rune) *MaskedTextBoxWidget {
	m.change(func() {
		m.text.set(m.mk.withPrompt(m.text.value, r))
		m.prompt = r
	})
	return m
}

// Text returns what the user typed, without the mask's literals and prompts:
// "1234567" for "123-4567".
func (m *MaskedTextBoxWidget) Text() string {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.value(m.text.value)
}

// TextWithLiterals returns the text with the literals, and a space for each
// slot not filled: "123-45  ".
func (m *MaskedTextBoxWidget) TextWithLiterals() string {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.literals(m.text.value)
}

// SetText fills the mask with the characters of s, one in each slot from the
// first, as Text returns them. It reports whether each one fitted its slot;
// if one did not, or there are more than slots, nothing changes. It does
// not run OnChange.
func (m *MaskedTextBoxWidget) SetText(s string) bool {
	ok := false
	m.change(func() {
		cp := *m.mk // the edits of the user use their own copy
		var text string
		if text, ok = cp.fill(s); ok {
			m.text.set(text)
		}
	})
	return ok
}

// MaskCompleted reports whether every required slot is filled, MaskFull
// whether every slot is.
func (m *MaskedTextBoxWidget) MaskCompleted() bool {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.completed(m.text.value)
}

func (m *MaskedTextBoxWidget) MaskFull() bool {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.full(m.text.value)
}

// OnChange runs fn, on the event goroutine, after the user changed the text.
func (m *MaskedTextBoxWidget) OnChange(fn func()) *MaskedTextBoxWidget {
	m.change(func() { m.onChange = fn })
	return m
}

// OnMaskInputRejected runs fn, on the event goroutine, for each character the
// user typed or pasted that no slot took.
func (m *MaskedTextBoxWidget) OnMaskInputRejected(fn func(r rune)) *MaskedTextBoxWidget {
	m.change(func() { m.onRejected = fn })
	return m
}

func (m *MaskedTextBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := m.text.bind()
	if changed && m.onChange != nil {
		Post(m.onChange)
	}
	cp := *m.mk // the user's edits run off the lock on this copy
	rejected := m.onRejected
	el := ui.TextInput(c, p).OnEdit(func(e ui.TextEdit) (string, int, bool) {
		text, caret, ok := cp.edit(e.Text, e.Start, e.End, e.Insert, e.Caret)
		if rejected != nil {
			for _, r := range cp.rejected {
				r := r
				Post(func() { rejected(r) })
			}
		}
		return text, caret, ok
	})
	return el.OnChange(edited(&m.text, &m.bound, m.onChange))
}

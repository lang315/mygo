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
	cells      []rune // what each slot holds, which the text shown cannot tell from a prompt
	onChange   func()
	onRejected func(r rune)
	edit       func() // editedOnce's
	onEditFn   func(e ui.TextEdit) (string, int, bool)
	limitFn    func(text string) int
}

// MaskedTextBox makes a masked input empty, with the prompt '_'.
func MaskedTextBox(spec string) *MaskedTextBoxWidget {
	m := &MaskedTextBoxWidget{spec: spec, prompt: '_'}
	m.mk = newMask(spec, m.prompt)
	m.cells = make([]rune, len(m.mk.slots))
	m.text.set(m.mk.display())
	m.init(m)
	return m
}

// SetMask changes the mask and puts the text in it, as Text returns it, in the
// new one: what its slots do not take is dropped. It does not run OnChange.
func (m *MaskedTextBoxWidget) SetMask(spec string) *MaskedTextBoxWidget {
	m.change(func() {
		chars := m.mk.valueOf(m.current(m.text.value))
		m.spec = spec
		m.mk = newMask(spec, m.prompt)
		m.cells = m.mk.refill(chars)
		m.text.set(m.mk.render(m.cells))
	})
	return m
}

// Mask returns the mask.
func (m *MaskedTextBoxWidget) Mask() string { mu.Lock(); defer mu.Unlock(); return m.spec }

// SetPromptChar changes the character that stands for a slot not filled
// (the default is '_'). A slot may still hold that character: the box tells
// it from an empty one.
func (m *MaskedTextBoxWidget) SetPromptChar(r rune) *MaskedTextBoxWidget {
	m.change(func() {
		cells := m.current(m.text.value)
		m.prompt = r
		m.mk = newMask(m.spec, r)
		m.cells = cells
		m.text.set(m.mk.render(cells))
	})
	return m
}

// current returns the cells that go with shown, the text on screen: the ones
// the last edit left, unless an undo or a redo changed the text since, which
// are read back from it (the prompt then stands for an empty slot). Under mu.
func (m *MaskedTextBoxWidget) current(shown string) []rune {
	if m.mk.render(m.cells) == shown {
		return m.cells
	}
	return m.mk.cells(shown)
}

// Text returns what the user typed, without the mask's literals and prompts:
// "1234567" for "123-4567".
func (m *MaskedTextBoxWidget) Text() string {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.valueOf(m.current(m.text.value))
}

// TextWithLiterals returns the text with the literals, and a space for each
// slot not filled: "123-45  ".
func (m *MaskedTextBoxWidget) TextWithLiterals() string {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.blanks(m.current(m.text.value))
}

// SetText fills the mask with the characters of s, one in each slot from the
// first, as Text returns them, or with the text as TextWithLiterals returns
// it, literals included. It reports whether each one fitted its slot; if one
// did not, or there are more than slots, nothing changes. It does not run
// OnChange.
func (m *MaskedTextBoxWidget) SetText(s string) bool {
	ok := false
	m.change(func() {
		var cells []rune
		if cells, ok = m.mk.fill(s); ok {
			m.cells = cells
			m.text.set(m.mk.render(cells))
		}
	})
	return ok
}

// MaskCompleted reports whether every required slot is filled, MaskFull
// whether every slot is.
func (m *MaskedTextBoxWidget) MaskCompleted() bool {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.completedOf(m.current(m.text.value))
}

func (m *MaskedTextBoxWidget) MaskFull() bool {
	mu.Lock()
	defer mu.Unlock()
	return m.mk.fullOf(m.current(m.text.value))
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

// editText is ui's OnEdit callback: the edit the mask makes of the user's.
// Off the lock when building: the mask is the one of the last frame, copied
// so that the rejected runes are this edit's.
func (m *MaskedTextBoxWidget) editText(e ui.TextEdit) (string, int, bool) {
	mu.Lock()
	cp := *m.mk
	cells := m.cells
	rejected := m.onRejected
	if cp.render(cells) != e.Text {
		cells = cp.cells(e.Text)
	}
	mu.Unlock()
	out, caret := cp.editCells(cells, e.Start, e.End, e.Insert, e.Caret)
	mu.Lock()
	m.cells = out
	mu.Unlock()
	if rejected != nil {
		for _, r := range cp.rejected {
			r := r
			Post(func() { rejected(r) })
		}
	}
	return cp.render(out), caret, true
}

// caretLimit is ui's CaretLimit callback.
func (m *MaskedTextBoxWidget) caretLimit(text string) int {
	mu.Lock()
	cp := *m.mk
	mu.Unlock()
	return cp.limit(cp.cells(text))
}

func (m *MaskedTextBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := m.text.bind()
	if changed && m.onChange != nil {
		Post(m.onChange)
	}
	if m.onEditFn == nil {
		// Made once: closures made in each build are an allocation each.
		m.onEditFn, m.limitFn = m.editText, m.caretLimit
	}
	el := ui.TextInput(c, p).OnEdit(m.onEditFn).
		// The caret never goes after the first empty slot that follows what
		// was typed, where what is typed next would find no room (WinForms
		// lets it).
		CaretLimit(m.limitFn)
	return el.OnChange(editedOnce(&m.edit, &m.text, &m.bound, &m.onChange))
}

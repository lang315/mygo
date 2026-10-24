package widget

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// LabelWidget shows a line of text.
type LabelWidget struct {
	common[LabelWidget]
	text  string
	spans []ui.Span
}

// Label makes a label showing s.
func Label(s string) *LabelWidget {
	l := &LabelWidget{text: s}
	l.init(l)
	return l
}

// SetText changes the text.
func (l *LabelWidget) SetText(s string) *LabelWidget {
	l.change(func() { l.text, l.spans = s, nil })
	return l
}

// Rich shows spans of styled text instead of plain text, where gota used
// Pango markup: bold is Weight 700, color is Color. Text returns their
// text; SetText makes the label plain again.
func (l *LabelWidget) Rich(spans ...ui.Span) *LabelWidget {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	l.change(func() { l.spans, l.text = spans, b.String() })
	return l
}

// Text returns the text.
func (l *LabelWidget) Text() string { mu.Lock(); defer mu.Unlock(); return l.text }

func (l *LabelWidget) build(c *ui.Context) ui.Element {
	if len(l.spans) > 0 {
		return ui.RichText(c, l.spans...)
	}
	return ui.Text(c, l.text)
}

// TextBoxWidget is a one-line text input.
type TextBoxWidget struct {
	common[TextBoxWidget]
	text        mirror[string]
	placeholder string
	password    bool
	readOnly    bool
	maxLen      int // runes, 0 for no limit
	onChange    func()
	onEnter     func()
}

// TextBox makes an empty one-line text input.
func TextBox() *TextBoxWidget {
	e := &TextBoxWidget{}
	e.init(e)
	return e
}

func (e *TextBoxWidget) Text() string { mu.Lock(); defer mu.Unlock(); return e.text.value }
func (e *TextBoxWidget) SetText(s string) *TextBoxWidget {
	e.change(func() { e.text.set(e.cut(s)) })
	return e
}

// cut returns s cut to the runes MaxLength allows. Under mu.
func (e *TextBoxWidget) cut(s string) string {
	if e.maxLen > 0 && utf8.RuneCountInString(s) > e.maxLen {
		return string([]rune(s)[:e.maxLen])
	}
	return s
}

// MaxLength limits the text to n runes, as WinForms' MaxLength does:
// what the user types or pastes past it is cut at it, and so is a text set
// by SetText, and the text the box holds when n is lowered. 0 means no
// limit.
func (e *TextBoxWidget) MaxLength(n int) *TextBoxWidget {
	e.change(func() {
		e.maxLen = max(n, 0)
		if s := e.cut(e.text.value); s != e.text.value {
			e.text.set(s)
		}
	})
	return e
}

// SetPlaceholder sets the text shown while the input is empty.
func (e *TextBoxWidget) SetPlaceholder(s string) *TextBoxWidget {
	e.change(func() { e.placeholder = s })
	return e
}

// Password hides what is typed.
func (e *TextBoxWidget) Password() *TextBoxWidget { e.change(func() { e.password = true }); return e }

// SetEditable lets the user edit the text, or not.
func (e *TextBoxWidget) SetEditable(on bool) *TextBoxWidget {
	e.change(func() { e.readOnly = !on })
	return e
}

// OnChange runs fn after each edit of the user.
func (e *TextBoxWidget) OnChange(fn func()) *TextBoxWidget {
	e.change(func() { e.onChange = fn })
	return e
}

// OnEnter runs fn when the user presses Enter.
func (e *TextBoxWidget) OnEnter(fn func()) *TextBoxWidget {
	e.change(func() { e.onEnter = fn })
	return e
}

func (e *TextBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := e.text.bind()
	el := ui.TextInput(c, p)
	if e.placeholder != "" {
		el.Placeholder(e.placeholder)
	}
	if e.password {
		el.Password()
	}
	el.ReadOnly(e.readOnly)
	el.MaxLength(e.maxLen)
	el.OnChange(edited(&e.text, &e.bound, e.onChange))
	if changed && e.onChange != nil {
		Post(e.onChange)
	}
	if h := e.onEnter; h != nil {
		el.OnSubmit(func() { Post(h) })
	}
	return el
}

// MultilineTextBoxWidget is a text input of several lines.
type MultilineTextBoxWidget struct {
	common[MultilineTextBoxWidget]
	text        mirror[string]
	placeholder string
	readOnly    bool
	onChange    func()
}

// MultilineTextBox makes an empty text input of several lines.
func MultilineTextBox() *MultilineTextBoxWidget {
	v := &MultilineTextBoxWidget{}
	v.init(v)
	return v
}

func (v *MultilineTextBoxWidget) Text() string { mu.Lock(); defer mu.Unlock(); return v.text.value }
func (v *MultilineTextBoxWidget) SetText(s string) *MultilineTextBoxWidget {
	v.change(func() { v.text.set(s) })
	return v
}
func (v *MultilineTextBoxWidget) SetPlaceholder(s string) *MultilineTextBoxWidget {
	v.change(func() { v.placeholder = s })
	return v
}
func (v *MultilineTextBoxWidget) SetEditable(on bool) *MultilineTextBoxWidget {
	v.change(func() { v.readOnly = !on })
	return v
}
func (v *MultilineTextBoxWidget) OnChange(fn func()) *MultilineTextBoxWidget {
	v.change(func() { v.onChange = fn })
	return v
}

func (v *MultilineTextBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := v.text.bind()
	el := ui.TextArea(c, p)
	if v.placeholder != "" {
		el.Placeholder(v.placeholder)
	}
	el.ReadOnly(v.readOnly)
	el.OnChange(onChange(&v.text, v.onChange))
	if changed && v.onChange != nil {
		Post(v.onChange)
	}
	return el
}

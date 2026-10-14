package widget

import "github.com/egoist/mygo/ui"

// LabelWidget shows a line of text.
type LabelWidget struct {
	common[LabelWidget]
	text string
}

// Label makes a label showing s.
func Label(s string) *LabelWidget {
	l := &LabelWidget{text: s}
	l.init(l)
	return l
}

// SetText changes the text.
func (l *LabelWidget) SetText(s string) *LabelWidget { change(func() { l.text = s }); return l }

// Text returns the text.
func (l *LabelWidget) Text() string { mu.Lock(); defer mu.Unlock(); return l.text }

func (l *LabelWidget) build(c *ui.Context) ui.Element { return ui.Text(c, l.text) }

// EntryWidget is a one-line text input.
type EntryWidget struct {
	common[EntryWidget]
	text        mirror[string]
	placeholder string
	password    bool
	readOnly    bool
	onChange    func()
	onEnter     func()
}

// Entry makes an empty one-line text input.
func Entry() *EntryWidget {
	e := &EntryWidget{}
	e.init(e)
	return e
}

func (e *EntryWidget) Text() string { mu.Lock(); defer mu.Unlock(); return e.text.value }
func (e *EntryWidget) SetText(s string) *EntryWidget {
	change(func() { e.text.set(s) })
	return e
}

// SetPlaceholder sets the text shown while the input is empty.
func (e *EntryWidget) SetPlaceholder(s string) *EntryWidget {
	change(func() { e.placeholder = s })
	return e
}

// Password hides what is typed.
func (e *EntryWidget) Password() *EntryWidget { change(func() { e.password = true }); return e }

// SetEditable lets the user edit the text, or not.
func (e *EntryWidget) SetEditable(on bool) *EntryWidget {
	change(func() { e.readOnly = !on })
	return e
}

// OnChange runs fn after each edit of the user.
func (e *EntryWidget) OnChange(fn func()) *EntryWidget { change(func() { e.onChange = fn }); return e }

// OnEnter runs fn when the user presses Enter.
func (e *EntryWidget) OnEnter(fn func()) *EntryWidget { change(func() { e.onEnter = fn }); return e }

func (e *EntryWidget) build(c *ui.Context) ui.Element {
	p, changed := e.text.bind()
	el := ui.TextInput(c, p)
	if e.placeholder != "" {
		el.Placeholder(e.placeholder)
	}
	if e.password {
		el.Password()
	}
	el.ReadOnly(e.readOnly)
	el.OnChange(onChange(&e.text, e.onChange))
	if changed && e.onChange != nil {
		Post(e.onChange)
	}
	if h := e.onEnter; h != nil {
		el.OnSubmit(func() { Post(h) })
	}
	return el
}

// TextViewWidget is a text input of several lines.
type TextViewWidget struct {
	common[TextViewWidget]
	text        mirror[string]
	placeholder string
	readOnly    bool
	onChange    func()
}

// TextView makes an empty text input of several lines.
func TextView() *TextViewWidget {
	v := &TextViewWidget{}
	v.init(v)
	return v
}

func (v *TextViewWidget) Text() string { mu.Lock(); defer mu.Unlock(); return v.text.value }
func (v *TextViewWidget) SetText(s string) *TextViewWidget {
	change(func() { v.text.set(s) })
	return v
}
func (v *TextViewWidget) SetPlaceholder(s string) *TextViewWidget {
	change(func() { v.placeholder = s })
	return v
}
func (v *TextViewWidget) SetEditable(on bool) *TextViewWidget {
	change(func() { v.readOnly = !on })
	return v
}
func (v *TextViewWidget) OnChange(fn func()) *TextViewWidget {
	change(func() { v.onChange = fn })
	return v
}

func (v *TextViewWidget) build(c *ui.Context) ui.Element {
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

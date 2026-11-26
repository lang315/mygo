package widget

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// LabelWidget shows a line of text.
type LabelWidget struct {
	common[LabelWidget]
	text        string
	spans       []ui.Span
	useMnemonic bool      // SetUseMnemonic(true)
	role        *textRole // a role the widgets give their own labels, as a dialog's title
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
	e := l.textElement(c)
	if l.role != nil {
		l.role.apply(c, e) // under the label's own style, which render applies
	}
	return e
}

// textElement builds the label's text.
func (l *LabelWidget) textElement(c *ui.Context) ui.Element {
	if len(l.spans) > 0 {
		return ui.RichText(c, l.spans...)
	}
	if !l.useMnemonic || !strings.Contains(l.text, "&") {
		return ui.Text(c, l.text)
	}
	display, key, at := parseMnemonic(l.text)
	if key == 0 || macMenus || buildingForm == nil {
		return ui.Text(c, display)
	}
	if c.Modifiers()&ui.Alt != 0 {
		e := ui.RichText(c, mnemonicSpans(display, at)...)
		registerLabelMnemonic(key, e, l)
		return e
	}
	e := ui.Text(c, display)
	registerLabelMnemonic(key, e, l)
	return e
}

// SetUseMnemonic says whether an "&" in the text marks a key, as WinForms'
// UseMnemonic: Alt and the key then give the keyboard focus to the next
// control that takes it, and the "&" is not shown ("&&" is one). Unlike
// WinForms it is off by default: a label shows data (a URL "a=1&b=2", a name
// "Tom & Jerry") and would lose its ampersands. The labels of Fields rows
// have mnemonics without it.
func (l *LabelWidget) SetUseMnemonic(on bool) *LabelWidget {
	l.change(func() { l.useMnemonic = on })
	return l
}

// TextBoxWidget is a one-line text input.
type TextBoxWidget struct {
	common[TextBoxWidget]
	text        mirror[string]
	placeholder string
	password    bool
	readOnly    bool
	maxLen      int // runes, 0 for no limit
	icon        *ui.Bitmap
	onChange    func()
	onEnter     func()
	edit        func() // onEdited, made once
	selectAll   bool   // SelectAll asked, not yet carried out

	suggestions []string // SetSuggestions's, then what Enter remembered
	mru         int      // SetMRU's
	remember    func()   // rememberText, set by SetMRU
	accept      func()   // formAccept at the last build, for rememberText
}

// autocomplete is ui.Autocomplete, set by SetSuggestions and SetMRU: an app
// that calls neither does not link it. Under mu.
var autocomplete func(c *ui.Context, value *string, suggestions []string) ui.Element

// SetSuggestions suggests items below the box, those containing what was
// typed, those starting with it first, as ui.Autocomplete does: Up and Down
// move among them and Enter takes one, as does a click. While there are
// none, the box is a plain one. A Password box ignores them.
func (e *TextBoxWidget) SetSuggestions(items ...string) *TextBoxWidget {
	items = slices.Clone(items) // ui reads the slice while it draws
	e.change(func() { e.suggestions, autocomplete = items, ui.Autocomplete })
	return e
}

// SetMRU makes the box remember what the user entered, as DevExpress's
// MRUEdit: after Enter, the text moves to the front of the suggestions,
// without duplicates, keeping at most max of them. 0 turns it off. A
// Password box ignores it.
func (e *TextBoxWidget) SetMRU(n int) *TextBoxWidget {
	e.change(func() {
		e.mru = max(n, 0)
		if e.mru > 0 && e.remember == nil {
			e.remember, autocomplete = e.rememberText, ui.Autocomplete
		}
	})
	return e
}

// rememberText is ui's OnSubmit callback of an MRU box: it puts the text in
// front of the suggestions, then runs OnEnter, or the form's accept button.
func (e *TextBoxWidget) rememberText() {
	s := e.text.shown // ui's, on the drawing thread as this callback
	mu.Lock()
	if s != "" && e.mru > 0 && !e.password {
		next := make([]string, 0, min(len(e.suggestions)+1, e.mru))
		next = append(next, s)
		for _, x := range e.suggestions {
			if len(next) == e.mru {
				break
			}
			if x != s {
				next = append(next, x)
			}
		}
		e.suggestions = next // a new slice: ui may still read the old one
	}
	h, accept := e.onEnter, e.accept
	mu.Unlock()
	if h != nil {
		Post(h)
	} else if accept != nil {
		accept()
	}
}

// TextBox makes an empty one-line text input.
func TextBox() *TextBoxWidget {
	e := &TextBoxWidget{}
	e.init(e)
	e.lookAs(skinhook.KindField)
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

// Icon shows img, 16 DIPs square, before the text inside the box, as gota's
// entry icon does, at its left; nil removes it.
func (e *TextBoxWidget) Icon(img *ui.Bitmap) *TextBoxWidget {
	e.change(func() { e.icon = img })
	return e
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

// SelectAll selects all the text, as a rename dialog that opens with the
// name selected does: call it with Focus. It does not give the keyboard
// focus, and it selects once, in the next frame the box is shown in.
func (e *TextBoxWidget) SelectAll() *TextBoxWidget {
	e.change(func() { e.selectAll = true })
	return e
}

// OnEnter runs fn when the user presses Enter.
func (e *TextBoxWidget) OnEnter(fn func()) *TextBoxWidget {
	e.change(func() { e.onEnter = fn })
	return e
}

// onEdited is the callback of ui's OnChange, made once for the life of the
// widget: building it again every frame allocates a closure for each text
// box. It takes the control's OnChange and its binding at the moment of the
// edit.
func (e *TextBoxWidget) onEdited() {
	mu.Lock()
	fn := edited(&e.text, &e.bound, e.onChange)
	mu.Unlock()
	fn()
}

func (e *TextBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := e.text.bind()
	var el ui.Element
	if len(e.suggestions) > 0 && !e.password {
		el = autocomplete(c, p, e.suggestions)
	} else {
		el = ui.TextInput(c, p)
	}
	if e.placeholder != "" {
		el.Placeholder(e.placeholder)
		if tk, _ := skinOf(c); tk != nil {
			el.PlaceholderColor(tk.TextPlaceholder)
		}
	}
	if e.password {
		el.Password()
	}
	el.ReadOnly(e.readOnly)
	el.MaxLength(e.maxLen)
	if e.selectAll && !c.Measuring() {
		el.SetTextSelection(0, utf8.RuneCountInString(*p))
		e.selectAll = false
	}
	if icon := e.icon; icon != nil {
		// The icon's room is added to the start padding once the user's
		// styles have set it, and the icon drawn in the padding.
		e.post = func(c *ui.Context, el ui.Element) {
			t := c.Theme()
			side, rtl := 3, c.Direction() == ui.RightToLeft
			if rtl {
				side = 1
			}
			pad, iconSize := el.Paddings()[side], iconSide(c)
			el.PaddingStart(pad + iconSize + t.Space(1.5))
			e.over = append(e.over, func(p *ui.Painter, r ui.Rect) {
				x := r.X + pad
				if rtl {
					x = r.X + r.W - pad - iconSize
				}
				p.Image(icon, ui.Rect{X: x, Y: r.Y + (r.H-iconSize)/2, W: iconSize, H: iconSize}, ui.Contain)
			})
		}
	}
	if e.errMsg != "" {
		// The error mark sits at the end: the text stops before it.
		prev := e.post
		e.post = func(c *ui.Context, el ui.Element) {
			if prev != nil {
				prev(c, el)
			}
			el.TextInsetEnd(errorMarkSize + errorMarkInset + 2)
		}
	}
	if e.edit == nil {
		e.edit = e.onEdited
	}
	el.OnChange(e.edit)
	if changed && e.onChange != nil {
		Post(e.onChange)
	}
	if e.mru > 0 && !e.password {
		e.accept = formAccept
		el.OnSubmit(e.remember)
	} else if h := e.onEnter; h != nil {
		el.OnSubmit(func() { Post(h) })
	} else if h := formAccept; h != nil {
		el.OnSubmit(h) // Enter in the line clicks the form's accept button
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
	edit        func() // onEdited, made once
	selectAll   bool   // SelectAll asked, not yet carried out
}

// MultilineTextBox makes an empty text input of several lines.
func MultilineTextBox() *MultilineTextBoxWidget {
	v := &MultilineTextBoxWidget{}
	v.init(v)
	v.lookAs(skinhook.KindField)
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

// SelectAll selects all the text (see TextBoxWidget.SelectAll).
func (v *MultilineTextBoxWidget) SelectAll() *MultilineTextBoxWidget {
	v.change(func() { v.selectAll = true })
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

// onEdited is the callback of ui's OnChange, made once (see TextBoxWidget).
func (v *MultilineTextBoxWidget) onEdited() {
	mu.Lock()
	fn := edited(&v.text, &v.bound, v.onChange)
	mu.Unlock()
	fn()
}

func (v *MultilineTextBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := v.text.bind()
	el := ui.TextArea(c, p).Grow(1) // as a table or a list does: it takes the room a box or a page gives
	if v.placeholder != "" {
		el.Placeholder(v.placeholder)
		if tk, _ := skinOf(c); tk != nil {
			el.PlaceholderColor(tk.TextPlaceholder)
		}
	}
	el.ReadOnly(v.readOnly)
	if v.selectAll && !c.Measuring() {
		el.SetTextSelection(0, utf8.RuneCountInString(*p))
		v.selectAll = false
	}
	if c.Measuring() {
		el.Lines(1, 8) // a window fitted to its content counts a few lines
	}
	if v.edit == nil {
		v.edit = v.onEdited
	}
	el.OnChange(v.edit)
	if changed && v.onChange != nil {
		Post(v.onChange)
	}
	return el
}

func (v *MultilineTextBoxWidget) fillsAcross() bool { return true }

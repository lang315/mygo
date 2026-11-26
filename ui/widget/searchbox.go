package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// SearchBoxWidget is a field for searching, as DevExpress's SearchControl:
// a magnifying glass, the text, and a button clearing it while it holds
// any, as Escape does then. It is ui.SearchField underneath, and takes the
// skin's look of a TextBox.
type SearchBoxWidget struct {
	edit func() // editedOnce's
	common[SearchBoxWidget]
	text        mirror[string]
	placeholder string
	onChange    func()
	onSubmit    func()
	submit      func() // submitted, made once
}

// SearchBox makes an empty search box, which shows Search while empty.
func SearchBox() *SearchBoxWidget {
	s := &SearchBoxWidget{}
	s.init(s)
	s.lookAs(skinhook.KindField)
	return s
}

// Text returns what the box holds.
func (s *SearchBoxWidget) Text() string { mu.Lock(); defer mu.Unlock(); return s.text.value }

// SetText sets what the box holds, without running OnChange.
func (s *SearchBoxWidget) SetText(v string) *SearchBoxWidget {
	s.change(func() { s.text.set(v) })
	return s
}

// SetPlaceholder sets the text shown while the box is empty, in place of
// Search.
func (s *SearchBoxWidget) SetPlaceholder(v string) *SearchBoxWidget {
	s.change(func() { s.placeholder = v })
	return s
}

// OnChange runs fn, on the event goroutine, after each edit of the user,
// the clear button and Escape included. A setter does not run it.
func (s *SearchBoxWidget) OnChange(fn func()) *SearchBoxWidget {
	s.change(func() { s.onChange = fn })
	return s
}

// OnSubmit runs fn, on the event goroutine, when the user presses Enter.
func (s *SearchBoxWidget) OnSubmit(fn func()) *SearchBoxWidget {
	s.change(func() { s.onSubmit = fn })
	return s
}

// submitted is ui's OnSubmit callback, made once.
func (s *SearchBoxWidget) submitted() {
	mu.Lock()
	h := s.onSubmit
	mu.Unlock()
	if h != nil {
		Post(h)
	}
}

func (s *SearchBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := s.text.bind()
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	el := ui.SearchField(c, p)
	if s.placeholder != "" {
		el.Placeholder(s.placeholder)
	}
	if tk, _ := skinOf(c); tk != nil {
		el.PlaceholderColor(tk.TextPlaceholder)
	}
	if s.submit == nil {
		s.submit = s.submitted
	}
	el.OnSubmit(s.submit)
	return el.OnChange(editedOnce(&s.edit, &s.text, &s.bound, &s.onChange))
}

func (s *SearchBoxWidget) bindValue() string  { return s.Text() }
func (s *SearchBoxWidget) bindApply(v string) { s.SetText(v) }

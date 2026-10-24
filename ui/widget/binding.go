package widget

import "math"

// Bindable is a control that edits one value: a TextBox (string), a
// CheckBox (bool), a ComboBox (the Value of its option, a string) or a
// NumericUpDown (float64). Only these are bindable; the interface cannot be
// implemented outside the package.
type Bindable[T comparable] interface {
	Control
	bindValue() T  // the control's value, as its getter
	bindApply(v T) // sets the control as a program setter does
}

// binding is what a control keeps of its binding, under mu.
type binding struct {
	write  func(v any) // gives the source the value the user entered, a T
	reload func()      // gives the source's value to the control
}

// edited returns ui's OnChange callback for m, a control's value: it
// commits what the user entered, and runs h, the control's OnChange, then
// the binding's write of that value, on the event goroutine. The value is
// taken as it is committed, under mu, so a setter that runs after it, in h
// or on another goroutine, is not what the source gets. Under mu, while
// building. A control with no binding gets the plain callback, so it costs
// nothing.
func edited[T comparable](m *mirror[T], b *binding, h func()) func() {
	if b.write == nil {
		return onChange(m, h)
	}
	return func() {
		mu.Lock()
		changed := m.value != m.shown
		m.value, m.dirty = m.shown, false
		v, write := m.value, b.write // the binding of the moment of the edit
		mu.Unlock()
		if !changed {
			return
		}
		Post(func() {
			if h != nil {
				h()
			}
			if write != nil {
				write(v)
			}
		})
	}
}

// Bind binds c to a value kept elsewhere, as a config is, and returns c.
// The control takes get()'s value at once, and after each edit of the
// user, get's counterpart set runs with the new value on the event
// goroutine, after the control's own OnChange (Bind does not replace it)
// and off the package's lock, so set may call the control's getters.
//
// A program setter (SetText, SetChecked, SetValue) does not call set: the
// program knows the value it gave. Nor does Reload, nor a value the control
// takes from get that is not the control's to keep: a ComboBox with no
// option of that value stays empty, as SetValue leaves it, and a
// NumericUpDown clamps it to its range; neither is written back. Binding a
// control again replaces its binding.
//
//	widget.Bind(name, func() string { return config.Get("name") }, func(v string) { config.Set("name", v) })
func Bind[C Bindable[T], T comparable](c C, get func() T, set func(T)) C {
	reload := func() { c.bindApply(get()) }
	c.base().change(func() {
		c.base().bound = binding{write: func(v any) { set(v.(T)) }, reload: reload}
	})
	reload()
	return c
}

// BindInt is Bind for a NumericUpDown whose value the source keeps as an
// int: the control's value is rounded to the nearest int for set.
func BindInt(c *NumericUpDownWidget, get func() int, set func(int)) *NumericUpDownWidget {
	return Bind(c, func() float64 { return float64(get()) }, func(v float64) { set(int(math.Round(v))) })
}

// Reload gives c the value of its binding's source again, as Bind did, for
// a source that changed elsewhere. It does nothing for a control that is
// not bound, and does not call set.
func Reload(c Control) {
	mu.Lock()
	reload := c.base().bound.reload
	mu.Unlock()
	if reload != nil {
		reload()
	}
}

func (e *TextBoxWidget) bindValue() string  { return e.Text() }
func (e *TextBoxWidget) bindApply(v string) { e.SetText(v) }

func (b *CheckBoxWidget) bindValue() bool  { return b.Checked() }
func (b *CheckBoxWidget) bindApply(v bool) { b.SetChecked(v) }

func (c *ComboBoxWidget) bindValue() string {
	mu.Lock()
	defer mu.Unlock()
	if c.editable && c.bindsText {
		return c.text.value
	}
	o, _ := c.chosen()
	return o.Value
}

func (c *ComboBoxWidget) bindApply(v string) {
	mu.Lock()
	text := c.editable && c.bindsText
	mu.Unlock()
	if text {
		c.SetText(v)
		return
	}
	c.SetValue(v)
}

func (s *NumericUpDownWidget) bindValue() float64 { return s.Value() }
func (s *NumericUpDownWidget) bindApply(v float64) {
	s.SetValue(min(max(v, s.min), s.max))
}

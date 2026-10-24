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
	write  func() // reads the control's value and gives it to the source
	reload func() // gives the source's value to the control
}

// after returns the handler of the user's edit: h, the control's OnChange,
// then the binding's write, if it has one. Under mu, while building. A
// control with no binding gets h itself, so it costs nothing.
func (b *binding) after(h func()) func() {
	if b.write == nil {
		return h
	}
	return func() {
		if h != nil {
			h()
		}
		mu.Lock()
		write := b.write
		mu.Unlock()
		if write != nil {
			write()
		}
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
		c.base().bound = binding{write: func() { set(c.bindValue()) }, reload: reload}
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

func (c *ComboBoxWidget) bindValue() string  { return c.Value() }
func (c *ComboBoxWidget) bindApply(v string) { c.SetValue(v) }

func (s *NumericUpDownWidget) bindValue() float64 { return s.Value() }
func (s *NumericUpDownWidget) bindApply(v float64) {
	s.SetValue(min(max(v, s.min), s.max))
}

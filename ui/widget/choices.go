package widget

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// SpinWidget edits a number between min and max.
type SpinWidget struct {
	common[SpinWidget]
	min, max, step float64
	value          mirror[float64]
	onChange       func()
}

// Spin makes a number input from min to max, stepping by step.
func Spin(min, max, step float64) *SpinWidget {
	s := &SpinWidget{min: min, max: max, step: step}
	s.value.set(min)
	s.init(s)
	return s
}

func (s *SpinWidget) Value() float64 { mu.Lock(); defer mu.Unlock(); return s.value.value }
func (s *SpinWidget) Int() int       { return int(math.Round(s.Value())) }
func (s *SpinWidget) SetValue(v float64) *SpinWidget {
	change(func() { s.value.set(v) })
	return s
}
func (s *SpinWidget) OnChange(fn func()) *SpinWidget { change(func() { s.onChange = fn }); return s }

func (s *SpinWidget) build(c *ui.Context) ui.Element {
	p, changed := s.value.bind()
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	return ui.NumberInput(c, p, s.min, s.max, s.step).OnChange(onChange(&s.value, s.onChange))
}

// RangeSpinWidget edits a range: two numbers, with text between and after.
type RangeSpinWidget struct {
	common[RangeSpinWidget]
	lo, hi          *SpinWidget
	between, suffix string
}

// RangeSpin makes two number inputs, the low one from min1 to max1 and the
// high one from min2 to max2, with between and suffix shown around them.
func RangeSpin(min1, max1, min2, max2, step float64, between, suffix string) *RangeSpinWidget {
	r := &RangeSpinWidget{lo: Spin(min1, max1, step), hi: Spin(min2, max2, step), between: between, suffix: suffix}
	r.init(r)
	return r
}

func (r *RangeSpinWidget) Values() (lo, hi float64) { return r.lo.Value(), r.hi.Value() }
func (r *RangeSpinWidget) Ints() (lo, hi int)       { return r.lo.Int(), r.hi.Int() }
func (r *RangeSpinWidget) SetValues(lo, hi float64) *RangeSpinWidget {
	r.lo.SetValue(lo)
	r.hi.SetValue(hi)
	return r
}
func (r *RangeSpinWidget) OnChange(fn func()) *RangeSpinWidget {
	r.lo.OnChange(fn)
	r.hi.OnChange(fn)
	return r
}

func (r *RangeSpinWidget) build(c *ui.Context) ui.Element {
	return ui.Row(c).Gap(c.Theme().Space(2)).Children(func() {
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		render(c, r.lo)
		if r.between != "" {
			ui.Text(c, r.between)
		}
		render(c, r.hi)
		if r.suffix != "" {
			ui.Text(c, r.suffix)
		}
	})
}

// ComboWidget is a drop-down choice of one option.
//
// shortcut: built on ui.Select, which chooses by text, so option texts
// must be unique; PR 2 moves it to ui.SelectBase for sections and icons.
type ComboWidget struct {
	common[ComboWidget]
	opts     []Option
	text     mirror[string] // the chosen option's text, "" for none
	onChange func()
}

func Combo(opts ...Option) *ComboWidget {
	c := &ComboWidget{opts: opts}
	c.init(c)
	return c
}

// Value returns the chosen option's value, "" for none.
func (c *ComboWidget) Value() string { o, _ := c.Selected(); return o.Value }

// Selected returns the chosen option.
func (c *ComboWidget) Selected() (Option, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, o := range c.opts {
		if o.Text == c.text.value && o.Text != "" {
			return o, true
		}
	}
	return Option{}, false
}

// SetValue chooses the option of value v; an unknown v chooses none.
func (c *ComboWidget) SetValue(v string) *ComboWidget {
	change(func() {
		t := ""
		for _, o := range c.opts {
			if o.Value == v {
				t = o.Text
			}
		}
		c.text.set(t)
	})
	return c
}

func (c *ComboWidget) Unselect() *ComboWidget { change(func() { c.text.set("") }); return c }

// SetOptions replaces the options; the choice stays if its text remains.
func (c *ComboWidget) SetOptions(opts ...Option) *ComboWidget {
	change(func() { c.opts = opts })
	return c
}

func (c *ComboWidget) OnChange(fn func()) *ComboWidget { change(func() { c.onChange = fn }); return c }

func (c *ComboWidget) build(ctx *ui.Context) ui.Element {
	p, changed := c.text.bind()
	if changed && c.onChange != nil {
		Post(c.onChange)
	}
	texts := make([]string, len(c.opts))
	for i, o := range c.opts {
		texts[i] = o.Text
	}
	return ui.Select(ctx, p, texts).OnChange(onChange(&c.text, c.onChange))
}

// TagSelectWidget is a choice of several options shown as chips.
type TagSelectWidget struct {
	common[TagSelectWidget]
	opts     []Option
	on       []mirror[bool]
	onChange func()
}

func TagSelect(opts ...Option) *TagSelectWidget {
	g := &TagSelectWidget{opts: opts, on: make([]mirror[bool], len(opts))}
	g.init(g)
	return g
}

// Values returns the values of the chosen options, in option order.
func (g *TagSelectWidget) Values() []string {
	mu.Lock()
	defer mu.Unlock()
	var vs []string
	for i, o := range g.opts {
		if g.on[i].value {
			vs = append(vs, o.Value)
		}
	}
	return vs
}

func (g *TagSelectWidget) SetValues(vs ...string) *TagSelectWidget {
	change(func() {
		for i, o := range g.opts {
			g.on[i].set(slices.Contains(vs, o.Value))
		}
	})
	return g
}

func (g *TagSelectWidget) OnChange(fn func()) *TagSelectWidget {
	change(func() { g.onChange = fn })
	return g
}

func (g *TagSelectWidget) build(c *ui.Context) ui.Element {
	return ui.Row(c).Wrap().Gap(c.Theme().Space(1.5)).Children(func() {
		for i, o := range g.opts {
			p, changed := g.on[i].bind()
			ui.Toggle(c.Key(i), p, o.Text).OnChange(onChange(&g.on[i], g.onChange))
			if changed && g.onChange != nil {
				Post(g.onChange)
			}
		}
	})
}

// ProgressWidget shows how far a task is.
type ProgressWidget struct {
	common[ProgressWidget]
	fraction float64
	text     string
}

func Progress() *ProgressWidget {
	p := &ProgressWidget{}
	p.init(p)
	return p
}

// SetFraction sets how far the task is, 0 to 1; a negative fraction shows
// a task of unknown length.
func (p *ProgressWidget) SetFraction(f float64) *ProgressWidget {
	change(func() { p.fraction = f })
	return p
}

// SetText sets the text shown under the bar.
func (p *ProgressWidget) SetText(s string) *ProgressWidget { change(func() { p.text = s }); return p }

func (p *ProgressWidget) build(c *ui.Context) ui.Element {
	return ui.Column(c).Gap(c.Theme().Space(1)).Children(func() {
		ui.Progress(c, p.fraction)
		if p.text != "" {
			ui.Text(c, p.text).TextColor(c.Theme().TextMuted)
		}
	})
}

// SpinnerWidget shows that something is busy while started.
type SpinnerWidget struct {
	common[SpinnerWidget]
	running bool
}

func Spinner() *SpinnerWidget {
	s := &SpinnerWidget{}
	s.init(s)
	return s
}

func (s *SpinnerWidget) Start() *SpinnerWidget { change(func() { s.running = true }); return s }
func (s *SpinnerWidget) Stop() *SpinnerWidget  { change(func() { s.running = false }); return s }

func (s *SpinnerWidget) build(c *ui.Context) ui.Element {
	if s.running {
		return ui.Spinner(c)
	}
	return ui.Box(c)
}

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
	onEnter        func()
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

// OnEnter runs fn when the user presses Enter in the number.
func (s *SpinWidget) OnEnter(fn func()) *SpinWidget { change(func() { s.onEnter = fn }); return s }

func (s *SpinWidget) OnChange(fn func()) *SpinWidget { change(func() { s.onChange = fn }); return s }

func (s *SpinWidget) build(c *ui.Context) ui.Element {
	p, changed := s.value.bind()
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	e := ui.NumberInput(c, p, s.min, s.max, s.step).OnChange(onChange(&s.value, s.onChange))
	// Enter in the number's text field: it is inside e, which reports it.
	if h := s.onEnter; h != nil && e.Shortcut(0, ui.KeyEnter) {
		Post(h)
	}
	return e
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

// ComboWidget is a drop-down choice of one option, optionally grouped in
// titled sections with icons, as gota's ComboBox. Values must be unique;
// texts need not be.
type ComboWidget struct {
	common[ComboWidget]
	sections []comboSection // [0]: the options outside any section
	value    mirror[string] // the chosen option's Value, "" for none
	onChange func()
}

// comboSection is a titled group of options.
type comboSection struct {
	title string
	icon  *ui.Bitmap
	opts  []Option
}

// sectionKey keys a section's header row in the popup.
type sectionKey int

// chevronDown is Lucide's chevron-down (ISC license).
var chevronDown = lucide(`<path d="m6 9 6 6 6-6"/>`)

// Combo makes a drop-down of opts, none chosen.
func Combo(opts ...Option) *ComboWidget {
	c := &ComboWidget{sections: []comboSection{{opts: opts}}}
	c.init(c)
	return c
}

// find returns the option of value v. Under mu.
func (c *ComboWidget) find(v string) (Option, bool) {
	if v == "" {
		return Option{}, false
	}
	for _, s := range c.sections {
		for _, o := range s.opts {
			if o.Value == v {
				return o, true
			}
		}
	}
	return Option{}, false
}

// edit applies fn to a copy of the sections, which a view may still be
// reading, then unchooses a choice whose option is gone. Under mu.
func (c *ComboWidget) edit(fn func(s []comboSection) []comboSection) {
	c.sections = fn(slices.Clone(c.sections))
	if _, ok := c.find(c.value.value); !ok && c.value.value != "" {
		c.value.set("")
	}
}

// Value returns the chosen option's value, "" for none.
func (c *ComboWidget) Value() string { mu.Lock(); defer mu.Unlock(); return c.value.value }

// Selected returns the chosen option.
func (c *ComboWidget) Selected() (Option, bool) {
	mu.Lock()
	defer mu.Unlock()
	return c.find(c.value.value)
}

// SetValue chooses the option of value v; an unknown v chooses none.
func (c *ComboWidget) SetValue(v string) *ComboWidget {
	change(func() {
		if _, ok := c.find(v); !ok {
			v = ""
		}
		c.value.set(v)
	})
	return c
}

func (c *ComboWidget) Unselect() *ComboWidget { change(func() { c.value.set("") }); return c }

// SetOptions replaces the options outside any section.
func (c *ComboWidget) SetOptions(opts ...Option) *ComboWidget {
	change(func() {
		c.edit(func(s []comboSection) []comboSection { s[0].opts = opts; return s })
	})
	return c
}

// AddSection adds a section titled title, with the icon in the bytes of
// a PNG (nil for none), holding opts.
func (c *ComboWidget) AddSection(title string, icon []byte, opts ...Option) *ComboWidget {
	bmp := decodeImage(icon) // outside mu
	change(func() {
		c.edit(func(s []comboSection) []comboSection { return append(s, comboSection{title, bmp, opts}) })
	})
	return c
}

// RemoveSection removes the section titled title and its options.
func (c *ComboWidget) RemoveSection(title string) *ComboWidget {
	change(func() {
		c.edit(func(s []comboSection) []comboSection {
			return slices.DeleteFunc(s, func(x comboSection) bool { return x.title != "" && x.title == title })
		})
	})
	return c
}

// RemoveOption removes the option of value v.
func (c *ComboWidget) RemoveOption(v string) *ComboWidget {
	change(func() {
		c.edit(func(s []comboSection) []comboSection {
			for i := range s {
				s[i].opts = slices.DeleteFunc(slices.Clone(s[i].opts), func(o Option) bool { return o.Value == v })
			}
			return s
		})
	})
	return c
}

// SetOptionText changes the text of the option of value v.
func (c *ComboWidget) SetOptionText(v, text string) *ComboWidget {
	change(func() {
		c.edit(func(s []comboSection) []comboSection {
			for i := range s {
				opts := slices.Clone(s[i].opts)
				for j := range opts {
					if opts[j].Value == v {
						opts[j].Text = text
					}
				}
				s[i].opts = opts
			}
			return s
		})
	})
	return c
}

// SectionLen returns how many options the section titled title holds.
func (c *ComboWidget) SectionLen(title string) int {
	mu.Lock()
	defer mu.Unlock()
	for _, s := range c.sections {
		if s.title == title {
			return len(s.opts)
		}
	}
	return 0
}

func (c *ComboWidget) OnChange(fn func()) *ComboWidget { change(func() { c.onChange = fn }); return c }

func (c *ComboWidget) build(ctx *ui.Context) ui.Element {
	p, changed := c.value.bind()
	if changed && c.onChange != nil {
		Post(c.onChange)
	}
	t, sections := ctx.Theme(), c.sections
	text := ""
	if o, ok := c.find(*p); ok {
		text = o.Text
	}
	sel := ui.SelectBase(ctx, p)
	// The look of ui.Select (ui/widgets.go coreSelect), from public API.
	sel.Trigger.Padding(t.Space(1.5), t.Space(3)).Radius(t.Radius).Border(1, t.Border).Background(t.Surface).
		Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(t.Space(2)).MinWidth(t.Space(35)).
		OnChange(onChange(&c.value, c.onChange)).
		Children(func() {
			ui.Text(ctx, text).SingleLine()
			ui.Icon(ctx, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
		})
	sel.Popup(func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Radius+2).
			Background(t.Background).Border(1, t.Border).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18))
		for i, s := range sections {
			if s.title != "" {
				ui.Row(ctx.Key(sectionKey(i))).Gap(t.Space(1.5)).Padding(t.Space(1.5), t.Space(2.5)).
					AlignItems(ui.Center).Children(func() {
					if s.icon != nil {
						ui.Image(ctx, s.icon).Width(16).Height(16)
					}
					ui.Text(ctx, s.title).Bold().TextColor(t.TextMuted)
				})
			}
			for _, o := range s.opts {
				item := sel.Item(o.Value).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius)
				switch {
				case item.Highlighted():
					item.Background(t.Accent).TextColor(t.AccentText)
				case o.Value == *p:
					item.Background(t.Surface)
				}
				item.Children(func() { ui.Text(ctx, o.Text).SingleLine() })
			}
		}
	})
	return sel.Trigger
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

package widget

import (
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// NumericUpDownWidget edits a number between min and max.
type NumericUpDownWidget struct {
	edit func() // editedOnce's
	common[NumericUpDownWidget]
	min, max, step float64
	value          mirror[float64]
	onChange       func()
	onEnter        func()
	name           string
}

// NumericUpDown makes a number input from min to max, stepping by step.
func NumericUpDown(min, max, step float64) *NumericUpDownWidget {
	s := &NumericUpDownWidget{min: min, max: max, step: step}
	s.value.set(min)
	s.init(s)
	return s
}

func (s *NumericUpDownWidget) Value() float64 { mu.Lock(); defer mu.Unlock(); return s.value.value }
func (s *NumericUpDownWidget) Int() int       { return int(math.Round(s.Value())) }
func (s *NumericUpDownWidget) SetValue(v float64) *NumericUpDownWidget {
	s.change(func() { s.value.set(v) })
	return s
}

// OnEnter runs fn when the user presses Enter in the number.
func (s *NumericUpDownWidget) OnEnter(fn func()) *NumericUpDownWidget {
	s.change(func() { s.onEnter = fn })
	return s
}

func (s *NumericUpDownWidget) OnChange(fn func()) *NumericUpDownWidget {
	s.change(func() { s.onChange = fn })
	return s
}

// SetAccessibleName names the number for assistive technology, which reads it
// before the buttons' "Increase" and "Decrease": "Copies Increase".
func (s *NumericUpDownWidget) SetAccessibleName(name string) *NumericUpDownWidget {
	s.change(func() { s.name = name })
	return s
}

func (s *NumericUpDownWidget) build(c *ui.Context) ui.Element {
	p, changed := s.value.bind()
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	e := ui.NumberInput(c, p, s.min, s.max, s.step).OnChange(editedOnce(&s.edit, &s.value, &s.bound, &s.onChange))
	if s.name != "" {
		e.Label(s.name)
	}
	if e.Submitted() {
		if h := s.onEnter; h != nil {
			Post(h)
		} else if h := formAccept; h != nil {
			h() // Enter clicks the form's accept button
		}
	}
	return e
}

// RangeSpinWidget edits a range: two numbers, with text between and after.
type RangeSpinWidget struct {
	common[RangeSpinWidget]
	lo, hi          *NumericUpDownWidget
	between, suffix string
}

// RangeSpin makes two number inputs, the low one from min1 to max1 and the
// high one from min2 to max2, with between and suffix shown around them.
func RangeSpin(min1, max1, min2, max2, step float64, between, suffix string) *RangeSpinWidget {
	r := &RangeSpinWidget{lo: NumericUpDown(min1, max1, step), hi: NumericUpDown(min2, max2, step), between: between, suffix: suffix}
	r.lo.name, r.hi.name = "minimum", "maximum"
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

// SetAccessibleNames names the two numbers for assistive technology; without
// them they are "minimum" and "maximum", as a range slider's knobs are.
func (r *RangeSpinWidget) SetAccessibleNames(low, high string) *RangeSpinWidget {
	r.lo.SetAccessibleName(low)
	r.hi.SetAccessibleName(high)
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

// ComboBoxWidget is a drop-down choice of one option, optionally grouped in
// titled sections with icons, as gota's ComboBox. Values must be unique;
// texts need not be.
type ComboBoxWidget struct {
	common[ComboBoxWidget]
	sections []comboSection // [0]: the options outside any section
	value    mirror[string] // the chosen option's Value, "" for none
	onChange func()
	// editable lets the user type any text, held in text; Value is then
	// the Value of the option with that text. bindsText says Bind takes
	// the text, not the Value, of an editable combo (Editable, then Bind).
	editable, bindsText bool
	text                mirror[string]
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

// ComboBox makes a drop-down of opts, none chosen.
func ComboBox(opts ...Option) *ComboBoxWidget {
	c := &ComboBoxWidget{sections: []comboSection{{opts: slices.Clone(opts)}}}
	c.init(c)
	return c
}

// find returns the option of value v. Under mu.
func (c *ComboBoxWidget) find(v string) (Option, bool) {
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

// edit applies fn to the sections, then unchooses a choice whose option is
// gone. Under mu.
func (c *ComboBoxWidget) edit(fn func(s []comboSection) []comboSection) {
	c.sections = fn(c.sections)
	if c.editable {
		return
	}
	if _, ok := c.find(c.value.value); !ok && c.value.value != "" {
		c.value.set("")
	}
}

// Editable lets the user type a text that is not one of the options, in
// the box itself, as a ComboBox with DropDownStyle DropDown does in
// WinForms; the options show below it, those that match what is typed.
// Text and SetText give and set the text; Value is the Value of the first
// option whose Text is the text, "" when it matches none, and SetValue puts
// the text of the option in the box. OnChange runs on typing and on
// choosing. Bind binds the text of a combo made Editable before it is
// bound, and the Value of one bound before.
func (c *ComboBoxWidget) Editable() *ComboBoxWidget {
	c.change(func() {
		if c.editable {
			return
		}
		c.editable, c.bindsText = true, c.bound.write == nil
		if o, ok := c.find(c.value.value); ok {
			c.text.set(o.Text)
		}
	})
	return c
}

// byText returns the option whose text is s. Under mu.
func (c *ComboBoxWidget) byText(s string) (Option, bool) {
	for _, sec := range c.sections {
		for _, o := range sec.opts {
			if o.Text == s {
				return o, true
			}
		}
	}
	return Option{}, false
}

// chosen returns the option the combo shows as chosen. Under mu.
func (c *ComboBoxWidget) chosen() (Option, bool) {
	if c.editable {
		if c.text.value == "" {
			return Option{}, false
		}
		return c.byText(c.text.value)
	}
	return c.find(c.value.value)
}

// Value returns the chosen option's value, "" for none.
func (c *ComboBoxWidget) Value() string {
	mu.Lock()
	defer mu.Unlock()
	o, _ := c.chosen()
	return o.Value
}

// Text returns the text shown: the chosen option's, or, in an Editable
// combo, what the user typed or SetText set.
func (c *ComboBoxWidget) Text() string {
	mu.Lock()
	defer mu.Unlock()
	return c.textOf()
}

// textOf is Text. Under mu.
func (c *ComboBoxWidget) textOf() string {
	if c.editable {
		return c.text.value
	}
	o, _ := c.find(c.value.value)
	return o.Text
}

// SetText sets the text of an Editable combo, which need not be an
// option's; of a plain one it chooses the option with that text, or none.
func (c *ComboBoxWidget) SetText(s string) *ComboBoxWidget {
	c.change(func() {
		if c.editable {
			c.text.set(s)
			return
		}
		o, _ := c.byText(s)
		c.value.set(o.Value)
	})
	return c
}

// Selected returns the chosen option.
func (c *ComboBoxWidget) Selected() (Option, bool) {
	mu.Lock()
	defer mu.Unlock()
	return c.chosen()
}

// SetValue chooses the option of value v; an unknown v chooses none.
func (c *ComboBoxWidget) SetValue(v string) *ComboBoxWidget {
	c.change(func() {
		o, ok := c.find(v)
		if !ok {
			v = ""
		}
		if c.editable {
			c.text.set(o.Text)
			return
		}
		c.value.set(v)
	})
	return c
}

func (c *ComboBoxWidget) Unselect() *ComboBoxWidget {
	c.change(func() {
		if c.editable {
			c.text.set("")
			return
		}
		c.value.set("")
	})
	return c
}

// SetOptions replaces the options outside any section.
func (c *ComboBoxWidget) SetOptions(opts ...Option) *ComboBoxWidget {
	opts = slices.Clone(opts) // the caller's slice may back opts
	c.change(func() {
		c.edit(func(s []comboSection) []comboSection { s[0].opts = opts; return s })
	})
	return c
}

// AddSection adds a section titled title, with the icon in the bytes of
// a PNG (nil for none), holding opts.
func (c *ComboBoxWidget) AddSection(title string, icon []byte, opts ...Option) *ComboBoxWidget {
	bmp, opts := decodeImage(icon), slices.Clone(opts) // outside mu
	c.change(func() {
		c.edit(func(s []comboSection) []comboSection { return append(s, comboSection{title, bmp, opts}) })
	})
	return c
}

// RemoveSection removes the section titled title and its options.
func (c *ComboBoxWidget) RemoveSection(title string) *ComboBoxWidget {
	c.change(func() {
		c.edit(func(s []comboSection) []comboSection {
			return slices.DeleteFunc(s, func(x comboSection) bool { return x.title != "" && x.title == title })
		})
	})
	return c
}

// RemoveOption removes the option of value v.
func (c *ComboBoxWidget) RemoveOption(v string) *ComboBoxWidget {
	c.change(func() {
		c.edit(func(s []comboSection) []comboSection {
			for i := range s {
				s[i].opts = slices.DeleteFunc(s[i].opts, func(o Option) bool { return o.Value == v })
			}
			return s
		})
	})
	return c
}

// SetOptionText changes the text of the option of value v.
func (c *ComboBoxWidget) SetOptionText(v, text string) *ComboBoxWidget {
	c.change(func() {
		c.edit(func(s []comboSection) []comboSection {
			for i := range s {
				for j := range s[i].opts {
					if s[i].opts[j].Value == v {
						s[i].opts[j].Text = text
					}
				}
			}
			return s
		})
	})
	return c
}

// SectionLen returns how many options the section titled title holds.
func (c *ComboBoxWidget) SectionLen(title string) int {
	mu.Lock()
	defer mu.Unlock()
	for _, s := range c.sections {
		if s.title == title {
			return len(s.opts)
		}
	}
	return 0
}

func (c *ComboBoxWidget) OnChange(fn func()) *ComboBoxWidget {
	c.change(func() { c.onChange = fn })
	return c
}

func (c *ComboBoxWidget) build(ctx *ui.Context) ui.Element {
	if c.editable {
		return c.buildEditable(ctx)
	}
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
		OnChange(edited(&c.value, &c.bound, c.onChange)).
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

// buildEditable builds an Editable combo: a text input with a chevron in
// the look of the drop-down, whose popup shows the options that match
// what the user typed. Under mu, while building.
func (c *ComboBoxWidget) buildEditable(ctx *ui.Context) ui.Element {
	p, _ := c.text.bind()
	t, sections := ctx.Theme(), c.sections
	frame := ui.Row(ctx).Radius(t.Radius).Border(1, t.Border).Background(t.Surface).
		AlignItems(ui.Center).MinWidth(t.Space(35))
	frame.Children(func() {
		cb := ui.ComboboxBase(ctx, p)
		cb.Input.Grow(1).Padding(t.Space(1.5), t.Space(3)).OnChange(c.typed())
		if h := formAccept; h != nil {
			cb.Input.OnSubmit(h) // Enter clicks the form's accept button
		}
		cb.Popup(func(panel ui.Element) {
			panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Radius+2).
				Background(t.Background).Border(1, t.Border).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18))
			typed := strings.ToLower(strings.TrimSpace(*p))
			for i, s := range sections {
				shown := s.opts
				if cb.Filtering() && typed != "" {
					shown = matchingOptions(s.opts, typed)
				}
				if len(shown) == 0 {
					continue
				}
				if s.title != "" {
					ui.Row(ctx.Key(sectionKey(i))).Gap(t.Space(1.5)).Padding(t.Space(1.5), t.Space(2.5)).
						AlignItems(ui.Center).Children(func() {
						if s.icon != nil {
							ui.Image(ctx, s.icon).Width(16).Height(16)
						}
						ui.Text(ctx, s.title).Bold().TextColor(t.TextMuted)
					})
				}
				for _, o := range shown {
					item := cb.Item(o.Value).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius)
					if item.Highlighted() {
						item.Background(t.Accent).TextColor(t.AccentText)
					}
					item.Children(func() { ui.Text(ctx, o.Text).SingleLine() })
				}
			}
		})
		if v, ok := cb.Chosen(); ok {
			if o, ok := c.find(v); ok {
				c.choose(o)
			}
		}
		ui.Icon(ctx, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted).Margin(0, t.Space(3), 0, 0)
	})
	return frame
}

// matchingOptions returns the options whose text holds query (lower case),
// those that start with it first.
func matchingOptions(opts []Option, query string) []Option {
	var starts, holds []Option
	for _, o := range opts {
		switch l := strings.ToLower(o.Text); {
		case strings.HasPrefix(l, query):
			starts = append(starts, o)
		case strings.Contains(l, query):
			holds = append(holds, o)
		}
	}
	return append(starts, holds...)
}

// typed is ui's OnChange for the text of an Editable combo: the user typed.
func (c *ComboBoxWidget) typed() func() {
	return func() {
		mu.Lock()
		changed := c.text.value != c.text.shown
		c.text.value, c.text.dirty = c.text.shown, false
		c.report(changed)
		mu.Unlock()
	}
}

// choose takes the option the user chose from the popup. Under mu.
func (c *ComboBoxWidget) choose(o Option) {
	changed := c.text.value != o.Text
	c.text.value, c.text.shown, c.text.dirty = o.Text, o.Text, false
	c.report(changed)
}

// report runs OnChange and the binding's write after an edit of the user,
// on the event goroutine. Under mu.
func (c *ComboBoxWidget) report(changed bool) {
	if !changed {
		return
	}
	h, write := c.onChange, c.bound.write
	v := c.text.value
	if !c.bindsText {
		o, _ := c.byText(v)
		v = o.Value
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
	g.change(func() {
		for i, o := range g.opts {
			g.on[i].set(slices.Contains(vs, o.Value))
		}
	})
	return g
}

func (g *TagSelectWidget) OnChange(fn func()) *TagSelectWidget {
	g.change(func() { g.onChange = fn })
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

// ProgressBarWidget shows how far a task is.
type ProgressBarWidget struct {
	common[ProgressBarWidget]
	fraction float64
	text     string
}

func ProgressBar() *ProgressBarWidget {
	p := &ProgressBarWidget{}
	p.init(p)
	return p
}

// SetFraction sets how far the task is, 0 to 1; a negative fraction shows
// a task of unknown length.
func (p *ProgressBarWidget) SetFraction(f float64) *ProgressBarWidget {
	p.change(func() { p.fraction = f })
	return p
}

// SetText sets the text shown under the bar.
func (p *ProgressBarWidget) SetText(s string) *ProgressBarWidget {
	p.change(func() { p.text = s })
	return p
}

func (p *ProgressBarWidget) build(c *ui.Context) ui.Element {
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

func (s *SpinnerWidget) Start() *SpinnerWidget { s.change(func() { s.running = true }); return s }
func (s *SpinnerWidget) Stop() *SpinnerWidget  { s.change(func() { s.running = false }); return s }

func (s *SpinnerWidget) build(c *ui.Context) ui.Element {
	if s.running {
		return ui.Spinner(c)
	}
	return ui.Box(c)
}

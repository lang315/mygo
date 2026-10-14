package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// Option is a choice of a group or a combo: Value is what the program
// reads, Text what the user sees.
type Option struct{ Value, Text string }

// ButtonWidget is a push button.
type ButtonWidget struct {
	common[ButtonWidget]
	text                   string
	icon                   *ui.SVG
	suggested, destructive bool
	onClick                func()
}

// Button makes a button showing text.
func Button(text string) *ButtonWidget {
	b := &ButtonWidget{text: text}
	b.init(b)
	return b
}

func (b *ButtonWidget) OnClick(fn func()) *ButtonWidget { change(func() { b.onClick = fn }); return b }
func (b *ButtonWidget) SetText(s string) *ButtonWidget  { change(func() { b.text = s }); return b }

// SetIcon shows an icon before the text; nil removes it.
func (b *ButtonWidget) SetIcon(s *ui.SVG) *ButtonWidget { change(func() { b.icon = s }); return b }

// Suggested draws the button as the main action, gota's suggested-action.
func (b *ButtonWidget) Suggested() *ButtonWidget { change(func() { b.suggested = true }); return b }

// Destructive draws the button as an action that destroys, as Delete.
func (b *ButtonWidget) Destructive() *ButtonWidget {
	change(func() { b.destructive = true })
	return b
}

func (b *ButtonWidget) build(c *ui.Context) ui.Element {
	label := b.text
	if b.icon != nil {
		label = ""
	}
	var e ui.Element
	if b.suggested {
		e = ui.PrimaryButton(c, label)
	} else {
		e = ui.Button(c, label)
	}
	if b.destructive {
		t := c.Theme()
		e.Background(t.Danger).TextColor(t.AccentText)
	}
	if b.icon != nil {
		icon, text := b.icon, b.text
		e.Children(func() {
			ui.Row(c).Gap(c.Theme().Space(1.5)).Children(func() {
				ui.Icon(c, icon)
				if text != "" {
					ui.Text(c, text)
				}
			})
		})
	}
	if h := b.onClick; h != nil {
		e.OnClick(func() { Post(h) })
	}
	return e
}

// LinkWidget opens a web address when clicked.
type LinkWidget struct {
	common[LinkWidget]
	text, url string
}

// Link makes a link showing text that opens url.
func Link(text, url string) *LinkWidget {
	l := &LinkWidget{text: text, url: url}
	l.init(l)
	return l
}

func (l *LinkWidget) build(c *ui.Context) ui.Element { return ui.Link(c, l.text, l.url) }

// ToggleWidget is a button that stays pressed.
type ToggleWidget struct {
	common[ToggleWidget]
	text     string
	on       mirror[bool]
	onChange func()
}

func Toggle(text string) *ToggleWidget {
	t := &ToggleWidget{text: text}
	t.init(t)
	return t
}

func (t *ToggleWidget) Active() bool                   { mu.Lock(); defer mu.Unlock(); return t.on.value }
func (t *ToggleWidget) SetActive(v bool) *ToggleWidget { change(func() { t.on.set(v) }); return t }
func (t *ToggleWidget) OnChange(fn func()) *ToggleWidget {
	change(func() { t.onChange = fn })
	return t
}

func (t *ToggleWidget) build(c *ui.Context) ui.Element {
	p, changed := t.on.bind()
	e := ui.Toggle(c, p, t.text).OnChange(onChange(&t.on, t.onChange))
	if changed && t.onChange != nil {
		Post(t.onChange)
	}
	return e
}

// CheckboxWidget is a check box with a label.
type CheckboxWidget struct {
	common[CheckboxWidget]
	text     string
	on       mirror[bool]
	onChange func()
}

func Checkbox(text string) *CheckboxWidget {
	b := &CheckboxWidget{text: text}
	b.init(b)
	return b
}

func (b *CheckboxWidget) Checked() bool { mu.Lock(); defer mu.Unlock(); return b.on.value }
func (b *CheckboxWidget) SetChecked(v bool) *CheckboxWidget {
	change(func() { b.on.set(v) })
	return b
}
func (b *CheckboxWidget) OnChange(fn func()) *CheckboxWidget {
	change(func() { b.onChange = fn })
	return b
}

func (b *CheckboxWidget) build(c *ui.Context) ui.Element {
	p, changed := b.on.bind()
	e := ui.Checkbox(c, p, b.text).OnChange(onChange(&b.on, b.onChange))
	if changed && b.onChange != nil {
		Post(b.onChange)
	}
	return e
}

// SwitchWidget is an on/off switch.
type SwitchWidget struct {
	common[SwitchWidget]
	on       mirror[bool]
	onChange func()
}

func Switch() *SwitchWidget {
	s := &SwitchWidget{}
	s.init(s)
	return s
}

func (s *SwitchWidget) On() bool                   { mu.Lock(); defer mu.Unlock(); return s.on.value }
func (s *SwitchWidget) SetOn(v bool) *SwitchWidget { change(func() { s.on.set(v) }); return s }
func (s *SwitchWidget) OnChange(fn func()) *SwitchWidget {
	change(func() { s.onChange = fn })
	return s
}

func (s *SwitchWidget) build(c *ui.Context) ui.Element {
	p, changed := s.on.bind()
	e := ui.Switch(c, p).OnChange(onChange(&s.on, s.onChange))
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	return e
}

// CheckboxGroupWidget is a check box per option.
type CheckboxGroupWidget struct {
	common[CheckboxGroupWidget]
	dir      direction
	opts     []Option
	on       []mirror[bool]
	onChange func()
}

// CheckboxGroup makes a check box per option, in a row or a column.
func CheckboxGroup(row bool, opts ...Option) *CheckboxGroupWidget {
	g := &CheckboxGroupWidget{opts: opts, on: make([]mirror[bool], len(opts))}
	if row {
		g.dir = horizontal
	}
	g.init(g)
	return g
}

// Values returns the values of the checked options, in option order.
func (g *CheckboxGroupWidget) Values() []string {
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

// SetValues checks the options of vs and unchecks the others.
func (g *CheckboxGroupWidget) SetValues(vs ...string) *CheckboxGroupWidget {
	change(func() {
		for i, o := range g.opts {
			g.on[i].set(slices.Contains(vs, o.Value))
		}
	})
	return g
}

func (g *CheckboxGroupWidget) OnChange(fn func()) *CheckboxGroupWidget {
	change(func() { g.onChange = fn })
	return g
}

func (g *CheckboxGroupWidget) build(c *ui.Context) ui.Element {
	e := ui.Column(c)
	if g.dir == horizontal {
		e = ui.Row(c)
	}
	return e.Gap(c.Theme().Space(3)).Children(func() {
		for i, o := range g.opts {
			p, changed := g.on[i].bind()
			ui.Checkbox(c.Key(i), p, o.Text).OnChange(onChange(&g.on[i], g.onChange))
			if changed && g.onChange != nil {
				Post(g.onChange)
			}
		}
	})
}

// RadioGroupWidget is a choice of one option.
type RadioGroupWidget struct {
	common[RadioGroupWidget]
	dir      direction
	opts     []Option
	value    mirror[string]
	onChange func()
}

// RadioGroup makes a radio button per option, in a row or a column, the
// first chosen.
func RadioGroup(row bool, opts ...Option) *RadioGroupWidget {
	g := &RadioGroupWidget{opts: opts}
	if row {
		g.dir = horizontal
	}
	if len(opts) > 0 {
		g.value.set(opts[0].Value)
	}
	g.init(g)
	return g
}

func (g *RadioGroupWidget) Value() string { mu.Lock(); defer mu.Unlock(); return g.value.value }
func (g *RadioGroupWidget) SetValue(v string) *RadioGroupWidget {
	change(func() { g.value.set(v) })
	return g
}
func (g *RadioGroupWidget) OnChange(fn func()) *RadioGroupWidget {
	change(func() { g.onChange = fn })
	return g
}

func (g *RadioGroupWidget) build(c *ui.Context) ui.Element {
	p, changed := g.value.bind()
	if changed && g.onChange != nil {
		Post(g.onChange)
	}
	cb := onChange(&g.value, g.onChange)
	e := ui.RadioGroup(c, func() {
		for _, o := range g.opts {
			ui.Radio(c, p, o.Value, o.Text).OnChange(cb)
		}
	})
	if g.dir == horizontal {
		e.Row().Gap(c.Theme().Space(3))
	}
	return e
}

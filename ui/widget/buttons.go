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
	text    string
	icon    *ui.SVG
	onClick func()
	clicked func() // onClicked, made once
}

// Button makes a button showing text.
func Button(text string) *ButtonWidget {
	b := &ButtonWidget{text: text}
	b.init(b)
	return b
}

func (b *ButtonWidget) OnClick(fn func()) *ButtonWidget {
	b.change(func() { b.onClick = fn })
	return b
}
func (b *ButtonWidget) SetText(s string) *ButtonWidget { b.change(func() { b.text = s }); return b }

// SetIcon shows an icon before the text; nil removes it.
func (b *ButtonWidget) SetIcon(s *ui.SVG) *ButtonWidget { b.change(func() { b.icon = s }); return b }

// Suggested draws the button as the main action, gota's suggested-action
// and tw's: the accent for its background, with the accent's text, and the
// accent's hover and pressed roles. It is the button's own style, which
// wins over applied ones.
func (b *ButtonWidget) Suggested() *ButtonWidget { return b.own(suggestedStyle) }

// suggestedStyle is the look of Suggested, of buttons and tool buttons.
func suggestedStyle(s Style) Style {
	s = s.Background(RoleAccent).TextColor(RoleAccentText)
	s = withMerged(s, vHover, NewStyle().Background(RoleAccentHover))
	return withMerged(s, vPressed, NewStyle().Background(RoleAccentPressed))
}

// Destructive draws the button as an action that destroys, as Delete,
// gota's destructive-action and tw's: the danger color for its background,
// with the accent's text, fading to 0.9 opacity when hovered and 0.8
// pressed. It is the button's own style.
func (b *ButtonWidget) Destructive() *ButtonWidget { return b.own(destructiveStyle) }

// destructiveStyle is the look of Destructive, of buttons and tool buttons.
func destructiveStyle(s Style) Style {
	s = s.Background(RoleDanger).TextColor(RoleAccentText)
	s = withMerged(s, vHover, NewStyle().Opacity(0.9))
	return withMerged(s, vPressed, NewStyle().Opacity(0.8))
}

// withMerged sets variant k of s to v over the one s has, so a hover or
// pressed look set before is kept where v sets nothing.
func withMerged(s Style, k variant, v Style) Style {
	if old := s.v[k]; old != nil {
		v = old.Merge(v)
	}
	return s.with(k, v)
}

func (b *ButtonWidget) build(c *ui.Context) ui.Element {
	label, key, at := parseMnemonic(b.text)
	// The underline shows while Alt is held, as Windows and GTK show access
	// keys, and only where the key is answered (a form is building it).
	underlined := key != 0 && b.icon == nil && !macMenus && buildingForm != nil && c.Modifiers()&ui.Alt != 0
	shown := label
	if b.icon != nil || underlined {
		shown = ""
	}
	e := ui.Button(c, shown)
	if underlined {
		e.Label(label).Children(func() { ui.RichText(c, mnemonicSpans(label, at)...) })
	}
	if h := b.onClick; h != nil && b.icon == nil {
		if key != 0 { // the closure is made only for a mnemonic: it would be allocated in each build
			registerMnemonic(key, e, b, func() { Post(h) })
		}
	}
	if b.icon != nil {
		text, _, _ := parseMnemonic(b.text)
		icon := b.icon
		e.Children(func() {
			ui.Row(c).Gap(c.Theme().Space(1.5)).Children(func() {
				ui.Icon(c, icon)
				if text != "" {
					ui.Text(c, text)
				}
			})
		})
	}
	if b.onClick != nil {
		if b.clicked == nil {
			b.clicked = b.onClicked // once: a closure made in each build is an allocation each
		}
		e.OnClick(b.clicked)
	}
	return e
}

// onClicked is ui's OnClick callback: it runs the OnClick of the moment.
func (b *ButtonWidget) onClicked() {
	mu.Lock()
	h := b.onClick
	mu.Unlock()
	if h != nil {
		Post(h)
	}
}

// LinkLabelWidget opens a web address when clicked, or runs its OnClick.
type LinkLabelWidget struct {
	common[LinkLabelWidget]
	text, url string
	onClick   func()
	clicked   func() // onClicked, made once
}

// LinkLabel makes a link showing text that opens url.
func LinkLabel(text, url string) *LinkLabelWidget {
	l := &LinkLabelWidget{text: text, url: url}
	l.init(l)
	return l
}

// OnClick runs fn on the event goroutine when the link is clicked, or
// activated with Enter or Space while it has the focus or by assistive
// technology, in place of opening its address (as WinForms' LinkClicked: fn
// decides what the link does). A nil fn gives back the opening. A disabled
// link does neither.
func (l *LinkLabelWidget) OnClick(fn func()) *LinkLabelWidget {
	l.change(func() { l.onClick = fn })
	return l
}

func (l *LinkLabelWidget) onClicked() {
	mu.Lock()
	h := l.onClick
	mu.Unlock()
	if h != nil {
		Post(h)
	}
}

func (l *LinkLabelWidget) build(c *ui.Context) ui.Element {
	if l.onClick == nil {
		return ui.Link(c, l.text, l.url)
	}
	e := ui.Link(c, l.text, "") // no address: ui opens nothing
	if l.clicked == nil {
		l.clicked = l.onClicked // once: a closure made in each build is an allocation each
	}
	return e.OnClick(l.clicked)
}

// ToggleWidget is a button that stays pressed.
type ToggleWidget struct {
	common[ToggleWidget]
	text     string
	on       mirror[bool]
	onChange func()
	toggled  func() // onToggled, made once
}

func Toggle(text string) *ToggleWidget {
	t := &ToggleWidget{text: text}
	t.init(t)
	return t
}

func (t *ToggleWidget) Active() bool                   { mu.Lock(); defer mu.Unlock(); return t.on.value }
func (t *ToggleWidget) SetActive(v bool) *ToggleWidget { t.change(func() { t.on.set(v) }); return t }
func (t *ToggleWidget) OnChange(fn func()) *ToggleWidget {
	t.change(func() { t.onChange = fn })
	return t
}

// onToggled is the callback of ui's OnChange, made once (see TextBoxWidget).
func (t *ToggleWidget) onToggled() {
	mu.Lock()
	fn := onChange(&t.on, t.onChange)
	mu.Unlock()
	fn()
}

func (t *ToggleWidget) build(c *ui.Context) ui.Element {
	p, changed := t.on.bind()
	if t.toggled == nil {
		t.toggled = t.onToggled
	}
	e := ui.Toggle(c, p, t.text).OnChange(t.toggled)
	if changed && t.onChange != nil {
		Post(t.onChange)
	}
	return e
}

// CheckBoxWidget is a check box with a label.
type CheckBoxWidget struct {
	common[CheckBoxWidget]
	text     string
	on       mirror[bool]
	onChange func()
	edit     func() // editedOnce's
}

func CheckBox(text string) *CheckBoxWidget {
	b := &CheckBoxWidget{text: text}
	b.init(b)
	return b
}

func (b *CheckBoxWidget) Checked() bool { mu.Lock(); defer mu.Unlock(); return b.on.value }
func (b *CheckBoxWidget) SetChecked(v bool) *CheckBoxWidget {
	b.change(func() { b.on.set(v) })
	return b
}
func (b *CheckBoxWidget) OnChange(fn func()) *CheckBoxWidget {
	b.change(func() { b.onChange = fn })
	return b
}

func (b *CheckBoxWidget) build(c *ui.Context) ui.Element {
	p, changed := b.on.bind()
	label, key, _ := parseMnemonic(b.text)
	e := ui.Checkbox(c, p, label).OnChange(editedOnce(&b.edit, &b.on, &b.bound, &b.onChange))
	if changed && b.onChange != nil {
		Post(b.onChange)
	}
	if key != 0 {
		registerMnemonic(key, e, b, func() {
			focusLater(b)
			toggle := edited(&b.on, &b.bound, b.onChange)
			Post(func() {
				mu.Lock()
				b.on.shown = !b.on.shown
				mu.Unlock()
				toggle()
			})
		})
	}
	return e
}

// SwitchWidget is an on/off switch.
type SwitchWidget struct {
	common[SwitchWidget]
	on       mirror[bool]
	onChange func()
	toggled  func() // onToggled, made once
}

func Switch() *SwitchWidget {
	s := &SwitchWidget{}
	s.init(s)
	return s
}

func (s *SwitchWidget) On() bool                   { mu.Lock(); defer mu.Unlock(); return s.on.value }
func (s *SwitchWidget) SetOn(v bool) *SwitchWidget { s.change(func() { s.on.set(v) }); return s }
func (s *SwitchWidget) OnChange(fn func()) *SwitchWidget {
	s.change(func() { s.onChange = fn })
	return s
}

// onToggled is the callback of ui's OnChange, made once (see TextBoxWidget).
func (s *SwitchWidget) onToggled() {
	mu.Lock()
	fn := onChange(&s.on, s.onChange)
	mu.Unlock()
	fn()
}

func (s *SwitchWidget) build(c *ui.Context) ui.Element {
	p, changed := s.on.bind()
	if s.toggled == nil {
		s.toggled = s.onToggled
	}
	e := ui.Switch(c, p).OnChange(s.toggled)
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	return e
}

// CheckBoxGroupWidget is a check box per option.
type CheckBoxGroupWidget struct {
	common[CheckBoxGroupWidget]
	dir      direction
	opts     []Option
	on       []mirror[bool]
	onChange func()
}

// CheckBoxGroup makes a check box per option, in a row or a column.
func CheckBoxGroup(row bool, opts ...Option) *CheckBoxGroupWidget {
	g := &CheckBoxGroupWidget{opts: opts, on: make([]mirror[bool], len(opts))}
	if row {
		g.dir = horizontal
	}
	g.init(g)
	return g
}

// Values returns the values of the checked options, in option order.
func (g *CheckBoxGroupWidget) Values() []string {
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
func (g *CheckBoxGroupWidget) SetValues(vs ...string) *CheckBoxGroupWidget {
	g.change(func() {
		for i, o := range g.opts {
			g.on[i].set(slices.Contains(vs, o.Value))
		}
	})
	return g
}

func (g *CheckBoxGroupWidget) OnChange(fn func()) *CheckBoxGroupWidget {
	g.change(func() { g.onChange = fn })
	return g
}

func (g *CheckBoxGroupWidget) build(c *ui.Context) ui.Element {
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
	g.change(func() { g.value.set(v) })
	return g
}
func (g *RadioGroupWidget) OnChange(fn func()) *RadioGroupWidget {
	g.change(func() { g.onChange = fn })
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

// RadioButtonWidget is a radio button of its own. As in WinForms, the radio
// buttons added to the same container (a box, a group box, fields, a flow
// panel) are one group: checking one, by the user or by SetChecked,
// unchecks the others. One outside a container stands alone.
type RadioButtonWidget struct {
	common[RadioButtonWidget]
	text     string
	on       mirror[string] // radioOn when checked
	onChange func()
	group    *radioGroup // of the container it is in
}

// radioOn is the value of a checked radio button's ui state.
const radioOn = "on"

// radioGroup is the radio buttons of a container. Under mu.
type radioGroup struct{ members []*RadioButtonWidget }

// RadioButton makes an unchecked radio button showing text.
func RadioButton(text string) *RadioButtonWidget {
	r := &RadioButtonWidget{text: text}
	r.init(r)
	return r
}

// adopt makes w, if it is a radio button, one of parent's group; orphan
// takes it out. Under mu.
func adopt(parent *core, w Control) {
	if r, ok := w.(*RadioButtonWidget); ok {
		orphan(w)
		if parent.group == nil {
			parent.group = &radioGroup{}
		}
		r.group = parent.group
		r.group.members = append(r.group.members, r)
		if r.on.value == radioOn {
			r.uncheckOthers() // the last one added checked wins
		}
	}
}

func orphan(w Control) {
	if r, ok := w.(*RadioButtonWidget); ok && r.group != nil {
		r.group.members = slices.DeleteFunc(r.group.members, func(m *RadioButtonWidget) bool { return m == r })
		r.group = nil
	}
}

// Checked reports whether the radio button is checked.
func (r *RadioButtonWidget) Checked() bool {
	mu.Lock()
	defer mu.Unlock()
	return r.on.value == radioOn
}

// SetChecked checks the radio button, which unchecks the others of its
// group, or unchecks it. It does not run OnChange.
func (r *RadioButtonWidget) SetChecked(v bool) *RadioButtonWidget {
	r.change(func() {
		if v {
			r.uncheckOthers()
			r.on.set(radioOn)
		} else {
			r.on.set("")
		}
	})
	return r
}

// uncheckOthers unchecks the other radio buttons of the group, and returns
// those that were checked. Under mu.
func (r *RadioButtonWidget) uncheckOthers() (was []*RadioButtonWidget) {
	if r.group == nil {
		return nil
	}
	for _, m := range r.group.members {
		if m != r && m.on.value == radioOn {
			m.on.set("")
			was = append(was, m)
		}
	}
	return was
}

// OnChange runs fn after the user checks the radio button, or another of
// its group unchecks it.
func (r *RadioButtonWidget) OnChange(fn func()) *RadioButtonWidget {
	r.change(func() { r.onChange = fn })
	return r
}

func (r *RadioButtonWidget) build(c *ui.Context) ui.Element {
	p, _ := r.on.bind()
	label, key, _ := parseMnemonic(r.text)
	e := ui.Radio(c, p, radioOn, label).OnChange(r.edited)
	if key != 0 {
		registerMnemonic(key, e, r, func() {
			focusLater(r)
			Post(func() {
				mu.Lock()
				r.on.shown = radioOn
				mu.Unlock()
				r.edited()
			})
		})
	}
	return e
}

// edited is ui's OnChange: the user checked the radio button, which
// unchecks the others, and both sides hear of it.
func (r *RadioButtonWidget) edited() {
	mu.Lock()
	changed := r.on.value != r.on.shown
	r.on.value, r.on.dirty = r.on.shown, false
	var hear []func()
	if changed {
		if r.onChange != nil {
			hear = append(hear, r.onChange)
		}
		if r.on.value == radioOn {
			for _, m := range r.uncheckOthers() {
				if m.onChange != nil {
					hear = append(hear, m.onChange)
				}
			}
		}
	}
	mu.Unlock()
	for _, h := range hear {
		Post(h)
	}
}

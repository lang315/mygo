package widget

import "github.com/egoist/mygo/ui"

// AccordionWidget is sections one above the other, each a header over its
// controls, as DevExpress's AccordionControl: one section is open at a
// time, a click on a header opens it, and a click on the open one closes
// it. A closed section's controls are not built. It is ui.Accordion
// underneath.
type AccordionWidget struct {
	edit func() // editedOnce's
	common[AccordionWidget]
	sections []*accordionSection
	open     mirror[int]
	onChange func()
	ctx      *ui.Context // of the build, for fill and the sections' bodies
	fill     func()      // builds the sections, made once
}

// accordionSection is a section of an accordion.
type accordionSection struct {
	title  string
	body   *BoxWidget
	isOpen bool   // ui's, which only the drawing thread touches
	build  func() // builds body, made once
	toggle func() // ui's OnChange callback, made once
}

// Accordion makes an accordion with no section, none open.
func Accordion() *AccordionWidget {
	a := &AccordionWidget{}
	a.open.set(-1)
	a.init(a)
	a.fill = a.buildSections
	return a
}

// AddSection adds a section titled title, holding controls one above the
// other, as a VBox does: its radio buttons are one group.
func (a *AccordionWidget) AddSection(title string, controls ...Control) *AccordionWidget {
	s := &accordionSection{title: title, body: VBox(8).Add(controls...)}
	s.build = func() { render(a.ctx, s.body) }
	a.change(func() {
		i := len(a.sections)
		s.toggle = func() {
			// The drawing thread, as ui's callbacks: shown is its.
			if s.isOpen {
				a.open.shown = i
			} else if a.open.shown == i {
				a.open.shown = -1
			}
			a.edit()
		}
		a.sections = append(a.sections, s)
	})
	return a
}

// Open returns the index of the open section, in the order added, or -1
// for none.
func (a *AccordionWidget) Open() int { mu.Lock(); defer mu.Unlock(); return a.open.value }

// SetOpen opens section i, closing the others, or closes them all for -1,
// without running OnChange. Any other index out of range is ignored.
func (a *AccordionWidget) SetOpen(i int) *AccordionWidget {
	a.change(func() {
		if i >= -1 && i < len(a.sections) {
			a.open.set(i)
		}
	})
	return a
}

// OnChange runs fn, on the event goroutine, after the user opened or
// closed a section. A setter does not run it.
func (a *AccordionWidget) OnChange(fn func()) *AccordionWidget {
	a.change(func() { a.onChange = fn })
	return a
}

func (a *AccordionWidget) build(c *ui.Context) ui.Element {
	p, changed := a.open.bind()
	if changed && a.onChange != nil {
		Post(a.onChange)
	}
	editedOnce(&a.edit, &a.open, &a.bound, &a.onChange)
	for i, s := range a.sections {
		s.isOpen = *p == i
	}
	a.ctx = c
	return ui.Accordion(c, a.fill)
}

// buildSections builds the sections, under mu while the accordion builds.
func (a *AccordionWidget) buildSections() {
	outer := axis
	axis = vertical
	for _, s := range a.sections {
		ui.AccordionItem(a.ctx, s.title, &s.isOpen, s.build).OnChange(s.toggle)
	}
	axis = outer
}

func (a *AccordionWidget) children() []Control {
	out := make([]Control, len(a.sections))
	for i, s := range a.sections {
		out[i] = s.body
	}
	return out
}

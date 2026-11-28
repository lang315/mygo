package widget

import (
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// WizardWidget walks the user through steps, one page at a time, as
// DevExpress's WizardControl: a StepProgressBar at the top, the current
// step's page in the middle, and Previous, Next (Done on the last step) and
// Cancel at the bottom. Enter is Next, or Done, and Escape is Cancel,
// wherever the focus is in the window, unless the control with the focus
// takes the key (a focused button clicks itself); the wizard takes them
// before the form's accept and cancel buttons, and Escape no longer closes
// a dialog showing it.
type WizardWidget struct {
	common[WizardWidget]
	bar                  *StepProgressBarWidget // its step is the wizard's
	pages                []Control
	prev, next, cancel   *ButtonWidget
	onValidate           func(step int) bool
	onStepChange         func()
	onFinish, onCancel   func()
	forward, quit, enter func() // made once: goNext, goCancel, and goNext posted
}

// Wizard makes a wizard with no steps: AddStep adds them.
func Wizard() *WizardWidget {
	w := &WizardWidget{bar: StepProgressBar()}
	w.init(w)
	w.forward, w.quit = w.goNext, w.goCancel
	w.enter = func() { Post(w.forward) }
	w.prev = Button(l10n.Previous()).OnClick(w.goBack)
	w.next = Button(l10n.Next()).OnClick(w.forward).Suggested()
	w.cancel = Button(l10n.Cancel()).OnClick(w.quit)
	return w
}

// AddStep adds a step titled title, which shows content.
func (w *WizardWidget) AddStep(title string, content Control) *WizardWidget {
	w.change(func() {
		w.bar.setSteps(append(w.bar.steps, title))
		w.pages = append(w.pages, content)
		adopt(&w.core, content)
	})
	return w
}

// Step returns the index of the current step.
func (w *WizardWidget) Step() int { mu.Lock(); defer mu.Unlock(); return w.bar.step }

// SetStep shows step i, the nearest step for an index out of range. It runs
// neither OnValidate nor OnStepChange.
func (w *WizardWidget) SetStep(i int) *WizardWidget {
	w.change(func() { w.bar.setStep(i) })
	return w
}

// OnValidate runs fn, on the event goroutine, with the current step when
// the user asks for the next step or for Done; the wizard stays where fn
// returns false. It runs without the package's lock, so it may call
// ErrorProvider.Validate. Previous does not ask it.
func (w *WizardWidget) OnValidate(fn func(step int) bool) *WizardWidget {
	w.change(func() { w.onValidate = fn })
	return w
}

// OnStepChange runs fn after the user moves to another step, with Next or
// Previous; SetStep does not run it.
func (w *WizardWidget) OnStepChange(fn func()) *WizardWidget {
	w.change(func() { w.onStepChange = fn })
	return w
}

// OnFinish runs fn when the user clicks Done, or presses Enter, on the last
// step and OnValidate allows it. The wizard stays on that step: the app
// closes its form.
func (w *WizardWidget) OnFinish(fn func()) *WizardWidget {
	w.change(func() { w.onFinish = fn })
	return w
}

// OnCancel runs fn when the user clicks Cancel or presses Escape.
func (w *WizardWidget) OnCancel(fn func()) *WizardWidget {
	w.change(func() { w.onCancel = fn })
	return w
}

// goNext is Next and Done, on the event goroutine: it asks OnValidate, then
// moves on, or finishes on the last step.
func (w *WizardWidget) goNext() {
	mu.Lock()
	i, n, valid := w.bar.step, len(w.pages), w.onValidate
	mu.Unlock()
	if n == 0 || valid != nil && !valid(i) {
		return
	}
	if i == n-1 {
		w.post(&w.onFinish)
		return
	}
	moved := false
	w.change(func() {
		if w.bar.step == i { // else the app moved it while OnValidate ran
			w.bar.setStep(i + 1)
			moved = true
		}
	})
	if moved {
		w.post(&w.onStepChange)
	}
}

// goBack is Previous, on the event goroutine.
func (w *WizardWidget) goBack() {
	moved := false
	w.change(func() {
		if w.bar.step > 0 {
			w.bar.setStep(w.bar.step - 1)
			moved = true
		}
	})
	if moved {
		w.post(&w.onStepChange)
	}
}

// goCancel is Cancel, on the event goroutine.
func (w *WizardWidget) goCancel() { w.post(&w.onCancel) }

// post posts the callback *fn holds, if any.
func (w *WizardWidget) post(fn *func()) {
	mu.Lock()
	h := *fn
	mu.Unlock()
	if h != nil {
		Post(h)
	}
}

func (w *WizardWidget) build(c *ui.Context) ui.Element {
	n, i := len(w.pages), w.bar.step
	last := i >= n-1
	// The buttons' fields, not their setters: build holds mu.
	w.prev.text, w.cancel.text = l10n.Previous(), l10n.Cancel()
	w.next.text = l10n.Next()
	if last {
		w.next.text = l10n.Done()
	}
	w.prev.disabled, w.next.disabled = i == 0, n == 0
	if !w.disabled {
		if c.Shortcut(0, ui.KeyEnter) {
			Post(w.forward)
		}
		if c.Shortcut(0, ui.KeyEscape) {
			Post(w.quit)
		}
	}
	t := c.Theme()
	return ui.Column(c).Gap(t.Space(3)).Children(func() {
		render(c, w.bar)
		if i < n {
			// Enter in a text field of the page, which takes it, is Next too.
			outer := formAccept
			formAccept = w.enter
			buildPage(c, w.pages[i])
			formAccept = outer
		} else {
			ui.Box(c).Grow(1)
		}
		ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
			render(c, w.prev)
			render(c, w.next)
			render(c, w.cancel)
		})
	})
}

func (w *WizardWidget) fillsAcross() bool { return true }

func (w *WizardWidget) children() []Control {
	return append([]Control{w.bar, w.prev, w.next, w.cancel}, w.pages...)
}

package widget

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// wizardCounts counts a wizard's callbacks. The handlers run on the event
// goroutine, which Settle waits for.
type wizardCounts struct {
	changes, finishes, cancels int
	validated                  []int
	valid                      bool
}

// threeSteps is a wizard of three steps whose callbacks count into n.
func threeSteps(n *wizardCounts) *WizardWidget {
	return Wizard().
		AddStep("Account", Label("Step one")).
		AddStep("Plan", Label("Step two")).
		AddStep("Confirm", Label("Step three")).
		OnStepChange(func() { n.changes++ }).
		OnFinish(func() { n.finishes++ }).
		OnCancel(func() { n.cancels++ })
}

func wizardTester(t *testing.T, w *WizardWidget) *Tester {
	t.Helper()
	tt := NewTester(t, w, 600, 400)
	tt.Settle()
	return tt
}

func wizButton(tt *Tester, label string) *Node {
	return tt.Find(ByRole(ui.RoleButton).Label(label))
}

func hasButton(tt *Tester, label string) bool {
	return tt.Has(ByRole(ui.RoleButton).Label(label))
}

// Case 5: Next moves 0→1 and fires OnStepChange once; Previous goes back;
// Previous is disabled on step 0; the last step's button says Done.
func TestWizardNavigation(t *testing.T) {
	var n wizardCounts
	w := threeSteps(&n)
	tt := wizardTester(t, w)
	if !tt.Has(ByLabel("Step one")) {
		t.Fatalf("step 0 shows %q", tt.Texts())
	}
	for _, s := range []string{"Account", "Plan", "Confirm"} {
		if !tt.Has(ByLabel(s)) {
			t.Errorf("the step bar lacks %q: %q", s, tt.Texts())
		}
	}
	if !wizButton(tt, l10n.Previous()).Disabled() {
		t.Errorf("Previous is enabled on step 0")
	}
	if !hasButton(tt, l10n.Cancel()) {
		t.Errorf("no Cancel button")
	}
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 1 || n.changes != 1 {
		t.Fatalf("after Next: step %d, %d changes; want 1 and 1", got, n.changes)
	}
	if !tt.Has(ByLabel("Step two")) || tt.Has(ByLabel("Step one")) {
		t.Errorf("step 1 shows %q", tt.Texts())
	}
	if wizButton(tt, l10n.Previous()).Disabled() {
		t.Errorf("Previous is disabled on step 1")
	}
	wizButton(tt, l10n.Previous()).Click()
	if got := w.Step(); got != 0 || n.changes != 2 {
		t.Fatalf("after Previous: step %d, %d changes; want 0 and 2", got, n.changes)
	}
	wizButton(tt, l10n.Next()).Click()
	if hasButton(tt, l10n.Done()) {
		t.Errorf("step 1 of 3 shows Done")
	}
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 2 {
		t.Fatalf("step %d, want 2", got)
	}
	if !hasButton(tt, l10n.Done()) || hasButton(tt, l10n.Next()) {
		t.Errorf("the last step's buttons: Done %v, Next %v; want Done alone", hasButton(tt, l10n.Done()), hasButton(tt, l10n.Next()))
	}
	if n.finishes != 0 {
		t.Errorf("reaching the last step finished")
	}
}

// Case 6: one step is the first and the last: Previous disabled, Done. No
// steps: nothing to finish, Next disabled, Enter does nothing.
func TestWizardOneAndNoSteps(t *testing.T) {
	var n wizardCounts
	one := Wizard().AddStep("Only", Label("The only step")).OnFinish(func() { n.finishes++ })
	tt := wizardTester(t, one)
	if !wizButton(tt, l10n.Previous()).Disabled() || !hasButton(tt, l10n.Done()) {
		t.Errorf("one step: Previous disabled %v, Done %v; want both", wizButton(tt, l10n.Previous()).Disabled(), hasButton(tt, l10n.Done()))
	}
	wizButton(tt, l10n.Done()).Click()
	if n.finishes != 1 {
		t.Errorf("Done of one step finished %d times, want 1", n.finishes)
	}
	none := Wizard().OnFinish(func() { n.finishes += 10 }).OnStepChange(func() { n.changes++ })
	tt = wizardTester(t, none)
	if !wizButton(tt, l10n.Done()).Disabled() {
		t.Errorf("no steps: Done is enabled")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if n.finishes != 1 || n.changes != 0 || none.Step() != 0 {
		t.Errorf("Enter with no steps: %d finishes, %d changes, step %d", n.finishes, n.changes, none.Step())
	}
}

// Case 7: OnValidate returning false keeps the step, true moves on. It runs
// on Next and on Done, with the step, and not on Previous.
func TestWizardValidate(t *testing.T) {
	var n wizardCounts
	w := threeSteps(&n).OnValidate(func(step int) bool {
		n.validated = append(n.validated, step)
		return n.valid
	})
	tt := wizardTester(t, w)
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 0 || n.changes != 0 {
		t.Fatalf("invalid Next: step %d, %d changes; want 0, 0", got, n.changes)
	}
	n.valid = true
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 1 || n.changes != 1 {
		t.Fatalf("valid Next: step %d, %d changes; want 1, 1", got, n.changes)
	}
	wizButton(tt, l10n.Previous()).Click()
	if want := []int{0, 0}; !slices.Equal(n.validated, want) {
		t.Errorf("validated %v after Previous, want %v: Previous does not validate", n.validated, want)
	}
	w.SetStep(2)
	tt.Settle()
	n.valid = false
	wizButton(tt, l10n.Done()).Click()
	if n.finishes != 0 {
		t.Errorf("an invalid Done finished")
	}
	n.valid = true
	wizButton(tt, l10n.Done()).Click()
	if n.finishes != 1 {
		t.Errorf("a valid Done finished %d times, want 1", n.finishes)
	}
	if want := []int{0, 0, 2, 2}; !slices.Equal(n.validated, want) {
		t.Errorf("validated %v, want %v", n.validated, want)
	}
}

// Case 7b: an OnValidate that moves the step itself (SetStep) wins over
// Next, which then neither moves nor fires OnStepChange.
func TestWizardValidateMovesStep(t *testing.T) {
	var n wizardCounts
	var w *WizardWidget
	w = threeSteps(&n).OnValidate(func(int) bool { w.SetStep(2); return true })
	tt := wizardTester(t, w)
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 2 || n.changes != 0 {
		t.Fatalf("OnValidate moved to 2: step %d, %d changes; want 2, 0", got, n.changes)
	}
}

// Case 8: OnValidate runs on the event goroutine, without the package's
// lock, so it can call ErrorProvider.Validate on the step's fields.
func TestWizardValidateWithErrorProvider(t *testing.T) {
	errs := ErrorProvider()
	name := Rule(errs, TextBox().SetAccessibleName("Name"), Required("Enter your name"))
	w := Wizard().AddStep("Who", name).AddStep("End", Label("Thanks")).
		OnValidate(func(int) bool { return errs.Validate() })
	tt := wizardTester(t, w)
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 0 {
		t.Fatalf("an empty required name moved to step %d", got)
	}
	tt.Find(ByLabel("Name")).Type("Ada")
	wizButton(tt, l10n.Next()).Click()
	if got := w.Step(); got != 1 {
		t.Fatalf("a valid name stayed at step %d", got)
	}
}

// Case 8b: Validate runs every rule of its provider, on every step: a
// provider for each step, validated by step, checks only that step's fields.
func TestWizardProviderPerStep(t *testing.T) {
	one, both := ErrorProvider(), ErrorProvider()
	name := Rule(one, TextBox().SetAccessibleName("Name"), Required("Enter your name"))
	Rule(both, name, Required("Enter your name"))
	mail := Rule(both, TextBox().SetAccessibleName("Mail"), Required("Enter a mail"))
	steps := []*ErrorProviderWidget{one, ErrorProvider()}
	Rule(steps[1], mail, Required("Enter a mail"))
	shared := Wizard().AddStep("Who", name).AddStep("Where", mail).AddStep("End", Label("Thanks")).
		OnValidate(func(int) bool { return both.Validate() })
	tt := wizardTester(t, shared)
	tt.Find(ByLabel("Name")).Type("Ada")
	wizButton(tt, l10n.Next()).Click()
	if got := shared.Step(); got != 0 {
		t.Fatalf("one provider for both steps: the empty mail of step 1 did not hold step 0 (step %d); docs/ui/widget/wizard.md says so", got)
	}
	perStep := Wizard().AddStep("Who", name).AddStep("Where", mail).AddStep("End", Label("Thanks")).
		OnValidate(func(step int) bool { return step >= len(steps) || steps[step].Validate() })
	tt = wizardTester(t, perStep)
	wizButton(tt, l10n.Next()).Click() // the same name, which keeps "Ada"
	if got := perStep.Step(); got != 1 {
		t.Fatalf("a provider per step: a filled name stayed at step %d", got)
	}
	wizButton(tt, l10n.Next()).Click()
	if got := perStep.Step(); got != 1 {
		t.Fatalf("a provider per step: the empty mail moved to step %d", got)
	}
}

// Case 9: Done fires OnFinish once; Cancel and Escape fire OnCancel once
// each; Enter is Next, and Done on the last step.
func TestWizardFinishCancel(t *testing.T) {
	var n wizardCounts
	w := threeSteps(&n)
	tt := wizardTester(t, w)
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if got := w.Step(); got != 1 || n.changes != 1 {
		t.Fatalf("Enter: step %d, %d changes; want 1, 1", got, n.changes)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	wizButton(tt, l10n.Done()).Click()
	if n.finishes != 1 || w.Step() != 2 {
		t.Fatalf("Done: %d finishes at step %d; want 1 at 2", n.finishes, w.Step())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if n.finishes != 2 {
		t.Errorf("Enter on the last step: %d finishes, want 2", n.finishes)
	}
	wizButton(tt, l10n.Cancel()).Click()
	if n.cancels != 1 {
		t.Fatalf("Cancel: %d cancels, want 1", n.cancels)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if n.cancels != 2 {
		t.Fatalf("Escape: %d cancels, want 2", n.cancels)
	}
	if e := tt.Escapes(); e != 0 {
		t.Errorf("the window got %d Escapes too; the wizard takes it", e)
	}
}

// Case 10: Enter in a text box of a step is Next, once.
func TestWizardEnterInTextBox(t *testing.T) {
	var n wizardCounts
	w := Wizard().AddStep("Who", TextBox().SetAccessibleName("Name")).AddStep("End", Label("Thanks")).
		OnStepChange(func() { n.changes++ })
	tt := wizardTester(t, w)
	tt.Find(ByLabel("Name")).Type("Ada").Key(0, ui.KeyEnter)
	tt.Settle()
	if got := w.Step(); got != 1 || n.changes != 1 {
		t.Fatalf("Enter in the step's text box: step %d, %d changes; want 1, 1", got, n.changes)
	}
}

// Case 10b: a wizard with no callbacks set works, and posts nothing that
// panics, on Next, Previous, Done and Cancel.
func TestWizardNoCallbacks(t *testing.T) {
	panics := 0
	OnPanic(func(any, []byte) { panics++ })
	t.Cleanup(func() { OnPanic(nil) })
	w := Wizard().AddStep("Account", Label("Step one")).AddStep("Plan", Label("Step two"))
	tt := wizardTester(t, w)
	wizButton(tt, l10n.Next()).Click()
	wizButton(tt, l10n.Previous()).Click()
	wizButton(tt, l10n.Next()).Click()
	wizButton(tt, l10n.Done()).Click()
	wizButton(tt, l10n.Cancel()).Click()
	if panics != 0 || w.Step() != 1 {
		t.Fatalf("no callbacks: %d panics, step %d; want none, 1", panics, w.Step())
	}
}

// Case 10c: in a form with its own accept and cancel buttons, the wizard
// takes Enter and Escape first. Enter in a text box outside the wizard is
// still the form's.
func TestWizardBeforeFormButtons(t *testing.T) {
	var n wizardCounts
	oks, quits := 0, 0
	ok := Button("OK").OnClick(func() { oks++ })
	quit := Button("Quit").OnClick(func() { quits++ })
	w := threeSteps(&n)
	f := NewForm("setup", 600, 400)
	f.SetContent(VBox(8).Pack(w, true, true).Add(TextBox().SetAccessibleName("Outside"), ok, quit)).
		SetAcceptButton(ok).SetCancelButton(quit)
	tt := TestForm(t, f)
	tt.Settle()
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if w.Step() != 1 || oks != 0 {
		t.Fatalf("Enter: step %d, the form's OK %d; want 1, 0", w.Step(), oks)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if n.cancels != 1 || quits != 0 {
		t.Fatalf("Escape: %d cancels, the form's Quit %d; want 1, 0", n.cancels, quits)
	}
	tt.Find(ByLabel("Outside")).Type("x").Key(0, ui.KeyEnter)
	tt.Settle()
	if oks != 1 || w.Step() != 1 {
		t.Fatalf("Enter in a text box after the wizard: the form's OK %d, step %d; want 1, 1", oks, w.Step())
	}
}

// Case 10d: the buttons sit at the end edge in order Previous, Next,
// Cancel, which right to left mirrors: Cancel at the left.
func TestWizardButtonsRightToLeft(t *testing.T) {
	var n wizardCounts
	tt := wizardTester(t, threeSteps(&n))
	x := func(s string) float32 { return wizButton(tt, s).Bounds().X }
	if !(x(l10n.Previous()) < x(l10n.Next()) && x(l10n.Next()) < x(l10n.Cancel())) || x(l10n.Cancel()) < 300 {
		t.Errorf("LTR buttons at %v, %v, %v: want Previous, Next, Cancel at the right", x(l10n.Previous()), x(l10n.Next()), x(l10n.Cancel()))
	}
	tt.SetDirection(ui.RightToLeft)
	tt.Settle()
	if !(x(l10n.Previous()) > x(l10n.Next()) && x(l10n.Next()) > x(l10n.Cancel())) || x(l10n.Previous()) > 300 {
		t.Errorf("RTL buttons at %v, %v, %v: want Cancel, Next, Previous at the left", x(l10n.Previous()), x(l10n.Next()), x(l10n.Cancel()))
	}
}

// Case 10e: the wizard's children are its bar, its buttons and every page,
// shown or not, as a TabControl's are, for Form.ValidateChildren and the
// form's other walks.
func TestWizardChildren(t *testing.T) {
	one, two := Label("Step one"), Label("Step two")
	w := Wizard().AddStep("Account", one).AddStep("Plan", two)
	mu.Lock()
	cs := w.children()
	mu.Unlock()
	for _, c := range []Control{one, two, w.bar, w.prev, w.next, w.cancel} {
		if !slices.Contains(cs, c) {
			t.Errorf("children lack %T", c)
		}
	}
}

// Case 11: a disabled wizard takes no Enter or Escape.
func TestWizardDisabledTakesNoKeys(t *testing.T) {
	var n wizardCounts
	w := threeSteps(&n).SetDisabled(true)
	tt := wizardTester(t, w)
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if w.Step() != 0 || n.changes != 0 || n.cancels != 0 {
		t.Errorf("disabled: step %d, %d changes, %d cancels; want none", w.Step(), n.changes, n.cancels)
	}
}

// Case 12: SetStep moves the step, clamped to the steps, and fires nothing.
func TestWizardSetStepDoesNotFire(t *testing.T) {
	var n wizardCounts
	w := threeSteps(&n).OnValidate(func(step int) bool { n.validated = append(n.validated, step); return true })
	tt := wizardTester(t, w)
	w.SetStep(2)
	tt.Settle()
	if w.Step() != 2 || !tt.Has(ByLabel("Step three")) {
		t.Fatalf("SetStep(2): step %d, shows %q", w.Step(), tt.Texts())
	}
	w.SetStep(0)
	tt.Settle()
	if n.changes != 0 || n.finishes != 0 || n.cancels != 0 || len(n.validated) != 0 {
		t.Errorf("SetStep fired: %+v", n)
	}
	// A Previous whose handler runs after the app moved to step 0 (SetStep
	// between the click and the handler) does nothing.
	Post(w.goBack)
	tt.Settle()
	if n.changes != 0 || w.Step() != 0 {
		t.Errorf("Previous on step 0: %d changes, step %d", n.changes, w.Step())
	}
	if got := w.SetStep(7).Step(); got != 2 {
		t.Errorf("SetStep(7) of 3: Step() = %d, want 2", got)
	}
	if got := w.SetStep(-3).Step(); got != 0 {
		t.Errorf("SetStep(-3): Step() = %d, want 0", got)
	}
}

// Case 13: the setters from another goroutine while frames run show on a
// following frame, without a race.
func TestWizardFromGoroutine(t *testing.T) {
	var n wizardCounts
	w := threeSteps(&n)
	tt := wizardTester(t, w)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			w.AddStep(fmt.Sprint("Extra ", i), Label(fmt.Sprint("Page ", i))).SetStep(i % 5).OnStepChange(func() {}).OnValidate(func(int) bool { return true })
			_ = w.Step()
		}
		w.SetStep(42)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	if w.Step() != 42 || !tt.Has(ByLabel("Page 39")) || !hasButton(tt, l10n.Done()) {
		t.Fatalf("after the goroutine: step %d, shows %q", w.Step(), tt.Texts())
	}
}

// Case 14: a frame at rest of a wizard allocates no more than the same
// controls in boxes: its bar, its page in a scroll panel and its buttons.
func TestWizardAllocNothingAtRest(t *testing.T) {
	frames := func(c Control) float64 {
		tt := ui.NewTester(content(c), 600, 400)
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(30, tt.Frame)
	}
	ref := frames(VBox(8).Add(
		StepProgressBar("Account", "Plan", "Confirm").SetStep(1),
		ScrollPanel(Label("Step two")),
		HBox(8).Add(Button(l10n.Previous()), Button(l10n.Next()), Button(l10n.Cancel())),
	))
	var n wizardCounts
	got := frames(threeSteps(&n).SetStep(1))
	if got > ref {
		t.Fatalf("a wizard's frame at rest allocates %v times; the same controls in boxes, %v", got, ref)
	}
	bar, label := frames(StepProgressBar(fourSteps...).SetStep(2)), frames(Label("One"))
	if bar > label {
		t.Fatalf("a step bar's frame at rest allocates %v times; a label's, %v", bar, label)
	}
	t.Logf("wizard %v, boxes %v; step bar %v, label %v", got, ref, bar, label)
}

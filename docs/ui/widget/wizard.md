# Wizard and StepProgressBar

`Wizard()` walks the user through steps one page at a time, as DevExpress's
`WizardControl`. A `StepProgressBar` sits at the top, the current step's page
in the middle, and Previous, Next and Cancel at the bottom. Next says Done on
the last step. The buttons' text is MyGo's own strings, in the user's
language.

```go
// One ErrorProvider for each step that checks its fields.
account, plan := widget.ErrorProvider(), widget.ErrorProvider()
name := widget.Rule(account, widget.TextBox(), widget.Required("Enter your name"))
code := widget.Rule(plan, widget.TextBox(), widget.Required("Enter a plan code"))
steps := []*widget.ErrorProviderWidget{account, plan}
var dlg *widget.Dialog
wiz := widget.Wizard().
	AddStep("Account", widget.Fields().Row("Name", name)).
	AddStep("Plan", widget.Fields().Row("Code", code)).
	AddStep("Confirm", widget.Label("Ready to create the account.")).
	OnValidate(func(step int) bool { return step >= len(steps) || steps[step].Validate() }).
	OnFinish(func() { dlg.Close() }).
	OnCancel(func() { dlg.Close() })
dlg = widget.NewDialog(nil, "New account", "", 560, 420, wiz)
```

- `AddStep(title, content)` adds a step: its title shows in the bar, and
  `content` is its page. A page taller than the room scrolls, as a
  `TabControl`'s does.
- `Step()` and `SetStep(i)` read and move the current step. An index out of
  range is the nearest step. `SetStep` runs neither `OnValidate` nor
  `OnStepChange`.
- `OnValidate(func(step int) bool)` runs on Next and on Done, with the
  current step. The wizard stays where it returns false. It runs on the
  event goroutine without the package's lock, so it may call
  `ErrorProvider.Validate` (see [ErrorProvider](errorprovider.md)).
  Previous does not ask it.
  - `ErrorProvider.Validate` runs every rule given for its provider, on
    every step. Give each step its own provider and validate the one of
    `step`, as above: one provider for all steps holds the first step until
    the last step's fields are filled.
  - Every step's controls are the wizard's, so a `Rule` on a later step's
    field fails `Form.ValidateChildren` too.
- `OnStepChange(fn)` runs after the user moves to another step, with Next
  or Previous.
- `OnFinish(fn)` runs when Done is clicked on the last step and
  `OnValidate` allows it. The wizard stays on that step: the app closes its
  form.
- `OnCancel(fn)` runs on Cancel and on Escape.

`OnStepChange`, `OnFinish` and `OnCancel` are posted to the event goroutine.

## Keys

- Enter is Next, or Done on the last step, wherever the focus is in the
  window. Enter in a one-line text box of a page is Next too.
- Escape is Cancel.
- A focused button takes Enter itself: Enter on a focused Previous goes
  back.
- Previous is disabled on the first step. With no steps, Done is disabled.
- A disabled wizard takes neither key.

The wizard takes Enter and Escape before the form's accept and cancel
buttons (see [Form keys](form-keys.md)), and Escape no longer closes a
dialog showing it: `OnCancel` closes it.

## StepProgressBar

`StepProgressBar(steps...)` shows the steps of a process as numbered
circles joined by lines, each with its title, as DevExpress's
`StepProgressBar`. The steps before the current one are done: a check in
an accent circle. The current one is an accent circle with its number, and
its title in bold. The steps ahead are outlined, in the muted text color.
The lines up to the current step are in the accent. Colors come from the theme or
the skin.

```go
bar := widget.StepProgressBar("Download", "Install", "Configure")
bar.SetStep(1)
```

- `Step()` and `SetStep(i)`: the current step. An index out of range is the
  nearest step.
- `SetSteps(steps...)` replaces the steps. The current step stays, or is
  the last where there are fewer now.
- `SetVertical(true)` stacks the steps from the top, each circle before its
  title.

The steps run from the start edge, so right to left the first step is at
the right, and a vertical bar's circles are at the right of their titles.
Assistive technology reads the current step as the current one.

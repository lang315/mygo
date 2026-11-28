package widget_test

import "github.com/egoist/mygo/ui/widget"

// The Wizard of docs/ui/widget/wizard.md: three steps in a dialog, each
// step's fields checked by its own ErrorProvider before the user moves on.
func ExampleWizard() {
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
}

// A StepProgressBar alone, showing where a long task is.
func ExampleStepProgressBar() {
	bar := widget.StepProgressBar("Download", "Install", "Configure")
	bar.SetStep(1)
	bar.SetVertical(true)
}

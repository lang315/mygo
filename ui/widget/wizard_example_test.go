package widget_test

import "github.com/egoist/mygo/ui/widget"

// The Wizard of docs/ui/widget/wizard.md: three steps in a dialog, the
// first checked before the user moves on.
func ExampleWizard() {
	errs := widget.ErrorProvider()
	name := widget.Rule(errs, widget.TextBox(), widget.Required("Enter your name"))
	var dlg *widget.Dialog
	wiz := widget.Wizard().
		AddStep("Account", widget.Fields().Row("Name", name)).
		AddStep("Plan", widget.Label("The free plan")).
		AddStep("Confirm", widget.Label("Ready to create the account.")).
		OnValidate(func(step int) bool { return step != 0 || errs.Validate() }).
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

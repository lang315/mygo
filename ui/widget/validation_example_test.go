package widget_test

import (
	"regexp"

	"github.com/egoist/mygo/ui/widget"
)

func saveSignUp(name, email string, age float64) {}

// The snippet of docs/ui/widget/errorprovider.md's rules, which must compile.
func ExampleRule() {
	errs := widget.ErrorProvider().SetValidateOnLeave(true)
	name := widget.Rule(errs, widget.TextBox(), widget.Required("Enter your name"))
	email := widget.Rule(errs, widget.TextBox(), widget.Required("Enter an email address"))
	widget.Rule(errs, email, widget.MatchRegexp(regexp.MustCompile(`^\S+@\S+\.\S+$`), "Not an email address"))
	age := widget.Rule(errs, widget.NumericUpDown(0, 150, 1), widget.InRange(18, 120, "18 or over"))

	form := widget.NewForm("Sign up", 400, 300)
	form.SetContent(widget.Fields().Row("Name", name).Row("Email", email).Row("Age", age).Row("",
		widget.Button("Sign up").OnClick(func() {
			if !form.ValidateChildren() {
				errs.FocusFirst(form)
				return
			}
			saveSignUp(name.Text(), email.Text(), age.Value())
		})))
}

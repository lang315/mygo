# ErrorProvider and validation

`ErrorProvider()` shows why a control's value is not valid, as WinForms'
`ErrorProvider`: a red mark over the end of the control, and the message as its
tooltip, which a keyboard user sees when the control has the focus; for assistive
technology the value is invalid and the message comes first in its description,
before the control's own tooltip text. The
mark is drawn over the control and takes no room, so showing it moves nothing,
and it draws beside a `TextBox`'s own icon (a `TextBox` keeps its text clear of the mark while it has an error). The provider holds controls weakly: one the app drops goes with its error.

```go
ep := widget.ErrorProvider()
email := widget.TextBox()
email.OnValidating(func() bool {
	if strings.Contains(email.Text(), "@") {
		ep.SetError(email, "")
		return true
	}
	ep.SetError(email, "Enter an email address")
	return false
})
save := widget.Button("Save").OnClick(func() {
	if form.ValidateChildren() {
		store(email.Text())
	}
})
```

- `SetError(w, msg)` marks `w`; `""` takes the mark off. `Error(w)` returns the
  message, `HasErrors()` says whether any control is marked, `Clear()` takes
  every mark off. A control has one error: the last provider's.
- `OnValidating(fn)` (any control) sets a handler for `Form.ValidateChildren()`;
  `fn` returns false when the value is not valid, and usually sets the error.
- `Form.ValidateChildren()` runs the handler of every control in the form's
  widget tree, in the order they are laid out, and returns whether all
  returned true. Every handler runs, so every error is set. That includes
  controls on pages of a `TabControl` or `Sidebar` that are not showing; only a
  control made invisible with `SetVisible(false)` (and what is in it) is
  skipped. A disabled control is validated, as in WinForms. A handler that
  panics ends the validation: the panic propagates and later handlers do not
  run. When it returns false, the first failing control's message is
  announced to assistive technology once. Call it from the event goroutine (a
  button's handler), not from a view.
- `FocusFirst(form)` gives the keyboard focus, at the next frame, to the first
  control the provider marks, in Tab order (not the order the errors were set),
  and reports whether there is one: a hidden or disabled control, or one in a
  hidden or disabled container, is passed over, and a control on a page of a
  `TabControl` or `Sidebar` that is not showing is reached by showing its pages
  first (`ValidateChildren` does not switch pages). Safe from any goroutine.

  ```go
  if !form.ValidateChildren() {
  	ep.FocusFirst(form)
  }
  ```
- **Nothing validates by itself:** not a focus change, not a typed character.
  Call `ValidateChildren` (or your own checks, with `SetError`) when you want
  errors, or turn on `SetValidateOnLeave` for rules.

## Rules

A rule checks a control's value for a provider, as DevExpress's
`DXValidationProvider` does, with no handler to write:

```go
errs := widget.ErrorProvider().SetValidateOnLeave(true)
name := widget.Rule(errs, widget.TextBox(), widget.Required("Enter your name"))
email := widget.Rule(errs, widget.TextBox(), widget.Required("Enter an email address"))
widget.Rule(errs, email, widget.MatchRegexp(regexp.MustCompile(`^\S+@\S+\.\S+$`), "Not an email address"))
age := widget.Rule(errs, widget.NumericUpDown(0, 150, 1), widget.InRange(18, 120, "18 or over"))

form.SetContent(widget.Fields().Row("Name", name).Row("Email", email).Row("Age", age).Row("",
	widget.Button("Sign up").OnClick(func() {
		if !form.ValidateChildren() {
			errs.FocusFirst(form)
			return
		}
		saveSignUp(name.Text(), email.Text(), age.Value())
	})))
```

- `Rule(p, c, check)` gives `c` a check for `p` and returns `c`. It is a
  package function, as `Bind` is, so `check` takes the control's value: a
  `string` for a `TextBox`, a `float64` for a `NumericUpDown` (any `Bindable`
  control). `check` returns the error text, or `""` when the value is valid.
  A control may have several rules: the first failing one's text shows. Rules
  of two providers on one control each run, and the last to mark it wins, as
  with `SetError`.
- Ready rules: `Required(msg)` fails on text that is empty once spaces are
  trimmed; `MatchRegexp(re, msg)` on text `re` does not match; `InRange(min,
  max, msg)` on a number below `min` or above `max` (both bounds are in).
- `p.Validate()` runs every rule of `p`, sets or clears each ruled control's
  mark, and returns true when none fails. It does not move the focus:
  `FocusFirst` does. A control with no rule keeps any mark `SetError` gave it.
- `p.SetValidateOnLeave(true)` also runs a control's rules when the focus
  leaves it, after its `OnFocusOut`. Off by default, as DevExpress's Manual
  mode: only `Validate` and `ValidateChildren` run rules then.
- `Form.ValidateChildren()` runs the rules of the controls it validates (those
  in the form and not hidden), after every `OnValidating` handler, and
  returns false when any fails. With no failing handler, the first failing
  rule's text is what it announces.
- Checks run on the goroutine that validates, with no lock held, so they may
  call the control's getters. An app that never calls `Rule` links no rule
  code.

WinForms' icon alignment and padding, `BlinkStyle`, `DataSource` and the
`Validating` event's `Cancel` that keeps the focus in the control are not here.

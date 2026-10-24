# ErrorProvider and validation

`ErrorProvider()` shows why a control's value is not valid, as WinForms'
`ErrorProvider`: a red mark over the end of the control, and the message as its
tooltip (which assistive technology reads as the control's description). The
mark is drawn over the control and takes no room, so showing it moves nothing,
and it draws beside a `TextBox`'s own icon.

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
- `Form.ValidateChildren()` runs the handler of every control the form shows,
  in the order they are laid out, and returns whether all returned true. Every
  handler runs, so every error is set. Hidden controls and controls the form
  never showed are skipped. Call it from the event goroutine (a button's
  handler), not from a view.
- **Nothing validates by itself:** not a focus change, not a typed character.
  Call `ValidateChildren` (or your own checks, with `SetError`) when you want
  errors.

WinForms' icon alignment and padding, `BlinkStyle`, `DataSource` and the
`Validating` event's `Cancel` that keeps the focus in the control are not here.

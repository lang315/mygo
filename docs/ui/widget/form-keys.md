# Form shortcuts, AcceptButton and CancelButton

A `Form` takes keyboard shortcuts of its own, which work whichever control has
the focus, and can name the buttons Enter and Escape click, as WinForms'
`AcceptButton` and `CancelButton`.

```go
dlg := widget.NewDialog(form, "Sign in", "", 360, 200, body)
dlg.Shortcut(ui.Ctrl, ui.KeyS, func() { save() })
dlg.SetAcceptButton(ok)
dlg.SetCancelButton(cancel)
```

## Shortcut(mods, key, fn)

`fn` runs on the event goroutine when the user presses `key` with exactly
`mods`, unless the control with the focus takes the key itself: a focused
button takes Enter and Space, a check box Space, a closed combo box Enter (it
opens its list), an open drop-down Escape and the arrows, and a text field,
a number box and an editable combo box what they type and edit. The same `mods` and `key`
again replace the handler; a nil `fn` removes it. Set and removed from any
goroutine. A form behind a modal dialog gets no key at all: the operating system's modal
window takes the keyboard (not tested here). The platform menu
handles the accelerators of a `MenuStrip` before the window sees the key (not
tested here: the fake backend has no native menu).

## SetAcceptButton(b) and SetCancelButton(b)

Both take a `*ButtonWidget`; nil clears. A disabled button, a hidden one (inside a hidden
container too), and one this form did not build in its last frame (another
form's) get no key, which then goes on as if none were set.

- **Enter** clicks the accept button from wherever the focus is, except where
  the control with the focus takes Enter: another button clicks itself, a
  multiline text box adds a line, a text box with `OnEnter` runs `OnEnter`
  **only** (the accept button is not clicked, so a form whose `OnEnter` submits
  does not submit twice). A text box, a number box or an editable combo box without `OnEnter` clicks the
  accept button.
  The accept button with the focus clicks itself once.
- **Escape** first closes an open drop-down. Then, with a cancel button, it
  clicks it **instead of** a dialog's request to close: `OnClose` is not asked,
  and `SetDeletable(false)` does not stop it. The button's `OnClick` decides, so
  a cancel button whose `OnClick` does not close leaves the dialog open.
  Without one, or with it disabled or hidden, Escape closes a dialog as before.

In a test, `dt.Escape()` of a `DialogTester` clicks the cancel button, and
`Find(...).Key(0, ui.KeyEnter)` presses Enter (see
[`widget.Tester`](tester.md)).

# Toasts

`Form.ShowToast(t)` shows a short message near the bottom of the window for a
few seconds, as the outcome of what the user just did: "12 accounts
exported". It does not take the focus or wait for a click, as a message box
does. Use a toast to tell an outcome, and a message box (`Alert`, `Confirm`)
to ask a question or stop the user on an error that needs reading.

```go
export.OnClick(func() {
	n := app.export()
	w.ShowToast(widget.Toast{Status: widget.StatusSuccess, Title: "Đã xuất", Message: fmt.Sprintf("%d tài khoản", n)})
})

trash.OnClick(func() {
	app.trash(note)
	w.ShowToast(widget.Toast{Title: "Đã xoá ghi chú", Action: "Hoàn tác", OnAction: func() { app.restore(note) }})
})
```

A `Toast` has:

- `Status`: colors the toast and picks its icon, as an
  [InfoBar](infobar.md)'s; `StatusNone` is neutral.
- `Title`, in semibold, and `Message`, which wraps.
- `Timeout`: 4 seconds when zero, 8 with an `Action`, until closed when
  negative. The time stops while the pointer rests on the toasts, or the
  keyboard focus is in them.
- `Action` and `OnAction`: a button that runs `OnAction` and closes the toast.
- `OnClose`: runs once when the toast closes, for any reason: its time is
  over, the user closed it (its close button, Escape with the focus in it,
  the action), or `CloseToast`.
- `ID`: showing a toast with the ID of one showing changes that toast.
  `ShowToast` makes one up when it is empty, and returns the ID.

`Form.CloseToast(id)` closes the toast of that ID; `CloseToast("")` closes
them all.

```go
id := w.ShowToast(widget.Toast{Status: widget.StatusRunning, Title: "Đang tải lên", Timeout: -1})
// ...
w.CloseToast(id)
```

`ShowToast` and `CloseToast` are safe from any goroutine. A form not shown
yet keeps its toasts, which show from its first frame, their time starting
then. A dialog's `ShowToast` (a `Dialog` is a `Form`) shows in the dialog's
window. `OnAction` and `OnClose` run on the event goroutine, as a button's
`OnClick` does, so they may show another toast.

The newest three show, bottom-centred, the newest at the bottom; older ones
wait, their time running. A toast is at most 400 DIPs wide and keeps 16 DIPs
from the window's sides. Its fill is the status's (as an InfoBar's) over the
theme's `Surface`, opaque, with the status's text and border and a shadow.
Assistive technology reads it out as it shows: an alert (`ui.RoleAlert`)
for `StatusWarning` and `StatusDanger`, a status (`ui.RoleStatus`) for the
rest. The close button is named "Close", in the user's language.

Once the app called `ShowToast`, a toast that `ui` code adds itself
(`c.Toast` in a `Custom` control) shows in the same look, neutral, in every
form; before, in `ui`'s own. An app that never calls
`ShowToast` links none of this.

In tests, a [Tester](tester.md) of the form shows the toasts; advance its
clock (`Advance`) to see one close.

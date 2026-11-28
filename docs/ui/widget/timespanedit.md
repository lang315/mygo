# TimeSpanEdit

`TimeSpanEdit()` edits a `time.Duration` in a one-line field, as
DevExpress's `TimeSpanEdit`. It shows the duration as `[-][d.]hh:mm[:ss]`:
`01:30:00` for 90 minutes, `26:00:00` for 26 hours, or `1.02:00:00` with
the days shown.

```go
var timeout *widget.TimeSpanEditWidget
timeout = widget.TimeSpanEdit().
	SetRange(time.Second, 2*time.Hour).
	SetValue(90 * time.Second).
	OnChange(func() { save(timeout.Value()) })
```

- `Value()` and `SetValue(d)`; `SetValue` clamps `d` to the range and does
  not run `OnChange`.
- `SetRange(min, max)` sets the least and greatest duration, `0` to
  `9999:59:59` at first, and clamps the value to them, without `OnChange`.
  `min` greater than `max` swaps them. A negative `min` lets the value go
  below zero, shown with a leading `-`.
- `SetShowSeconds(false)` hides the seconds (shown at first). Hidden seconds
  stay in the value, through steps too, until the user types a new value.
- `SetShowDays(true)` shows the days before the hours, so the hours stop at
  23. Without it (the default) the hours run past 23.
- Only whole seconds show. A value with a fraction of a second keeps it
  through steps, until the user types a new value.
- `OnChange(fn)` runs on the event goroutine once after each change by the
  user: a step, or a typed value read on leaving the field. Not for a setter,
  and not for a step that the range stops.
- It can be bound to a `time.Duration`: `widget.Bind(timeout, get, set)`. A
  source value outside the range is clamped and not written back. See
  [Binding](../widget.md#binding).

## Keyboard and buttons

Up and Down step the part under the caret by one: days, hours, minutes or
seconds. The step carries into the next part, so Up on `00:59:00` with the
caret in the minutes gives `01:00:00`. A step that goes past the range stops
at its bound. The caret stays in the part it stepped. The − and + buttons at
the field's end do the same as Down and Up. They are not Tab stops, and a
click on one leaves the keyboard in the text. In a field that has never had
the focus, they step its first part (the hours, or the days when shown).

You can also type a whole value: `1:2:3`, `1:30`, `5` (hours) or
`1.02:00:00`, with minutes and seconds below 60. It is read when the focus
leaves the field, or on Enter, which then clicks the form's accept button as
in a `TextBox`. Text that does not read as a duration goes back to the last
value, and `OnChange` does not run. A step while typed text is in the field
starts from that text when it reads as a duration, else from the last value.

## Look

The field has the frame of a `ButtonEdit`: the skin's look of a `TextBox`
(`Field`), with the accent on its edge while the text has the focus. In a
right-to-left layout the buttons sit at the field's left. The icons use the
theme's muted text color.

## Testing

Assistive technology sees a text field, then the buttons "Decrease" and
"Increase", the names ui's `SpinBox` gives its arrows:

```go
in := tt.Find(widget.ByRole(ui.RoleTextField))
in.Focus()
in.Key(0, ui.KeyHome)
in.Key(0, ui.KeyUp) // one hour more
tt.Find(widget.ByRole(ui.RoleButton).Label("Increase")).Click()
```

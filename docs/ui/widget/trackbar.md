# TrackBar

`TrackBar(min, max)` is a slider choosing a number from `min` to `max`. The user
drags the knob, presses on the track, or uses the arrows, Page Up and Page Down
(a tenth of the range), Home and End while it has the keyboard focus. It is
[`ui.Slider`](../slider.md) underneath.

```go
volume := widget.TrackBar(0, 100).SetValue(60).OnChange(func() { player.SetVolume(volume.Value()) })
quality := widget.TrackBar(0, 100).SetStep(25) // 0, 25, 50, 75, 100, with ticks
```

- `Value()` and `Int()`; `SetValue(v)` clamps to the range and does not run
  `OnChange`.
- `SetStep(step)` snaps the value to the minimum and multiples of `step` from
  it, shows a tick at each, and has the arrows move by `step`. 0 is a
  continuous track, the default.
- `OnChange(fn)` runs on the event goroutine after the user moved the knob, for
  every value of a drag, so save it from a debounced handler rather than
  on every call.
- It can be bound to a float: `widget.Bind(volume, get, set)`. See
  [Binding](../widget.md#binding).

WinForms' vertical `Orientation`, `SmallChange` and `LargeChange` are not here:
`ui.Slider` is horizontal, and its keys are fixed.

Assistive technology sees a slider with its minimum, maximum and value; in a
test, `tt.Find(widget.ByRole(ui.RoleSlider))` has `Min()`, `Max()` and `Now()`
(see [`widget.Tester`](tester.md)).

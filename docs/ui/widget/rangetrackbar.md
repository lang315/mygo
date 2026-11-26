# RangeTrackBar

`RangeTrackBar(min, max)` is a slider of two knobs choosing a range from `min`
to `max`, as a price range: DevExpress's RangeTrackBarControl. The user drags a
knob, presses on the track to move the nearest knob, or uses the arrows, Home
and End on the knob that has the keyboard focus (Tab moves between them). It is
[`ui.RangeSlider`](../range-slider.md) underneath.

```go
var price *widget.RangeTrackBarWidget
shown := widget.Label("$0 to $500")
price = widget.RangeTrackBar(0, 500).SetStep(10).SetValues(50, 200).OnChange(func() {
	lo, hi := price.Values()
	shown.SetText(fmt.Sprintf("$%.0f to $%.0f", lo, hi))
})
```

- It starts at the whole range: `Values()` gives `(min, max)`. A `max` below
  `min` is raised to it, as `TrackBar`'s.
- `SetValues(low, high)` moves both knobs: a value outside the track goes to
  its end, and `low` and `high` swap when `low` is above `high`. It does not
  run `OnChange`; a NaN is ignored.
- `SetStep(step)` snaps the user's values to the minimum and multiples of
  `step` from it, with a tick at each, and has the arrows move by `step`. 0 is
  a continuous track, the default, whose arrows move by a hundredth.
- `OnChange(fn)` runs on the event goroutine after the user moved a knob, for
  every value of a drag.

Assistive technology sees two sliders, "minimum" and "maximum"; in a test,
`tt.Find(widget.ByRole(ui.RoleSlider).Nth(0))` is the low knob, with `Now()`.

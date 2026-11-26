# Rating

`Rating(max)` is a row of `max` stars, of which the first `Value()` are filled:
DevExpress's RatingControl. A click on a star sets the value, a click on the
star already set clears it to 0, and the arrows, Home and End change it while it
has the keyboard focus. It is [`ui.Rating`](../rating.md) underneath.

```go
var stars *widget.RatingWidget
stars = widget.Rating(5).SetValue(3).OnChange(func() { fmt.Println("rated", stars.Value()) })
score := 4
bound := widget.Bind(widget.Rating(5), func() int { return score }, func(v int) { score = v })
```

- It starts at 0. A `max` below 0 is 0.
- `SetValue(v)` clamps to 0 through `max` and does not run `OnChange`.
- `OnChange(fn)` runs on the event goroutine after the user changed the
  rating, once a change.
- It can be bound to an int: `widget.Bind(stars, get, set)`. See
  [Binding](../widget.md#binding).

Assistive technology sees a slider from 0 to `max`; in a test,
`tt.Find(widget.ByRole(ui.RoleSlider))` has `Now()`.

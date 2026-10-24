# Chart

`Chart(kind)` draws a line, bar or pie chart of its data. It is a thin wrapper
over [`ui/chart`](../chart.md): no chart engine of its own, and no look
settings. For a chart of your own, use `ui/chart` in a `Custom` control.

```go
var sales *widget.ChartWidget
sales = widget.Chart(widget.BarChart).
	SetTitle("Sales").
	SetLabels("Jan", "Feb", "Mar").
	SetSeries(widget.ChartSeries{Name: "2026", Values: []float64{10, 20, 30}}).
	OnSelect(func() { show(sales.Selected()) })

// Later, from any goroutine:
latest := []float64{1, 2, 3}
sales.SetSeries(widget.ChartSeries{Name: "2026", Values: latest})
```

- `Chart(widget.LineChart | BarChart | PieChart)`.
- `SetLabels(labels...)`: the x axis of a line or bar chart, the slices of a
  pie. `SetSeries(series...)`: one series for a pie (its values, in the labels'
  order), any number for lines and bars, each in its own color, with a legend
  when there are two or more (twelve entries at most, then "+N more").
- A value beyond the labels is not drawn. **NaN and infinite values are left
  out** everywhere: a line breaks there, a bar is missing, a slice is empty.
- Both setters copy their arguments and take effect at the next frame, from any
  goroutine. A selection beyond the new labels or data is cleared, without
  running `OnSelect`.
- `SetTitle(s)` names the plot for assistive technology (`"Chart"` by default).

## Reading and selecting

A line or bar chart is **one** control, not one per point, so a chart of ten
thousand points costs a frame what one of ten does. The point nearest the
pointer, or the one the arrows, Home and End come to (the plot takes the
focus), is the plot's value for assistive technology, and its tooltip-like
description: `"2026, Jan: 10"`, or all the series at that label
(`"Search, Q1: 5; Direct, Q1: 7"`). A pie is the same: it takes the focus, the arrows, Home and End move between slices, Escape clears, Enter and Space select the one the keys are on, and it names the slice under the pointer or the keys
(`"Win: 60 (60%)"`, one decimal under 1%).

`Selected()` and `OnSelect(fn)`: a click selects the label of that point or
slice (`-1` for none). `OnSelect` runs on the event goroutine once per click.
A setter does not run it.

In a test, move the pointer over the plot and read the node's value:
`tt.Move(x, y)`, then `tt.Find(widget.ByLabel("Sales")).Value()` (see
[`widget.Tester`](tester.md)).

Not here: scales other than linear, stacking, zoom, area charts, axis titles,
chart colors of your own. A line is always
drawn reduced to the pixel columns (the lowest and highest sample of each), so that spikes still show.

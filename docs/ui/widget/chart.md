# Chart

`Chart(kind)` draws a line, bar or pie chart of its data. It is a thin wrapper
over [`ui/chart`](../chart.md): no chart engine of its own, and no look
settings. For a chart of your own, use `ui/chart` in a `Custom` control.

```go
sales := widget.Chart(widget.BarChart).
	SetTitle("Sales").
	SetLabels("Jan", "Feb", "Mar").
	SetSeries(widget.ChartSeries{Name: "2026", Values: []float64{10, 20, 30}}).
	OnSelect(func() { show(sales.Selected()) })

// Later, from any goroutine:
sales.SetSeries(widget.ChartSeries{Name: "2026", Values: latest})
```

- `Chart(widget.LineChart | BarChart | PieChart)`.
- `SetLabels(labels...)`: the x axis of a line or bar chart, the slices of a
  pie. `SetSeries(series...)`: one series for a pie (its values, in the labels'
  order), any number for lines and bars, each in its own color, with a legend
  when there are two or more. A value beyond the labels is not drawn; NaN
  leaves a gap in bars and a hole in nothing else (a line mark is skipped).
- Both setters copy their arguments and take effect at the next frame, from any
  goroutine.
- `SetTitle(s)` names the chart for assistive technology.
- `Selected()` and `OnSelect(fn)`: clicking a bar, a point or a slice selects
  its label (`-1` for none). `OnSelect` runs on the event goroutine once per
  click. A setter does not run it, and `SetLabels` clears a selection beyond
  the new labels.

Each bar, point and slice is a button named by its data (`"2026, Jan: 10"`,
`"Win: 60 (60%)"`) with the same text as its tooltip. In a test,
`tt.FindAll(widget.ByRole(ui.RoleButton))` lists them (see
[`widget.Tester`](tester.md)).

Not here: scales other than linear, stacking, zooming, area charts, axis
titles and chart colors of your own.

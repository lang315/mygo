package widget_test

import "github.com/egoist/mygo/ui/widget"

func show(int) {}

// The snippet of docs/ui/widget/chart.md, which must compile.
func ExampleChart() {
	var sales *widget.ChartWidget
	sales = widget.Chart(widget.BarChart).
		SetTitle("Sales").
		SetLabels("Jan", "Feb", "Mar").
		SetSeries(widget.ChartSeries{Name: "2026", Values: []float64{10, 20, 30}}).
		OnSelect(func() { show(sales.Selected()) })

	// Later, from any goroutine:
	latest := []float64{1, 2, 3}
	sales.SetSeries(widget.ChartSeries{Name: "2026", Values: latest})
}

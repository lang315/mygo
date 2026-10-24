package widget

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// ChartKind says how a Chart draws its data.
type ChartKind uint8

const (
	// LineChart draws each series as a line through its values.
	LineChart ChartKind = iota
	// BarChart draws each value as a bar, the series side by side.
	BarChart
	// PieChart divides a donut among the first series' values.
	PieChart
)

// ChartSeries is a named list of values, one for each label of the chart.
type ChartSeries struct {
	Name   string
	Values []float64
}

// ChartWidget draws a line, bar or pie chart of its data, over package
// ui/chart. Set the data with SetLabels and SetSeries from any goroutine;
// the chart draws the new data at the next frame. It has no look settings:
// use ui/chart in a Custom control for a chart of your own.
type ChartWidget struct {
	common[ChartWidget]
	kind     ChartKind
	title    string
	labels   []string
	series   []ChartSeries
	selected int
	onSelect func()
	taken    int // the selection a click made, to report once
}

// Chart makes an empty chart of a kind.
func Chart(kind ChartKind) *ChartWidget {
	c := &ChartWidget{kind: kind, selected: -1, taken: -1}
	c.init(c)
	return c
}

// SetTitle names the chart for assistive technology.
func (c *ChartWidget) SetTitle(s string) *ChartWidget {
	c.change(func() { c.title = s })
	return c
}

// SetLabels sets the category of each value: the x axis of a line or bar
// chart, the slices of a pie. The slices are copied.
func (c *ChartWidget) SetLabels(labels ...string) *ChartWidget {
	c.change(func() {
		c.labels = append([]string(nil), labels...)
		if c.selected >= len(c.labels) {
			c.selected = -1
		}
	})
	return c
}

// SetSeries sets the data: one series for a pie, any number for a line or
// bar chart, whose legend names them. A value beyond the labels is not
// drawn, a missing one is not either, and NaN breaks a line and leaves a
// gap in bars.
func (c *ChartWidget) SetSeries(series ...ChartSeries) *ChartWidget {
	c.change(func() {
		c.series = make([]ChartSeries, len(series))
		for i, s := range series {
			c.series[i] = ChartSeries{s.Name, append([]float64(nil), s.Values...)}
		}
	})
	return c
}

// Selected returns the index of the label the user clicked a bar, point or
// slice of, -1 for none.
func (c *ChartWidget) Selected() int { mu.Lock(); defer mu.Unlock(); return c.selected }

// OnSelect runs fn, on the event goroutine, after the user clicked a bar,
// point or slice. A setter does not run it.
func (c *ChartWidget) OnSelect(fn func()) *ChartWidget {
	c.change(func() { c.onSelect = fn })
	return c
}

// chartColors are the series' and slices' colors, in order.
var chartColors = []ui.Color{
	ui.Hex("#2563eb"), ui.Hex("#16a34a"), ui.Hex("#ea580c"), ui.Hex("#9333ea"),
	ui.Hex("#0891b2"), ui.Hex("#db2777"), ui.Hex("#ca8a04"), ui.Hex("#4b5563"),
}

func chartColor(i int) ui.Color { return chartColors[i%len(chartColors)] }

func fmtValue(v float64) string { return fmt.Sprintf("%.4g", v) }

func (c *ChartWidget) build(ctx *ui.Context) ui.Element {
	t := ctx.Theme()
	pick := -1
	root := ui.Column(ctx).Gap(t.Space(2)).Role(ui.RoleGroup)
	if c.title != "" {
		root.Label(c.title)
	}
	root.Children(func() {
		switch c.kind {
		case PieChart:
			pick = c.buildPie(ctx)
		default:
			pick = c.buildPlot(ctx)
		}
		c.buildLegend(ctx)
	})
	if pick >= 0 {
		c.selected = pick
		if h := c.onSelect; h != nil {
			Post(h)
		}
	}
	return root
}

func (c *ChartWidget) buildLegend(ctx *ui.Context) {
	t := ctx.Theme()
	if c.kind != PieChart && len(c.series) < 2 {
		return
	}
	ui.Row(ctx).Gap(t.Space(3)).Children(func() {
		items := len(c.series)
		if c.kind == PieChart {
			items = len(c.labels)
		}
		for i := 0; i < items; i++ {
			name := ""
			if c.kind == PieChart {
				name = c.labels[i]
			} else {
				name = c.series[i].Name
			}
			ui.Row(ctx.Key(i)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				ui.Box(ctx).Size(10, 10).Radius(2).Background(chartColor(i))
				ui.Text(ctx, name).NoWrap().FontSize(t.FontSize - 1)
			})
		}
	})
}

// extent returns the lowest and highest value drawn, with 0 inside.
func (c *ChartWidget) extent() (lo, hi float64) {
	for _, s := range c.series {
		for i, v := range s.Values {
			if i >= len(c.labels) || math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if hi == lo {
		hi = lo + 1
	}
	return lo, hi
}

func (c *ChartWidget) buildPlot(ctx *ui.Context) (picked int) {
	t := ctx.Theme()
	picked = -1
	lo, hi := c.extent()
	y := chart.NewLinear(lo, hi).Nice(4)
	x := chart.NewBand(c.labels...).Padding(0.25, 0.1)
	if c.kind == LineChart {
		x = chart.NewBand(c.labels...)
	}
	ticks := y.Ticks(4)
	label := func(side chart.Side, tk chart.Tick) {
		ui.Text(ctx, tk.Label).NoWrap().FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	}
	chart.Plot(ctx, chart.Axes{Left: ticks, Bottom: x.Ticks(), Label: label}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tk := range ticks {
				p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border)
			}
			if c.kind != LineChart {
				return
			}
			for si, s := range c.series {
				xs, ys := make([]float32, 0, len(s.Values)), make([]float32, 0, len(s.Values))
				for i, v := range s.Values {
					if i >= len(c.labels) {
						break
					}
					xs = append(xs, x.At(i))
					if math.IsNaN(v) {
						ys = append(ys, float32(math.NaN()))
					} else {
						ys = append(ys, y.At(v))
					}
				}
				p.StrokePath(chart.Line(r, xs, ys, chart.CurveLinear), 2, chartColor(si))
			}
		})
		n := max(len(c.series), 1)
		for si, s := range c.series {
			for i, v := range s.Values {
				if i >= len(c.labels) || math.IsNaN(v) {
					continue
				}
				name := fmt.Sprintf("%s: %s", c.labels[i], fmtValue(v))
				if s.Name != "" {
					name = s.Name + ", " + name
				}
				var mark ui.Element
				if c.kind == LineChart {
					mark = chart.Point(ctx, x.At(i), y.At(v), 10, 10).Radius(5).Background(chartColor(si))
				} else {
					b := x.Band(i)
					w := (b.Hi - b.Lo) / float32(n)
					mark = chart.Rect(ctx, chart.Span{Lo: b.Lo + w*float32(si), Hi: b.Lo + w*float32(si+1)}, y.Span(0, v)).
						Background(chartColor(si)).Radius(2)
				}
				mark = mark.Key(si*100000 + i).Label(name).Tooltip(name).Role(ui.RoleButton)
				if i == c.selected {
					mark.Border(2, t.Text)
				}
				if mark.Clicked() {
					picked = i
				}
			}
		}
	}).Height(220).Gap(6)
	return picked
}

func (c *ChartWidget) buildPie(ctx *ui.Context) (picked int) {
	picked = -1
	var values []float64
	if len(c.series) > 0 {
		for i, v := range c.series[0].Values {
			if i < len(c.labels) && v > 0 && !math.IsInf(v, 0) {
				values = append(values, v)
			} else if i < len(c.labels) {
				values = append(values, 0)
			}
		}
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	slices := chart.Pie(values, 0.02)
	chart.Plot(ctx, chart.Axes{}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			size := min(r.W, r.H)
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			for i, s := range slices {
				outer := size / 2
				p.FillPath(chart.Sector(cx, cy, outer*0.55, outer, s), chartColor(i))
			}
		})
		for i, s := range slices {
			if total == 0 || values[i] == 0 {
				continue
			}
			mid := (s.Start + s.End) / 2
			share := values[i] / total * 100
			name := fmt.Sprintf("%s: %s (%.0f%%)", c.labels[i], fmtValue(values[i]), share)
			// The middle of the ring, a fraction of the square plot.
			fx, fy := float32(0.5+0.39*math.Sin(mid)), float32(0.5+0.39*math.Cos(mid))
			mark := chart.Point(ctx, fx, fy, 16, 16).Radius(8).Key(i).Label(name).Tooltip(name).Role(ui.RoleButton)
			if mark.Clicked() {
				picked = i
			}
		}
	}).Height(200).Width(200)
	return picked
}

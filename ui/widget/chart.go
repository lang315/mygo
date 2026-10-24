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
// the chart draws the new data at the next frame. NaN and infinite values
// are not drawn: a line breaks there and a bar is missing. It has no look
// settings: use ui/chart in a Custom control for a chart of your own.
//
// Line and bar charts are one control that takes the pointer and the
// keyboard: the point nearest the pointer, or the one the arrows, Home and
// End come to, is named for assistive technology and in the tooltip-like
// value of the plot; a click selects it. The cost of a frame does not grow
// with the number of points that are not drawn.
type ChartWidget struct {
	common[ChartWidget]
	kind     ChartKind
	title    string
	labels   []string
	series   []ChartSeries
	selected int
	onSelect func()
	hover    int // the point under the pointer or the keys, -1 for none

	// xs are the fractions of the labels along the plot, kept while the
	// labels stay (replaced, never changed, so a paint may hold one).
	xs       []float32
	xsLabels int
	// The drawing's scratch, touched only by paints, outside mu.
	bx, by    []float32
	points    [][2]float32
	ysScratch []float32
}

// Chart makes an empty chart of a kind.
func Chart(kind ChartKind) *ChartWidget {
	c := &ChartWidget{kind: kind, selected: -1, hover: -1}
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
		c.xs = nil
		c.clampSelection()
	})
	return c
}

// SetSeries sets the data: one series for a pie, any number for a line or
// bar chart, whose legend names them. A value beyond the labels is not
// drawn, a missing one is not either, and NaN or infinite values are left
// out: a line breaks there and a bar is missing. A selection beyond the new
// data is cleared, without running OnSelect.
func (c *ChartWidget) SetSeries(series ...ChartSeries) *ChartWidget {
	c.change(func() {
		c.series = make([]ChartSeries, len(series))
		for i, s := range series {
			c.series[i] = ChartSeries{s.Name, append([]float64(nil), s.Values...)}
		}
		c.clampSelection()
	})
	return c
}

// clampSelection clears a selection beyond the data. Under mu.
func (c *ChartWidget) clampSelection() {
	n := 0
	for _, s := range c.series {
		n = max(n, len(s.Values))
	}
	if c.selected >= min(n, len(c.labels)) {
		c.selected = -1
	}
}

// Selected returns the index of the label the user clicked, -1 for none.
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

// finite reports whether v is drawn.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// legendMax is how many entries a legend shows.
const legendMax = 12

func (c *ChartWidget) build(ctx *ui.Context) ui.Element {
	t := ctx.Theme()
	// The drawing and the selection read these copies: the setters replace
	// the slices, never change them.
	labels, series, selected := c.labels, c.series, c.selected
	pick := -1
	root := ui.Column(ctx).Gap(t.Space(2))
	root.Children(func() {
		if c.kind == PieChart {
			pick = c.buildPie(ctx, labels, series, selected)
		} else {
			pick = c.buildPlot(ctx, labels, series, selected)
		}
		c.buildLegend(ctx, labels, series)
	})
	if pick >= 0 {
		c.selected = pick
		if h := c.onSelect; h != nil {
			Post(h)
		}
	}
	return root
}

func (c *ChartWidget) buildLegend(ctx *ui.Context, labels []string, series []ChartSeries) {
	t := ctx.Theme()
	items := len(series)
	if c.kind == PieChart {
		items = len(labels)
	} else if items < 2 {
		return
	}
	more := max(items-legendMax, 0)
	items = min(items, legendMax)
	ui.Row(ctx).Gap(t.Space(3)).Wrap().Children(func() {
		for i := 0; i < items; i++ {
			name := ""
			if c.kind == PieChart {
				name = labels[i]
			} else {
				name = series[i].Name
			}
			ui.Row(ctx.Key(i)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
				ui.Box(ctx).Size(10, 10).Radius(2).Background(chartColor(i))
				ui.Text(ctx, name).NoWrap().FontSize(t.FontSize - 1)
			})
		}
		if more > 0 {
			ui.Text(ctx, fmt.Sprintf("+%d more", more)).NoWrap().FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		}
	})
}

// extent returns the lowest and highest value drawn, with 0 inside.
func extent(labels []string, series []ChartSeries) (lo, hi float64) {
	for _, s := range series {
		for i, v := range s.Values {
			if i >= len(labels) || !finite(v) {
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

// axisTicks returns about eight of the x axis's ticks, so that a long axis
// costs the same as a short one.
func axisTicks(x chart.Band, labels []string) []chart.Tick {
	if len(labels) <= 8 {
		return x.Ticks()
	}
	step := (len(labels) + 7) / 8
	var out []chart.Tick
	for i := 0; i < len(labels); i += step {
		out = append(out, chart.Tick{At: x.At(i), Value: float64(i), Label: labels[i]})
	}
	return out
}

func (c *ChartWidget) buildPlot(ctx *ui.Context, labels []string, series []ChartSeries, selected int) (picked int) {
	t := ctx.Theme()
	picked = -1
	lo, hi := extent(labels, series)
	y := chart.NewLinear(lo, hi).Nice(4)
	x := chart.NewBand(labels...).Padding(0.25, 0.1)
	if c.kind == LineChart {
		x = chart.NewBand(labels...)
	}
	if len(c.xs) != len(labels) || c.xsLabels != len(labels) {
		c.xs = make([]float32, len(labels))
		for i := range c.xs {
			c.xs[i] = x.At(i)
		}
		c.xsLabels = len(labels)
	}
	xs, kind := c.xs, c.kind
	ticks := y.Ticks(4)
	label := func(side chart.Side, tk chart.Tick) {
		ui.Text(ctx, tk.Label).NoWrap().FontSize(t.FontSize - 2).TextColor(t.TextMuted)
	}
	name := c.title
	if name == "" {
		name = "Chart"
	}
	chart.Plot(ctx, chart.Axes{Left: ticks, Bottom: axisTicks(x, labels), Label: label}, func(area ui.Element) {
		area.Label(name)
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tk := range ticks {
				p.Line(r.X, chart.Y(r, tk.At), r.X+r.W, chart.Y(r, tk.At), 1, t.Border)
			}
			if kind == LineChart {
				c.drawLines(p, r, y, xs, labels, series)
			} else {
				c.drawBars(p, r, x, y, labels, series)
			}
			if selected >= 0 && selected < len(xs) {
				fx := chart.X(r, xs[selected])
				p.Line(fx, r.Y, fx, r.Y+r.H, 1, t.TextMuted)
			}
		})
		describe := func(i int) string {
			var parts []string
			for _, s := range series {
				if i < len(s.Values) && finite(s.Values[i]) {
					n := fmt.Sprintf("%s: %s", labels[i], fmtValue(s.Values[i]))
					if s.Name != "" {
						n = s.Name + ", " + n
					}
					parts = append(parts, n)
				}
			}
			if len(parts) == 0 {
				return labels[i]
			}
			out := parts[0]
			for _, s := range parts[1:] {
				out += "; " + s
			}
			return out
		}
		chart.Select(ctx, area, &c.hover, xs, describe)
		if area.Clicked() && c.hover >= 0 {
			picked = c.hover
		}
	}).Height(220).Gap(6)
	return picked
}

// drawLines paints the lines. It runs outside mu, on the drawing thread, and
// touches only its arguments and the scratch below.
func (c *ChartWidget) drawLines(p *ui.Painter, r ui.Rect, y chart.Linear, xsAll []float32, labels []string, series []ChartSeries) {
	px := int(math.Ceil(float64(r.W * p.Scale())))
	for si, s := range series {
		n := min(len(s.Values), len(labels))
		xs, ys := c.ensureScratch(n)
		for i := 0; i < n; i++ {
			xs[i] = xsAll[i]
			if finite(s.Values[i]) {
				ys[i] = y.At(s.Values[i])
			} else {
				ys[i] = float32(math.NaN())
			}
		}
		c.bx, c.by = chart.MinMax(c.bx, c.by, xs, ys, px)
		c.points = c.points[:0]
		for i, fx := range c.bx {
			c.points = append(c.points, [2]float32{chart.X(r, fx), chart.Y(r, c.by[i])})
		}
		p.StrokePolyline(c.points, 2, chartColor(si))
	}
}

// ensureScratch returns two slices of n floats, reused from frame to frame.
func (c *ChartWidget) ensureScratch(n int) (xs, ys []float32) {
	if cap(c.ysScratch) < 2*n {
		c.ysScratch = make([]float32, 2*n)
	}
	return c.ysScratch[:n], c.ysScratch[n : 2*n]
}

func (c *ChartWidget) drawBars(p *ui.Painter, r ui.Rect, x chart.Band, y chart.Linear, labels []string, series []ChartSeries) {
	n := max(len(series), 1)
	for si, s := range series {
		for i, v := range s.Values {
			if i >= len(labels) || !finite(v) {
				continue
			}
			b := x.Band(i)
			w := (b.Hi - b.Lo) / float32(n)
			sp := y.Span(0, v)
			x0, x1 := chart.X(r, b.Lo+w*float32(si)), chart.X(r, b.Lo+w*float32(si+1))
			y0, y1 := chart.Y(r, sp.Hi), chart.Y(r, sp.Lo)
			p.Fill(ui.Rect{X: x0, Y: min(y0, y1), W: x1 - x0, H: abs32(y1 - y0)}, chartColor(si), 2)
		}
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (c *ChartWidget) buildPie(ctx *ui.Context, labels []string, series []ChartSeries, selected int) (picked int) {
	picked = -1
	var values []float64
	if len(series) > 0 {
		for i, v := range series[0].Values {
			if i < len(labels) {
				if !finite(v) {
					v = 0
				}
				values = append(values, v)
			}
		}
	}
	total := 0.0
	for _, v := range values {
		total += math.Max(v, 0)
	}
	slices := chart.Pie(values, 0.02)
	describe := func(i int) string {
		share := 0.0
		if total > 0 {
			share = math.Max(values[i], 0) / total * 100
		}
		if share > 0 && share < 1 {
			return fmt.Sprintf("%s: %s (%.1f%%)", labels[i], fmtValue(values[i]), share)
		}
		return fmt.Sprintf("%s: %s (%.0f%%)", labels[i], fmtValue(values[i]), share)
	}
	name := c.title
	if name == "" {
		name = "Chart"
	}
	chart.Plot(ctx, chart.Axes{}, func(area ui.Element) {
		area.Label(name)
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			size := min(r.W, r.H)
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			for i, s := range slices {
				outer := size / 2
				if i == selected {
					outer -= 4
				}
				p.FillPath(chart.Sector(cx, cy, size/2*0.55, outer, s), chartColor(i))
			}
		})
		// The slice under the pointer, named as the value of the plot.
		if px, py, over := area.PointerPosition(); over && len(slices) > 0 {
			b := area.Bounds()
			size := min(b.W, b.H)
			if i, ok := chart.SliceAt(slices, px-b.W/2, py-b.H/2, size/2*0.55, size/2); ok && total > 0 {
				d := describe(i)
				area.Value(d).Tooltip(d)
				if area.Clicked() {
					picked = i
				}
			}
		}
	}).Height(200).Width(200)
	return picked
}

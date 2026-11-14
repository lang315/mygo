package widget

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
)

// sentinelColor is a color no chart draws, for the controls around one.
var sentinelColor = ui.Hex("#ff00ff")

// boxChart is a chart of kind k with a legend: two series for lines and
// bars, three slices for a pie.
func boxChart(k ChartKind) *ChartWidget {
	c := Chart(k).SetTitle("Box").SetLabels("Jan", "Feb", "Mar")
	if k == PieChart {
		return c.SetSeries(ChartSeries{Values: []float64{60, 30, 10}})
	}
	return c.SetSeries(ChartSeries{Name: "Search", Values: []float64{10, 20, 30}},
		ChartSeries{Name: "Direct", Values: []float64{30, 5, 15}})
}

// sentinel is a 300 by 40 box of sentinelColor, named for a Tester.
func sentinel(name string) *BoxWidget {
	return VBox(0).SetAccessibleName(name).Background(sentinelColor).SetSize(300, 40)
}

// eachLook runs fn under no skin and Calm, light and dark.
func eachLook(t *testing.T, fn func(t *testing.T, dark bool)) {
	t.Cleanup(func() { skin.Use(nil) })
	for _, s := range []*skin.Skin{nil, skin.Calm} {
		for _, dark := range []bool{false, true} {
			name := "plain"
			if s != nil {
				name = s.Name()
			}
			if dark {
				name += " dark"
			}
			t.Run(name, func(t *testing.T) {
				skin.Use(s)
				fn(t, dark)
			})
		}
	}
}

func samePixel(a, b color.RGBA) bool { return a.R == b.R && a.G == b.G && a.B == b.B }

// checkChartInBand puts the chart mk makes, sized 300 by h, between two
// sentinels and checks that it draws nothing on them, below them or to the
// right of its 300 DIPs; then alone under the top one (checkChartLast).
func checkChartInBand(t *testing.T, name string, mk func() *ChartWidget, h float32, dark bool) {
	t.Helper()
	c := mk().SetSize(300, h)
	tt := NewTester(t, VBox(0).Add(sentinel("top"), c, sentinel("bottom")), 400, 500)
	if dark {
		tt.SetDark(true)
		tt.Settle()
	}
	top, bottom := tt.Find(ByLabel("top")).Bounds(), tt.Find(ByLabel("bottom")).Bounds()
	if got := bottom.Y - (top.Y + top.H); got != h {
		t.Fatalf("the band between the sentinels is %v high, want %v", got, h)
	}
	img := tt.Image()
	bg := img.RGBAAt(img.Bounds().Max.X-1, img.Bounds().Max.Y-1)
	magenta := color.RGBA{R: 0xff, B: 0xff, A: 0xff}
	for _, r := range []ui.Rect{top, bottom} {
		if n := countNotPixel(img, r, magenta); n > 0 {
			t.Errorf("%s, height %v: %d pixels of the sentinel at %v are not its color", name, h, n, r)
		}
	}
	// Nothing to the right of the chart, nor below the bottom sentinel.
	right := ui.Rect{X: 300, Y: 0, W: 100, H: 500}
	below := ui.Rect{X: 0, Y: bottom.Y + bottom.H, W: 400, H: 500 - (bottom.Y + bottom.H)}
	for _, r := range []ui.Rect{right, below} {
		if n := countNotPixel(img, r, bg); n > 0 {
			t.Errorf("%s, height %v: %d pixels in %v differ from the page", name, h, n, r)
		}
	}
	checkChartLast(t, name, mk, h, dark)
}

// checkChartLast puts the chart last, under the top sentinel, where nothing
// paints over what it draws below its box: every row from 40+h down is the
// page's.
func checkChartLast(t *testing.T, name string, mk func() *ChartWidget, h float32, dark bool) {
	t.Helper()
	tt := NewTester(t, VBox(0).Add(sentinel("top"), mk().SetSize(300, h)), 400, 500)
	if dark {
		tt.SetDark(true)
		tt.Settle()
	}
	img := tt.Image()
	bg := img.RGBAAt(img.Bounds().Max.X-1, img.Bounds().Max.Y-1)
	if n := countNotPixel(img, ui.Rect{Y: 40 + h, W: 400, H: 500}, bg); n > 0 {
		t.Errorf("%s last, height %v: %d pixels below its box differ from the page", name, h, n)
	}
}

// kinds are the charts of the box cases: each kind with a legend.
var kinds = []struct {
	name string
	mk   func() *ChartWidget
}{
	{"line", func() *ChartWidget { return boxChart(LineChart) }},
	{"bar", func() *ChartWidget { return boxChart(BarChart) }},
	{"pie", func() *ChartWidget { return boxChart(PieChart) }},
}

// countNotPixel counts the pixels of img inside r that differ from c.
func countNotPixel(img *image.RGBA, r ui.Rect, c color.RGBA) int {
	n := 0
	b := image.Rect(int(r.X), int(r.Y), int(r.X+r.W), int(r.Y+r.H)).Intersect(img.Bounds())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !samePixel(img.RGBAAt(x, y), c) {
				n++
			}
		}
	}
	return n
}

func TestChartDrawsOnlyInItsBox(t *testing.T) {
	eachLook(t, func(t *testing.T, dark bool) {
		for _, k := range kinds {
			checkChartInBand(t, k.name, k.mk, 200, dark)
		}
	})
}

func TestChartSmallBoxStaysInside(t *testing.T) {
	eachLook(t, func(t *testing.T, dark bool) {
		for _, k := range kinds {
			checkChartInBand(t, k.name, k.mk, 120, dark)
		}
	})
}

// Smaller than the plot's 60 DIPs and the legend: the chart clips.
func TestChartTinyBoxClips(t *testing.T) {
	eachLook(t, func(t *testing.T, dark bool) {
		for _, k := range kinds {
			checkChartInBand(t, k.name, k.mk, 70, dark)
		}
	})
}

// In a box of 120 the plot gives up its height, not the legend: both of
// its 10 by 10 swatches show whole, out of the plot.
// (Without a skin: the swatches are plainChartColors.)
func TestChartSmallBoxKeepsItsLegend(t *testing.T) {
	skin.Use(nil)
	for _, k := range kinds[:2] {
		tt := NewTester(t, k.mk().SetSize(300, 120), 400, 300)
		plot := tt.Find(ByLabel("Box")).Bounds()
		img := tt.Image()
		for i := range 2 {
			// 88 for a whole swatch with its rounded corners, 50 or fewer for half.
			if n := countNear(img, plot, true, plainChartColors[i], 6); n < 80 {
				t.Errorf("%s: %d pixels of the swatch of series %d, want 80 or more", k.name, n, i)
			}
		}
	}
}

// Below 60 DIPs and the legend, the plot keeps its 60 and the chart clips
// the rest: the plot is not crushed.
func TestChartPlotKeepsItsMinimumHeight(t *testing.T) {
	for _, k := range kinds {
		tt := NewTester(t, VBox(0).Add(k.mk().SetSize(300, 70)), 400, 300)
		// The pie's plot is its area. The others' runs from the chart's top
		// (y 0) to the bottom of the x axis's labels, whatever the font.
		h := tt.Find(ByLabel("Box")).Bounds().H
		if k.name != "pie" {
			jan := tt.Find(ByLabel("Jan")).Bounds()
			h = jan.Y + jan.H
		}
		if h < 60 {
			t.Errorf("%s: the plot is %v high, want 60", k.name, h)
		}
	}
}

// The top tick's label shows whole, inside the chart's box, as high as the
// bottom one: the clip does not cut it.
func TestChartTopTickLabelIsWhole(t *testing.T) {
	eachLook(t, func(t *testing.T, dark bool) {
		for _, k := range kinds[:2] {
			tt := NewTester(t, VBox(0).Add(sentinel("top"), k.mk().SetSize(300, 200)), 400, 300)
			if dark {
				tt.SetDark(true)
				tt.Settle()
			}
			checkTopLabel(t, tt, k.name)
		}
	})
	// A large font, whose lines are taller than its size: the label's line
	// is still the inset's twice.
	t.Run("font 20", func(t *testing.T) {
		big := ui.LightTheme()
		big.FontSize = 20
		ui.SetDefaultThemes(big, nil)
		t.Cleanup(func() { ui.SetDefaultThemes(nil, nil) })
		for _, k := range kinds[:2] {
			checkTopLabel(t, NewTester(t, VBox(0).Add(sentinel("top"), k.mk().SetSize(300, 200)), 400, 300), k.name)
		}
	})
}

func checkTopLabel(t *testing.T, tt *Tester, name string) {
	t.Helper()
	top, bottom := tt.Find(ByLabel("30")).Bounds(), tt.Find(ByLabel("0")).Bounds()
	if top.Y < 40 || top.H != bottom.H {
		t.Errorf("%s: the top tick's label is at %v, the bottom one's %v; want it under y 40 and as high", name, top, bottom)
	}
}

// A legend that wraps to several lines still stays in a box of 120.
func TestChartWrappedLegendStaysInside(t *testing.T) {
	labels := []string{"January", "February", "March", "April", "May", "June", "July", "August", "September"}
	pie := func() *ChartWidget {
		return Chart(PieChart).SetTitle("Box").SetLabels(labels...).
			SetSeries(ChartSeries{Values: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}})
	}
	bars := func() *ChartWidget {
		var s []ChartSeries
		for i, l := range labels {
			s = append(s, ChartSeries{Name: l, Values: []float64{float64(i + 1), 2}})
		}
		return Chart(BarChart).SetTitle("Box").SetLabels("a", "b").SetSeries(s...)
	}
	eachLook(t, func(t *testing.T, dark bool) {
		checkChartInBand(t, "pie of nine", pie, 120, dark)
		checkChartInBand(t, "bars of nine", bars, 120, dark)
	})
}

// A pie in a box narrower than its 200 DIPs keeps to the box.
func TestChartPieNarrowBoxStaysInside(t *testing.T) {
	skin.Use(nil)
	c := boxChart(PieChart).SetSize(160, 200)
	tt := NewTester(t, VBox(0).Add(c), 400, 500)
	if b := tt.Find(ByLabel("Box")).Bounds(); b.X+b.W > 160 {
		t.Fatalf("the pie's plot %v reaches past the chart's 160 DIPs", b)
	}
	img := tt.Image()
	bg := img.RGBAAt(img.Bounds().Max.X-1, img.Bounds().Max.Y-1)
	if n := countNotPixel(img, ui.Rect{X: 160, W: 240, H: 500}, bg); n > 0 {
		t.Fatalf("%d pixels to the right of the 160 DIPs differ from the page", n)
	}
	// The donut is whole: as far from the box's left as from its right, not
	// cut by the chart's clip. (Without a skin: its slices are
	// plainChartColors.)
	plot := tt.Find(ByLabel("Box")).Bounds()
	lo, hi := 400, -1
	for y := int(plot.Y); y < int(plot.Y+plot.H); y++ {
		for x := 0; x < 400; x++ {
			for _, c := range plainChartColors[:3] {
				if nearTol(img.RGBAAt(x, y), c, 6) {
					lo, hi = min(lo, x), max(hi, x+1)
				}
			}
		}
	}
	if hi < 0 || lo-0 > 160-hi+2 || 160-hi > lo+2 {
		t.Fatalf("the donut spans x %d to %d, want it centered in the 160 DIPs", lo, hi)
	}
}

// gridRows returns the topmost and bottommost rows of the plot r that hold
// a grid line: a row of which nine pixels in ten differ from the page.
func gridRows(img *image.RGBA, r ui.Rect, bg color.RGBA) (top, bottom int) {
	top, bottom = -1, -1
	for y := int(r.Y) - 1; y <= int(r.Y+r.H); y++ {
		n, w := 0, 0
		for x := int(r.X); x < int(r.X+r.W); x++ {
			w++
			if !samePixel(img.RGBAAt(x, y), bg) {
				n++
			}
		}
		if w > 0 && n*10 >= w*9 {
			if top < 0 {
				top = y
			}
			bottom = y
		}
	}
	return top, bottom
}

func TestChartPlotFillsItsHeight(t *testing.T) {
	c := Chart(LineChart).SetTitle("Tall").SetLabels("a", "b", "c").
		SetSeries(ChartSeries{Name: "s", Values: []float64{1, 3, 2}}).SetSize(300, 300)
	tt := NewTester(t, c, 400, 400)
	plot := tt.Find(ByLabel("Tall")).Bounds()
	img := tt.Image()
	bg := img.RGBAAt(img.Bounds().Max.X-1, img.Bounds().Max.Y-1)
	top, bottom := gridRows(img, plot, bg)
	if top < 0 || bottom-top <= 220 {
		t.Fatalf("the grid lines span rows %d to %d (%d DIPs), want more than 220; plot %v", top, bottom, bottom-top, plot)
	}
	if bottom >= 300 {
		t.Fatalf("the grid reaches row %d, out of the chart's 300", bottom)
	}
}

// A chart with no size keeps the 220 DIPs of plot it had.
func TestChartWithNoSizeKeepsItsHeight(t *testing.T) {
	c := Chart(LineChart).SetTitle("Free").SetLabels("a", "b").
		SetSeries(ChartSeries{Name: "s", Values: []float64{1, 2}})
	tt := NewTester(t, VBox(0).Add(c, sentinel("after")), 400, 500)
	// One series: no legend, the plot alone.
	if y := tt.Find(ByLabel("after")).Bounds().Y; y != 220 {
		t.Fatalf("a chart with no size is %v high, want 220", y)
	}
}

// paintedSpan returns the first and past-the-last x of row y within 2 of
// c: the extent of a box filled with c, whatever its contents cover.
func paintedSpan(img *image.RGBA, y int, c ui.Color) (lo, hi int) {
	lo, hi = -1, -1
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		if nearTol(img.RGBAAt(x, y), c, 2) {
			if lo < 0 {
				lo = x
			}
			hi = x + 1
		}
	}
	return lo, hi
}

// chartBacks are backgrounds no chart draws, to read each chart's box.
var chartBacks = []ui.Color{ui.Hex("#ff00ff"), ui.Hex("#00ffff"), ui.Hex("#ffff00")}

func TestChartHasAMinimumWidth(t *testing.T) {
	var charts []Control
	for i, k := range []ChartKind{LineChart, BarChart, PieChart} {
		charts = append(charts, boxChart(k).SetTitle(fmt.Sprint("plot", i)).SetSize(0, 200).Background(chartBacks[i]))
	}
	tt := NewTester(t, HBox(12).Add(charts...), 1200, 400)
	img := tt.Image()
	for i := range charts {
		// Row 1 is the top inset of the plot, where the chart draws little.
		if lo, hi := paintedSpan(img, 1, chartBacks[i]); lo < 0 || hi-lo < 160 {
			t.Errorf("chart %d is %d wide (x %d to %d), want 160 or more", i, hi-lo, lo, hi)
		}
	}
}

// An app's own width wins over the minimum, narrower as wider.
func TestChartOwnWidthWins(t *testing.T) {
	for _, c := range []struct {
		name     string
		w        *ChartWidget
		min, max int
	}{
		{"MinWidth(240)", boxChart(LineChart).SetSize(0, 200).Apply(NewStyle().MinWidth(240)), 240, 240},
		// Its own minimum replaces the 160: the chart is its content's width.
		{"MinWidth(100)", boxChart(LineChart).SetSize(0, 200).Apply(NewStyle().MinWidth(100)), 100, 159},
		{"SetSize(100, 200)", boxChart(LineChart).SetSize(100, 200), 100, 100},
	} {
		tt := NewTester(t, HBox(12).Add(c.w.Background(chartBacks[0])), 1200, 400)
		if lo, hi := paintedSpan(tt.Image(), 1, chartBacks[0]); hi-lo < c.min || hi-lo > c.max {
			t.Errorf("%s: the chart is %d wide (x %d to %d), want %d to %d", c.name, hi-lo, lo, hi, c.min, c.max)
		}
	}
}

// The chart clips to its box, so the plot rings itself inside: Tab to it
// and its right edge (the chart's) changes along most of its height.
func TestChartFocusRingIsInsideTheClip(t *testing.T) {
	eachLook(t, func(t *testing.T, dark bool) { checkRingInside(t, dark) })
}

func checkRingInside(t *testing.T, dark bool) {
	for _, k := range kinds {
		tt := NewTester(t, VBox(0).Add(k.mk().SetSize(300, 200)), 400, 300)
		if dark {
			tt.SetDark(true)
			tt.Settle()
		}
		plot := tt.Find(ByLabel("Box")).Bounds()
		before := tt.Image()
		tt.Key(0, ui.KeyTab)
		tt.Settle()
		if !tt.Find(ByLabel("Box")).Focused() {
			t.Fatalf("%s: Tab did not focus the plot", k.name)
		}
		after := tt.Image()
		// The pie's plot is 200 wide, at the chart's left: ring its left edge.
		x := int(plot.X + plot.W - 1)
		if k.name == "pie" {
			x = int(plot.X)
		}
		changed := 0
		for y := int(plot.Y); y < int(plot.Y+plot.H); y++ {
			if !samePixel(before.RGBAAt(x, y), after.RGBAAt(x, y)) {
				changed++
			}
		}
		if float32(changed) < plot.H*0.8 {
			t.Errorf("%s: the ring changed %d of %v rows of the plot's edge at x %d", k.name, changed, plot.H, x)
		}
	}
}

func TestBarChartDrawsBars(t *testing.T) {
	skin.Use(nil)
	c := barChart().SetSize(300, 200)
	tt := NewTester(t, c, 400, 300)
	plot := tt.Find(ByLabel("Sales")).Bounds()
	img := tt.Image()
	y := int(plot.Y+plot.H) - 10
	runs, in := 0, false
	for x := int(plot.X); x < int(plot.X+plot.W); x++ {
		on := nearTol(img.RGBAAt(x, y), plainChartColors[0], 6)
		if on && !in {
			runs++
		}
		in = on
	}
	if runs < 3 {
		t.Fatalf("%d runs of the series color along row %d, want 3 or more", runs, y)
	}
}

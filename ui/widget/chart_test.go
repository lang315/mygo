package widget

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

func barChart() *ChartWidget {
	return Chart(BarChart).SetTitle("Sales").SetLabels("Jan", "Feb", "Mar").
		SetSeries(ChartSeries{Name: "2026", Values: []float64{10, 20, 30}})
}

// plotValues moves the pointer across the plot and returns the value
// (the description of the point under it) at each of n positions, and the
// x of each.
func plotValues(tt *Tester, name string, n int) (vals []string, xs []float32) {
	for i := 0; i < n; i++ {
		b := tt.Find(ByLabel(name)).Bounds()
		x := b.X + b.W*(float32(i)+0.5)/float32(n)
		tt.Move(x, b.Y+b.H/2)
		tt.Frame()
		vals = append(vals, tt.Find(ByLabel(name)).Value())
		xs = append(xs, x)
	}
	return vals, xs
}

func distinct(vals []string) map[string]bool {
	m := map[string]bool{}
	for _, v := range vals {
		if v != "" {
			m[v] = true
		}
	}
	return m
}

func TestBarChartNamesEachBarByItsDataUnderThePointer(t *testing.T) {
	tt := NewTester(t, barChart(), 500, 300)
	for _, s := range []string{"Jan", "Feb", "Mar"} {
		if !tt.Has(ByLabel(s)) {
			t.Errorf("no axis label %q in %q", s, tt.Texts())
		}
	}
	vals, _ := plotValues(tt, "Sales", 30)
	got := distinct(vals)
	for _, want := range []string{"2026, Jan: 10", "2026, Feb: 20", "2026, Mar: 30"} {
		if !got[want] {
			t.Errorf("no point named %q among %v", want, got)
		}
	}
}

func TestChartFollowsItsDataThroughSetters(t *testing.T) {
	c := barChart()
	tt := NewTester(t, c, 500, 300)
	c.SetSeries(ChartSeries{Name: "2026", Values: []float64{30, 20, 10}})
	tt.Settle()
	vals, _ := plotValues(tt, "Sales", 30)
	if got := distinct(vals); !got["2026, Jan: 30"] || got["2026, Jan: 10"] {
		t.Fatalf("the new data is not what the chart reports: %v", got)
	}
	c.SetLabels("A", "B", "C", "D").SetSeries(ChartSeries{Values: []float64{1, 2, 3, 4}})
	tt.Settle()
	vals, _ = plotValues(tt, "Sales", 40)
	if got := distinct(vals); !got["D: 4"] || !tt.Has(ByLabel("D")) {
		t.Fatalf("after growing to four: %v", got)
	}
}

func TestChartSeriesGetALegendAndAllReportInThePointsName(t *testing.T) {
	c := Chart(BarChart).SetLabels("Q1", "Q2").SetSeries(
		ChartSeries{Name: "Search", Values: []float64{5, 6}},
		ChartSeries{Name: "Direct", Values: []float64{7, 8}})
	tt := NewTester(t, c, 500, 300)
	if !tt.HasText("Search") || !tt.HasText("Direct") {
		t.Fatalf("no legend: %q", tt.Texts())
	}
	vals, _ := plotValues(tt, "Chart", 20)
	if got := distinct(vals); !got["Search, Q1: 5; Direct, Q1: 7"] {
		t.Fatalf("a point reports all its series: %v", got)
	}
}

func TestChartClickingABarSelectsItsLabelOnce(t *testing.T) {
	got := make(chan int, 4)
	var c *ChartWidget
	c = barChart().OnSelect(func() { got <- c.Selected() })
	tt := NewTester(t, c, 500, 300)
	if c.Selected() != -1 {
		t.Fatalf("Selected %d before any click", c.Selected())
	}
	vals, xs := plotValues(tt, "Sales", 30)
	var x float32
	for i, v := range vals {
		if v == "2026, Feb: 20" {
			x = xs[i]
			break
		}
	}
	b := tt.Find(ByLabel("Sales")).Bounds()
	tt.ClickAt(x, b.Y+b.H/2)
	tt.Settle()
	if i := received(t, got); i != 1 || c.Selected() != 1 {
		t.Fatalf("OnSelect saw %d, Selected %d, want 1", i, c.Selected())
	}
	for range 4 {
		tt.Frame()
	}
	tt.Settle()
	select {
	case <-got:
		t.Fatal("OnSelect ran again without a click")
	default:
	}
}

func TestPieChartNamesSlicesWithTheirShareAndHasALegend(t *testing.T) {
	c := Chart(PieChart).SetTitle("Users").SetLabels("Win", "Mac", "Linux").
		SetSeries(ChartSeries{Values: []float64{60, 30, 10}})
	tt := NewTester(t, c, 400, 300)
	for _, s := range []string{"Win", "Mac", "Linux"} {
		if !tt.HasText(s) {
			t.Errorf("no legend entry %q in %q", s, tt.Texts())
		}
	}
	b := tt.Find(ByLabel("Users")).Bounds()
	seen := map[string]bool{}
	for a := 0.0; a < 2*math.Pi; a += 0.25 {
		tt.Move(b.X+b.W/2+float32(math.Sin(a))*0.4*b.W, b.Y+b.H/2-float32(math.Cos(a))*0.4*b.H)
		tt.Frame()
		seen[tt.Find(ByLabel("Users")).Value()] = true
	}
	if !seen["Win: 60 (60%)"] || !seen["Mac: 30 (30%)"] || !seen["Linux: 10 (10%)"] {
		t.Fatalf("slices seen: %v", seen)
	}
}

func TestClickingASliceSelectsItsLabel(t *testing.T) {
	c := Chart(PieChart).SetTitle("Users").SetLabels("Win", "Mac", "Linux").
		SetSeries(ChartSeries{Values: []float64{60, 30, 10}})
	tt := NewTester(t, c, 400, 300)
	b := tt.Find(ByLabel("Users")).Bounds()
	// Just left of twelve o'clock is the last slice (Linux).
	tt.ClickAt(b.X+b.W/2-0.05*b.W, b.Y+b.H/2-0.4*b.H)
	tt.Settle()
	if c.Selected() != 2 {
		t.Fatalf("Selected %d after clicking the last slice", c.Selected())
	}
}

func TestLineChartReportsPointsAndHasAxis(t *testing.T) {
	c := Chart(LineChart).SetLabels("Mon", "Tue", "Wed").
		SetSeries(ChartSeries{Name: "Temp", Values: []float64{12, 18, 15}})
	tt := NewTester(t, c, 500, 300)
	if !tt.Has(ByLabel("Tue")) {
		t.Fatalf("no axis: %q", tt.Texts())
	}
	vals, _ := plotValues(tt, "Chart", 30)
	if got := distinct(vals); !got["Temp, Tue: 18"] {
		t.Fatalf("points: %v", got)
	}
}

func TestChartWithNoDataShowsAnEmptyChartAndDoesNotPanic(t *testing.T) {
	for _, k := range []ChartKind{LineChart, BarChart, PieChart} {
		c := Chart(k)
		NewTester(t, c, 300, 200)
		c.SetLabels("a", "b").SetSeries(ChartSeries{Values: []float64{0, 0}})
		NewTester(t, c, 300, 200)
	}
}

func TestChartLeavesOutNaNAndInfinity(t *testing.T) {
	c := Chart(BarChart).SetLabels("a", "b", "c", "d").
		SetSeries(ChartSeries{Values: []float64{1, math.NaN(), math.Inf(1), 4}})
	tt := NewTester(t, c, 400, 300)
	vals, _ := plotValues(tt, "Chart", 40)
	for v := range distinct(vals) {
		if strings.Contains(v, "Inf") || strings.Contains(v, "NaN") {
			t.Errorf("a point is named %q", v)
		}
	}
	if got := distinct(vals); !got["a: 1"] || !got["d: 4"] {
		t.Fatalf("points: %v", got)
	}
	for _, k := range []ChartKind{LineChart, PieChart} {
		c := Chart(k).SetLabels("a", "b", "c").SetSeries(ChartSeries{Values: []float64{1, math.Inf(-1), math.NaN()}})
		NewTester(t, c, 300, 200).Frame()
	}
}

func TestChartSetSeriesClearsASelectionBeyondTheData(t *testing.T) {
	fired := make(chan struct{}, 4)
	c := barChart().OnSelect(func() { fired <- struct{}{} })
	tt := NewTester(t, c, 500, 300)
	_, xs := plotValues(tt, "Sales", 30)
	b := tt.Find(ByLabel("Sales")).Bounds()
	tt.ClickAt(xs[len(xs)-2], b.Y+b.H/2) // the last bar
	tt.Settle()
	received(t, fired)
	if c.Selected() != 2 {
		t.Fatalf("Selected %d", c.Selected())
	}
	c.SetSeries(ChartSeries{Values: []float64{5}})
	if c.Selected() != -1 {
		t.Fatalf("Selected %d with one value left", c.Selected())
	}
	tt.Settle()
	select {
	case <-fired:
		t.Fatal("SetSeries ran OnSelect")
	default:
	}
}

func TestChartSetLabelsDropsASelectionBeyondThem(t *testing.T) {
	c := barChart()
	tt := NewTester(t, c, 500, 300)
	_, xs := plotValues(tt, "Sales", 30)
	b := tt.Find(ByLabel("Sales")).Bounds()
	tt.ClickAt(xs[len(xs)-2], b.Y+b.H/2)
	tt.Settle()
	c.SetLabels("Jan")
	if c.Selected() != -1 {
		t.Fatalf("Selected %d with one label left", c.Selected())
	}
}

// A worker sets the data while the chart paints: no data race.
func TestChartPaintAndSettersFromAWorkerDoNotRace(t *testing.T) {
	c := Chart(LineChart).SetLabels("a", "b", "c").SetSeries(ChartSeries{Values: []float64{1, 2, 3}})
	tt := NewTester(t, c, 400, 300)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			c.SetLabels("a", "b", "c", "d").SetSeries(ChartSeries{Values: []float64{float64(i), 2, 3, 4}})
		}
	}()
	for i := 0; i < 100; i++ {
		tt.Frame()
		tt.Image()
	}
	wg.Wait()
}

// The chart colors of today, which a chart without a skin keeps.
var todayChartColors = []ui.Color{
	ui.Hex("#2563eb"), ui.Hex("#16a34a"), ui.Hex("#ea580c"), ui.Hex("#9333ea"),
	ui.Hex("#0891b2"), ui.Hex("#db2777"), ui.Hex("#ca8a04"), ui.Hex("#4b5563"),
}

// nineSeries is a chart of kind k with nine series (bars, lines) or nine
// slices (pie), so that the ninth takes the first color again.
func nineSeries(k ChartKind) *ChartWidget {
	c := Chart(k).SetTitle("Nine")
	if k == PieChart {
		return c.SetLabels("a", "b", "c", "d", "e", "f", "g", "h", "i").
			SetSeries(ChartSeries{Values: []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}})
	}
	var s []ChartSeries
	for i := range 9 {
		s = append(s, ChartSeries{Name: fmt.Sprint("s", i), Values: []float64{float64(i + 1), float64(i + 1)}})
	}
	return c.SetLabels("x", "y").SetSeries(s...)
}

// A chart with a skin draws its series, slices and legend in the skin's
// chart colors, in order, the ninth in the first again, light and dark;
// without one, in today's colors.
func TestChartUsesSkinColors(t *testing.T) {
	if !slices.Equal(plainChartColors, todayChartColors) {
		t.Fatalf("plainChartColors is %v, want today's %v", plainChartColors, todayChartColors)
	}
	light, dark := testTokens()
	for _, k := range []ChartKind{BarChart, LineChart, PieChart} {
		for _, c := range []struct {
			name   string
			skin   bool
			dark   bool
			colors []ui.Color
		}{
			{"plain", false, false, todayChartColors},
			{"plain dark", false, true, todayChartColors},
			{"light", true, false, light.Chart[:]},
			{"dark", true, true, dark.Chart[:]},
		} {
			if c.skin {
				useTestSkin(t)
			} else {
				skinhook.Set(nil, nil)
			}
			tt := NewTester(t, nineSeries(k), 600, 420)
			if c.dark {
				tt.SetDark(true)
				tt.Settle()
			}
			plot := tt.Find(ByLabel("Nine")).Bounds()
			img := tt.Image()
			for i, col := range c.colors {
				// The plot: bars and slices are filled, lines 2 wide.
				if n := countNear(img, plot, false, col, 6); n < 20 {
					t.Errorf("kind %d, %s: %d pixels of color %d (%v) in the plot, want 20 or more", k, c.name, n, i, col)
				}
				// The legend, under the plot: a 10 by 10 box an entry, two of the
				// first color.
				min := 50
				if i == 0 {
					min = 150
				}
				if n := countNear(img, plot, true, col, 6); n < min {
					t.Errorf("kind %d, %s: %d pixels of color %d (%v) in the legend, want %d or more", k, c.name, n, i, col, min)
				}
			}
			// None of the other set's colors, whichever skin is on.
			others := [][]ui.Color{todayChartColors, light.Chart[:], dark.Chart[:]}
			for _, o := range others {
				if slices.Equal(o, c.colors) {
					continue
				}
				for i, col := range o {
					if slices.Contains(c.colors, col) {
						continue
					}
					if n := countNear(img, ui.Rect{}, true, col, 2); n > 4 {
						t.Errorf("kind %d, %s: %d pixels of the color %v (index %d of another set)", k, c.name, n, col, i)
					}
				}
			}
		}
	}
	skinhook.Set(nil, nil)
}

// The series take the skin's colors in order: with three series, colors 0
// to 2 in the plot and no other; bars left to right, lines (values 1, 2, 3)
// bottom to top, slices (values 1, 2, 3) smallest to largest.
func TestChartDrawsSkinColorsInOrder(t *testing.T) {
	light, _ := useTestSkin(t)
	for _, k := range []ChartKind{BarChart, LineChart, PieChart} {
		c := Chart(k).SetTitle("Three")
		if k == PieChart {
			c.SetLabels("a", "b", "c").SetSeries(ChartSeries{Values: []float64{1, 2, 3}})
		} else {
			c.SetLabels("x", "y").SetSeries(ChartSeries{Name: "s0", Values: []float64{1, 1}},
				ChartSeries{Name: "s1", Values: []float64{2, 2}}, ChartSeries{Name: "s2", Values: []float64{3, 3}})
		}
		tt := NewTester(t, c, 600, 420)
		plot := tt.Find(ByLabel("Three")).Bounds()
		img := tt.Image()
		var n [3]int
		var mx, my [3]float64
		for y := int(plot.Y); y < int(plot.Y+plot.H); y++ {
			for x := int(plot.X); x < int(plot.X+plot.W); x++ {
				for i := range 3 {
					if nearTol(img.RGBAAt(x, y), light.Chart[i], 6) {
						n[i]++
						mx[i] += float64(x)
						my[i] += float64(y)
					}
				}
			}
		}
		for i := range 3 {
			if n[i] < 20 {
				t.Fatalf("kind %d: %d pixels of color %d in the plot, want 20 or more", k, n[i], i)
			}
			mx[i], my[i] = mx[i]/float64(n[i]), my[i]/float64(n[i])
		}
		for i := 3; i < 8; i++ {
			if m := countNear(img, plot, false, light.Chart[i], 6); m > 0 {
				t.Errorf("kind %d: %d pixels of color %d in a plot of three series", k, m, i)
			}
		}
		switch k {
		case BarChart:
			if !(mx[0] < mx[1] && mx[1] < mx[2]) {
				t.Errorf("bars: the mean x of colors 0, 1, 2 is %v, want rising", mx)
			}
		case LineChart:
			if !(my[0] > my[1] && my[1] > my[2]) {
				t.Errorf("lines: the mean y of colors 0, 1, 2 is %v, want falling (values 1, 2, 3)", my)
			}
		case PieChart:
			if !(n[0] < n[1] && n[1] < n[2]) {
				t.Errorf("pie: colors 0, 1, 2 cover %v pixels, want rising (values 1, 2, 3)", n)
			}
		}
	}
}

func linePoints(n int) *ChartWidget {
	labels := make([]string, n)
	vals := make([]float64, n)
	for i := range labels {
		labels[i] = fmt.Sprint("p", i)
		vals[i] = float64(i % 17)
	}
	return Chart(LineChart).SetLabels(labels...).SetSeries(ChartSeries{Name: "s", Values: vals})
}

// An idle frame costs the same whatever the number of points.
func TestChartFrameAllocationsDoNotGrowWithThePoints(t *testing.T) {
	cost := func(n int) float64 {
		tt := NewTester(t, linePoints(n), 600, 300)
		tt.Frame()
		return allocrun.Least(20, func() { tt.Frame(); tt.Image() })
	}
	small, big := cost(10), cost(10000)
	t.Logf("allocs per frame: %v with 10 points, %v with 10,000", small, big)
	if big > small*2+50 {
		t.Fatalf("allocations per frame: %v with 10 points, %v with 10,000", small, big)
	}
}

func TestPieOfAThousandSlicesKeepsTheLegendShortAndShowsSmallShares(t *testing.T) {
	labels := make([]string, 1000)
	vals := make([]float64, 1000)
	for i := range labels {
		labels[i], vals[i] = fmt.Sprint("s", i), 1
	}
	vals[0] = 1000
	c := Chart(PieChart).SetTitle("P").SetLabels(labels...).SetSeries(ChartSeries{Values: vals})
	tt := NewTester(t, c, 400, 400)
	if !tt.HasText("+988 more") {
		t.Errorf("legend not capped: %d texts", len(tt.Texts()))
	}
	b := tt.Find(ByLabel("P")).Bounds()
	seen := map[string]bool{}
	for a := 0.0; a < 2*math.Pi; a += 0.01 {
		tt.Move(b.X+b.W/2+float32(math.Sin(a))*0.4*b.W, b.Y+b.H/2-float32(math.Cos(a))*0.4*b.H)
		tt.Frame()
		seen[tt.Find(ByLabel("P")).Value()] = true
	}
	small := false
	for v := range seen {
		if strings.HasSuffix(v, "(0.0%)") {
			t.Errorf("a share rounded to zero: %q", v)
		}
		if strings.HasSuffix(v, "(0.1%)") || strings.HasSuffix(v, "(0.2%)") || strings.HasSuffix(v, "(0.3%)") {
			small = true
		}
	}
	if !small {
		t.Errorf("no small share with a decimal among %d seen", len(seen))
	}
}

func TestPieTakesTheKeyboard(t *testing.T) {
	got := make(chan int, 4)
	var c *ChartWidget
	c = Chart(PieChart).SetTitle("Users").SetLabels("Win", "Mac", "Linux").
		SetSeries(ChartSeries{Values: []float64{60, 30, 10}}).
		OnSelect(func() { got <- c.Selected() })
	tt := NewTester(t, c, 400, 300)
	plot := tt.Find(ByLabel("Users")).Focus() // fails if the plot cannot take the focus
	plot.Key(0, ui.KeyRight)
	if v := plot.Value(); v != "Win: 60 (60%)" {
		t.Fatalf("after Right the plot says %q", v)
	}
	plot.Key(0, ui.KeyRight)
	if v := plot.Value(); v != "Mac: 30 (30%)" {
		t.Fatalf("after a second Right: %q", v)
	}
	plot.Key(0, ui.KeyEnd)
	if v := plot.Value(); v != "Linux: 10 (10%)" {
		t.Fatalf("after End: %q", v)
	}
	plot.Key(0, ui.KeyEnter)
	if i := received(t, got); i != 2 || c.Selected() != 2 {
		t.Fatalf("Enter selected %d (Selected %d), want 2", i, c.Selected())
	}
	plot.Key(0, ui.KeyHome).Key(0, ui.KeySpace)
	if i := received(t, got); i != 0 {
		t.Fatalf("Space selected %d, want 0", i)
	}
	plot.Key(0, ui.KeyEscape)
	if v := plot.Value(); v != "" {
		t.Fatalf("Escape left %q", v)
	}
}

func TestPieHoverEndsWhenThePointerLeavesAndEnterActsOnTheKeys(t *testing.T) {
	fired := make(chan int, 4)
	var c *ChartWidget
	c = Chart(PieChart).SetTitle("Users").SetLabels("Win", "Mac", "Linux").
		SetSeries(ChartSeries{Values: []float64{60, 30, 10}}).
		OnSelect(func() { fired <- c.Selected() })
	tt := NewTester(t, c, 400, 300)
	b := tt.Find(ByLabel("Users")).Bounds()
	// Over the Win slice (just right of twelve o'clock).
	tt.Move(b.X+b.W/2+0.1*b.W, b.Y+b.H/2-0.4*b.H)
	tt.Frame()
	if v := tt.Find(ByLabel("Users")).Value(); v != "Win: 60 (60%)" {
		t.Fatalf("hovering says %q", v)
	}
	tt.Move(b.X+b.W+50, b.Y+b.H+50) // out of the plot
	tt.Frame()
	if v := tt.Find(ByLabel("Users")).Value(); v != "" {
		t.Fatalf("after the pointer left the plot says %q", v)
	}
	tt.Find(ByLabel("Users")).Focus().Key(0, ui.KeyEnter)
	tt.Settle()
	select {
	case i := <-fired:
		t.Fatalf("Enter selected %d from a stale hover", i)
	default:
	}
}

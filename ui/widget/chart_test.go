package widget

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
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

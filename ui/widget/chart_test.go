package widget

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func barChart() *ChartWidget {
	return Chart(BarChart).SetTitle("Sales").SetLabels("Jan", "Feb", "Mar").
		SetSeries(ChartSeries{Name: "2026", Values: []float64{10, 20, 30}})
}

func TestBarChartDrawsOneBarPerValueWithItsAxisLabels(t *testing.T) {
	tt := NewTester(t, barChart(), 500, 300)
	for _, s := range []string{"Jan", "Feb", "Mar"} {
		if !tt.Has(ByLabel(s)) {
			t.Errorf("no axis label %q in %q", s, tt.Texts())
		}
	}
	bars := tt.FindAll(ByRole(ui.RoleButton))
	if len(bars) != 3 {
		t.Fatalf("%d bars, want 3", len(bars))
	}
	if h0, h2 := bars[0].Bounds().H, bars[2].Bounds().H; !(h2 > h0*2.5 && h2 < h0*3.5) {
		t.Errorf("heights %v and %v: 30 should be three times 10", h0, h2)
	}
	if !tt.Has(ByLabel("Jan: 10")) && !tt.Has(ByLabel("2026, Jan: 10")) {
		t.Errorf("bars are not named by their data: %+v", tt.Nodes())
	}
}

func TestChartFollowsItsDataThroughSetters(t *testing.T) {
	c := barChart()
	tt := NewTester(t, c, 500, 300)
	before := tt.FindAll(ByRole(ui.RoleButton))[0].Bounds().H
	c.SetSeries(ChartSeries{Name: "2026", Values: []float64{30, 20, 10}})
	tt.Settle()
	after := tt.FindAll(ByRole(ui.RoleButton))[0].Bounds().H
	if !(after > before*2.5) {
		t.Fatalf("first bar %v -> %v: the new data is not drawn", before, after)
	}
	c.SetLabels("A", "B", "C", "D").SetSeries(ChartSeries{Values: []float64{1, 2, 3, 4}})
	tt.Settle()
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 4 || !tt.Has(ByLabel("D")) {
		t.Fatalf("%d bars after growing to four", n)
	}
}

func TestChartSeriesGetTheirOwnBarsAndALegend(t *testing.T) {
	c := Chart(BarChart).SetLabels("Q1", "Q2").SetSeries(
		ChartSeries{Name: "Search", Values: []float64{5, 6}},
		ChartSeries{Name: "Direct", Values: []float64{7, 8}})
	tt := NewTester(t, c, 500, 300)
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 4 {
		t.Fatalf("%d bars, want 2 series x 2 labels", n)
	}
	if !tt.Has(ByLabel("Search")) || !tt.Has(ByLabel("Direct")) {
		t.Fatalf("no legend: %q", tt.Texts())
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
	tt.FindAll(ByRole(ui.RoleButton))[1].Click()
	if i := received(t, got); i != 1 || c.Selected() != 1 {
		t.Fatalf("OnSelect saw %d, Selected %d, want 1", i, c.Selected())
	}
}

func TestPieChartHasOneSliceAndLegendEntryPerLabel(t *testing.T) {
	c := Chart(PieChart).SetLabels("Win", "Mac", "Linux").
		SetSeries(ChartSeries{Values: []float64{60, 30, 10}})
	tt := NewTester(t, c, 400, 300)
	for _, s := range []string{"Win", "Mac", "Linux"} {
		if !tt.HasText(s) {
			t.Errorf("no legend entry %q in %q", s, tt.Texts())
		}
	}
	if !tt.Has(ByLabel("Win: 60 (60%)")) || !tt.HasText("Win") {
		t.Errorf("slices are not named with their share: %+v", tt.Nodes())
	}
}

func TestLineChartIsDrawnWithItsPointsNamed(t *testing.T) {
	c := Chart(LineChart).SetLabels("Mon", "Tue", "Wed").
		SetSeries(ChartSeries{Name: "Temp", Values: []float64{12, 18, 15}})
	tt := NewTester(t, c, 500, 300)
	if !tt.Has(ByLabel("Tue")) {
		t.Fatalf("no axis: %q", tt.Texts())
	}
	if !tt.Has(ByLabel("Temp, Tue: 18")) && !tt.Has(ByLabel("Tue: 18")) {
		t.Fatalf("points are not named by their data: %+v", tt.Nodes())
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

func TestChartIgnoresNaNAndMissingValues(t *testing.T) {
	c := Chart(BarChart).SetLabels("a", "b", "c").SetSeries(ChartSeries{Values: []float64{1, math.NaN(), 3}})
	tt := NewTester(t, c, 400, 300)
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 2 {
		t.Fatalf("%d bars for one NaN value, want 2", n)
	}
}

func TestClickingASliceOrAPointSelectsItsLabel(t *testing.T) {
	for name, c := range map[string]*ChartWidget{
		"pie":  Chart(PieChart).SetLabels("Win", "Mac", "Linux").SetSeries(ChartSeries{Values: []float64{60, 30, 10}}),
		"line": Chart(LineChart).SetLabels("Mon", "Tue", "Wed").SetSeries(ChartSeries{Values: []float64{12, 18, 15}}),
	} {
		tt := NewTester(t, c, 500, 400)
		marks := tt.FindAll(ByRole(ui.RoleButton))
		if len(marks) != 3 {
			t.Fatalf("%s: %d marks", name, len(marks))
		}
		marks[2].Click()
		if c.Selected() != 2 {
			t.Errorf("%s: Selected %d after clicking the third mark", name, c.Selected())
		}
	}
}

func TestChartOnSelectRunsOncePerClickNotPerFrame(t *testing.T) {
	n := make(chan struct{}, 8)
	c := barChart().OnSelect(func() { n <- struct{}{} })
	tt := NewTester(t, c, 500, 300)
	tt.FindAll(ByRole(ui.RoleButton))[0].Click()
	for range 5 {
		tt.Frame()
	}
	tt.Settle()
	received(t, n)
	select {
	case <-n:
		t.Fatal("OnSelect ran again without a click")
	default:
	}
}

func TestChartSetLabelsDropsASelectionBeyondThem(t *testing.T) {
	c := barChart()
	tt := NewTester(t, c, 500, 300)
	tt.FindAll(ByRole(ui.RoleButton))[2].Click()
	c.SetLabels("Jan")
	if c.Selected() != -1 {
		t.Fatalf("Selected %d with one label left", c.Selected())
	}
}

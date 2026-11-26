package widget

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestValueControlsFromGoroutine sets the value controls from other
// goroutines while their window draws: the race detector watches it, and
// the next frame shows each value.
func TestValueControlsFromGoroutine(t *testing.T) {
	r := RangeTrackBar(0, 100)
	g := Rating(5)
	b := CheckButton("Bold")
	s := Segmented("List", "Grid")
	sp := SplitButton("Save", MenuItem{Text: "Save As…"})
	tt := NewTester(t, VBox(8).Add(r, g, b, s, sp), 600, 400)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.SetValues(float64(i), 100-float64(i)).SetStep(float64(i % 2))
			g.SetValue(i % 6)
			b.SetChecked(i%2 == 0)
			s.SetSelected(i % 2)
			sp.SetIcon(nil).OnClick(func() {})
		}()
		tt.Frame()
	}
	wg.Wait()
	r.SetValues(10, 90)
	g.SetValue(4)
	b.SetChecked(true)
	s.SetSelected(1)
	tt.Frame()
	if lo, hi := tt.Find(ByRole(ui.RoleSlider).Nth(0)).Now(), tt.Find(ByRole(ui.RoleSlider).Nth(1)).Now(); lo != 10 || hi != 90 {
		t.Fatalf("the range shows (%v, %v), want (10, 90)", lo, hi)
	}
	if n := tt.Find(ByRole(ui.RoleSlider).Nth(2)).Now(); n != 4 {
		t.Fatalf("the rating shows %v, want 4", n)
	}
	if !tt.Find(ByLabel("Bold")).Checked() || !tt.Find(ByLabel("Grid")).Checked() {
		t.Fatal("the check button or the segment set last does not show")
	}
}

// TestValueControlsAllocNothingAtRest: a frame of a built control, with
// nothing changing, allocates nothing of the widget layer's own: no more
// than the ui element it wraps, drawn alone. ui's own are not 0 (Rating's
// stars, 44 a frame; RangeSlider 7, Segmented 4, SplitButton 10), and are
// logged.
func TestValueControlsAllocNothingAtRest(t *testing.T) {
	lo, hi, stars, seg, on := 0.0, 100.0, 3, 0, true
	raw := func(fn func(c *ui.Context)) func(c *ui.Context) {
		return func(c *ui.Context) { ui.Column(c).Fill().Children(func() { fn(c) }) }
	}
	for _, k := range []struct {
		name string
		make func() Control
		ui   func(c *ui.Context)
	}{
		{"RangeTrackBar", func() Control { return RangeTrackBar(0, 100).SetStep(10).OnChange(func() {}) },
			func(c *ui.Context) { ui.RangeSlider(c, &lo, &hi, 0, 100, 10) }},
		{"Rating", func() Control { return Rating(5).SetValue(3).OnChange(func() {}) },
			func(c *ui.Context) { ui.Rating(c, &stars, 5) }},
		{"CheckButton", func() Control { return CheckButton("Bold").SetChecked(true).OnChange(func() {}) },
			func(c *ui.Context) { ui.Toggle(c, &on, "Bold") }},
		{"Segmented", func() Control { return Segmented("List", "Grid").SetFillWidth(true).OnChange(func() {}) },
			func(c *ui.Context) { ui.Segmented(c, &seg, "List", "Grid").FillEqually() }},
		{"SplitButton", func() Control { return SplitButton("Save", MenuItem{Text: "Save As…"}).OnClick(func() {}) },
			func(c *ui.Context) { ui.SplitButton(c, "Save").Arrow.Menu(func(m *ui.Menu) {}) }},
	} {
		t.Run(k.name, func(t *testing.T) {
			own := restAllocs(ui.NewTester(raw(k.ui), 600, 400))
			got := restAllocs(ui.NewTester(content(k.make()), 600, 400))
			t.Logf("%v allocations in a frame at rest, %v of them ui's", got, own)
			if got > own {
				t.Fatalf("%v allocations in a frame at rest, want ui's own %v: the widget adds %v", got, own, got-own)
			}
		})
	}
}

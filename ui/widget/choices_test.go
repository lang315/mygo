package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSpinTypedValue(t *testing.T) {
	s := Spin(0, 100, 1).SetValue(5)
	tt := tester(t, s)
	focus(tt, 1)
	tt.Key(0, ui.KeyEnd)
	tt.Type("0")
	tt.Frame()
	if s.Int() != 50 || s.Value() != 50 {
		t.Fatalf("Value %v Int %d, want 50", s.Value(), s.Int())
	}
}

func TestRangeSpinValues(t *testing.T) {
	r := RangeSpin(0, 10, 0, 100, 1, "to", "seconds").SetValues(2, 30)
	tt := tester(t, r)
	for _, s := range []string{"to", "seconds"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	if lo, hi := r.Ints(); lo != 2 || hi != 30 {
		t.Fatalf("Ints %d %d", lo, hi)
	}
}

func TestComboValueAndText(t *testing.T) {
	c := Combo(Option{"vn", "Vietnam"}, Option{"us", "United States"})
	c.SetValue("us")
	tt := tester(t, c)
	if !tt.HasText("United States") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if o, ok := c.Selected(); !ok || o.Value != "us" {
		t.Fatalf("Selected %v %v", o, ok)
	}
	c.Unselect()
	if _, ok := c.Selected(); ok || c.Value() != "" {
		t.Fatal("Unselect left a choice")
	}
	c.SetOptions(Option{"x", "X"})
	c.SetValue("x")
	tt.Frame()
	if !tt.HasText("X") {
		t.Fatalf("texts after SetOptions %q", tt.Texts())
	}
}

func TestTagSelect(t *testing.T) {
	g := TagSelect(Option{"a", "Alpha"}, Option{"b", "Beta"})
	tt := tester(t, g)
	click(t, tt, "Beta")
	if got := g.Values(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Values %v", got)
	}
}

func TestProgressAndSpinner(t *testing.T) {
	p := Progress().SetFraction(0.5).SetText("Half")
	s := Spinner()
	tt := tester(t, VBox(0).Add(p, s))
	if !tt.HasText("Half") {
		t.Fatalf("texts %q", tt.Texts())
	}
	s.Start()
	tt.Frame()
	s.Stop()
	tt.Frame()
}

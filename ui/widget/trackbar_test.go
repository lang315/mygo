package widget

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func trackTester(t *testing.T, tb *TrackBarWidget) (*Tester, *Node) {
	t.Helper()
	tt := NewTester(t, tb, 400, 300)
	return tt, tt.Find(ByRole(ui.RoleSlider))
}

func TestTrackBarShowsItsRangeAndValue(t *testing.T) {
	_, s := trackTester(t, TrackBar(10, 50).SetValue(30))
	if s.Min() != 10 || s.Max() != 50 || s.Now() != 30 {
		t.Fatalf("slider %v..%v at %v", s.Min(), s.Max(), s.Now())
	}
}

func TestTrackBarKeysMoveItAndReportOnce(t *testing.T) {
	changed := make(chan float64, 8)
	var tb *TrackBarWidget
	tb = TrackBar(0, 100).SetValue(50).OnChange(func() { changed <- tb.Value() })
	_, s := trackTester(t, tb)
	s.Key(0, ui.KeyRight)
	if got := received(t, changed); got <= 50 || tb.Value() != got {
		t.Fatalf("OnChange saw %v, Value %v after Right from 50", got, tb.Value())
	}
	s.Key(0, ui.KeyEnd)
	if got := received(t, changed); got != 100 {
		t.Fatalf("End gave %v", got)
	}
	s.Key(0, ui.KeyHome)
	if got := received(t, changed); got != 0 || tb.Int() != 0 {
		t.Fatalf("Home gave %v", got)
	}
}

func TestTrackBarSetValueClampsAndDoesNotReport(t *testing.T) {
	changed := make(chan struct{}, 2)
	tb := TrackBar(0, 10).OnChange(func() { changed <- struct{}{} })
	tt, s := trackTester(t, tb)
	tb.SetValue(99)
	tt.Settle()
	if tb.Value() != 10 || s.Now() != 10 {
		t.Fatalf("Value %v, node %v: 99 must clamp to 10", tb.Value(), s.Now())
	}
	tb.SetValue(-5)
	tt.Settle()
	if tb.Value() != 0 {
		t.Fatalf("Value %v, want 0", tb.Value())
	}
	select {
	case <-changed:
		t.Fatal("SetValue ran OnChange")
	default:
	}
}

func TestTrackBarStepSnapsAndArrowsMoveByIt(t *testing.T) {
	tb := TrackBar(0, 100).SetStep(25)
	_, s := trackTester(t, tb)
	s.Key(0, ui.KeyRight)
	if tb.Value() != 25 {
		t.Fatalf("Right with step 25 gave %v", tb.Value())
	}
	s.Key(0, ui.KeyRight)
	if tb.Value() != 50 {
		t.Fatalf("second Right gave %v", tb.Value())
	}
}

func TestTrackBarPressOnTheTrackSetsTheValue(t *testing.T) {
	tb := TrackBar(0, 100)
	tt, s := trackTester(t, tb)
	b := s.Bounds()
	tt.ClickAt(b.X+b.W*0.75, b.Y+b.H/2)
	tt.Settle()
	if got := tb.Value(); math.Abs(got-75) > 8 {
		t.Fatalf("a press at 75%% of the track gave %v", got)
	}
}

func TestTrackBarBindsAFloat(t *testing.T) {
	v := 20.0
	tb := Bind(TrackBar(0, 100), func() float64 { return v }, func(x float64) { v = x })
	_, s := trackTester(t, tb)
	if s.Now() != 20 {
		t.Fatalf("the bar shows %v, not the source's 20", s.Now())
	}
	s.Key(0, ui.KeyEnd)
	if v != 100 {
		t.Fatalf("the source got %v", v)
	}
}

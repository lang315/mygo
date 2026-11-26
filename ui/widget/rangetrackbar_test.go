package widget

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRangeTrackBarValues(t *testing.T) {
	r := RangeTrackBar(0, 100).SetValues(20, 80)
	if lo, hi := r.Values(); lo != 20 || hi != 80 {
		t.Fatalf("Values (%v, %v), want (20, 80)", lo, hi)
	}
	tt := NewTester(t, r, 400, 300)
	low, high := tt.Find(ByRole(ui.RoleSlider).Nth(0)), tt.Find(ByRole(ui.RoleSlider).Nth(1))
	if low.Now() != 20 || high.Now() != 80 || low.Min() != 0 || high.Max() != 100 {
		t.Fatalf("knobs at %v and %v of %v..%v", low.Now(), high.Now(), low.Min(), high.Max())
	}
}

func TestRangeTrackBarClampsAndSwaps(t *testing.T) {
	r := RangeTrackBar(0, 100).SetValues(90, -5)
	if lo, hi := r.Values(); lo != 0 || hi != 90 {
		t.Fatalf("Values (%v, %v) after SetValues(90, -5), want (0, 90)", lo, hi)
	}
	r.SetValues(math.NaN(), 50)
	if lo, hi := r.Values(); lo != 0 || hi != 90 {
		t.Fatalf("Values (%v, %v) after SetValues(NaN, 50), want (0, 90) kept", lo, hi)
	}
}

func TestRangeTrackBarStep(t *testing.T) {
	changed := make(chan float64, 4)
	var r *RangeTrackBarWidget
	r = RangeTrackBar(0, 100).SetStep(10).SetValues(20, 80).OnChange(func() { lo, _ := r.Values(); changed <- lo })
	tt := NewTester(t, r, 400, 300)
	tt.Find(ByRole(ui.RoleSlider).Nth(0)).Key(0, ui.KeyRight)
	if got := received(t, changed); got != 30 {
		t.Fatalf("Right on the low knob with step 10 gave %v from 20", got)
	}
	if _, hi := r.Values(); hi != 80 {
		t.Fatalf("the high value moved to %v", hi)
	}
}

func TestRangeTrackBarDragFiresOnChangeOnce(t *testing.T) {
	changed := make(chan struct{}, 8)
	r := RangeTrackBar(0, 100).SetValues(20, 80).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, r, 400, 300)
	k := tt.Find(ByRole(ui.RoleSlider).Nth(0)).Bounds()
	track := tt.Find(ByRole(ui.RoleGroup)).Bounds()
	x, y := middle(k)
	to := track.X + track.W/2
	tt.Press(x, y)
	tt.Settle()
	// A press on the knob's middle may move it by a rounding (20 to
	// 20.0000003), which is a value of its own: drop it.
	select {
	case <-changed:
	default:
	}
	tt.Move(to, y)
	tt.Frame()
	tt.Release(to, y)
	tt.Settle()
	received(t, changed)
	select {
	case <-changed:
		t.Fatal("OnChange ran twice for one move of a drag")
	default:
	}
	if lo, hi := r.Values(); lo <= 20 || lo >= 80 || hi != 80 {
		t.Fatalf("Values (%v, %v) after dragging the low knob to the middle", lo, hi)
	}
}

func TestRangeTrackBarSetterDoesNotFireOnChange(t *testing.T) {
	changed := make(chan struct{}, 4)
	r := RangeTrackBar(0, 100).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, r, 400, 300)
	r.SetValues(90, -5)
	tt.Settle()
	if tt.Find(ByRole(ui.RoleSlider).Nth(1)).Now() != 90 {
		t.Fatal("the high knob does not show 90")
	}
	select {
	case <-changed:
		t.Fatal("SetValues ran OnChange")
	default:
	}
}

func TestRangeTrackBarDisabled(t *testing.T) {
	changed := make(chan struct{}, 4)
	r := RangeTrackBar(0, 100).SetValues(20, 80).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, r.SetDisabled(true), 400, 300)
	track := tt.Find(ByRole(ui.RoleGroup)).Bounds()
	tt.ClickAt(track.X+track.W/2, track.Y+track.H/2)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyRight)
	tt.Settle()
	if lo, hi := r.Values(); lo != 20 || hi != 80 {
		t.Fatalf("a disabled RangeTrackBar moved to (%v, %v)", lo, hi)
	}
	select {
	case <-changed:
		t.Fatal("a disabled RangeTrackBar ran OnChange")
	default:
	}
}

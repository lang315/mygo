package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// clickStar clicks star i (from 1) of the rating r shows: the stars are
// squares as high as the rating, at its start (a stretched row leaves them
// there), with a gap of half a theme unit between them.
func clickStar(tt *Tester, i int) {
	b := tt.Find(ByRole(ui.RoleSlider)).Bounds()
	gap := ui.LightTheme().Space(0.5)
	tt.ClickAt(b.X+float32(i-1)*(b.H+gap)+b.H/2, b.Y+b.H/2)
	tt.Settle()
}

func TestRatingSetValueClamps(t *testing.T) {
	r := Rating(5)
	tt := NewTester(t, r, 400, 300)
	r.SetValue(9)
	tt.Settle()
	if r.Value() != 5 || tt.Find(ByRole(ui.RoleSlider)).Now() != 5 {
		t.Fatalf("Value %d after SetValue(9), want 5", r.Value())
	}
	r.SetValue(-1)
	tt.Settle()
	if r.Value() != 0 {
		t.Fatalf("Value %d after SetValue(-1), want 0", r.Value())
	}
}

func TestRatingClickSetsAndClears(t *testing.T) {
	changed := make(chan int, 4)
	var r *RatingWidget
	r = Rating(5).OnChange(func() { changed <- r.Value() })
	tt := NewTester(t, r, 400, 300)
	clickStar(tt, 3)
	if got := received(t, changed); got != 3 {
		t.Fatalf("a click on star 3 gave %d", got)
	}
	clickStar(tt, 3)
	if got := received(t, changed); got != 0 {
		t.Fatalf("a second click on star 3 gave %d, want 0", got)
	}
	select {
	case <-changed:
		t.Fatal("OnChange ran more than once a click")
	default:
	}
}

func TestRatingDisabled(t *testing.T) {
	changed := make(chan struct{}, 4)
	r := Rating(5).SetValue(2).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, r.SetDisabled(true), 400, 300)
	clickStar(tt, 4)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnd)
	tt.Settle()
	if r.Value() != 2 {
		t.Fatalf("a disabled Rating changed to %d", r.Value())
	}
	select {
	case <-changed:
		t.Fatal("a disabled Rating ran OnChange")
	default:
	}
}

func TestRatingSetterDoesNotFireOnChange(t *testing.T) {
	changed := make(chan struct{}, 4)
	r := Rating(5).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, r, 400, 300)
	r.SetValue(4)
	tt.Settle()
	if tt.Find(ByRole(ui.RoleSlider)).Now() != 4 {
		t.Fatal("the rating does not show 4")
	}
	select {
	case <-changed:
		t.Fatal("SetValue ran OnChange")
	default:
	}
}

func TestRatingBindsAnInt(t *testing.T) {
	v := 2
	r := Bind(Rating(5), func() int { return v }, func(x int) { v = x })
	tt := NewTester(t, r, 400, 300)
	if tt.Find(ByRole(ui.RoleSlider)).Now() != 2 {
		t.Fatal("the rating does not show the source's 2")
	}
	tt.Find(ByRole(ui.RoleSlider)).Key(0, ui.KeyEnd)
	if v != 5 {
		t.Fatalf("the source got %d", v)
	}
}

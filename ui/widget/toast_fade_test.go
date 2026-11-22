package widget

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// toastOpacity returns how opaque the toast c draws, over toastRed: its fill
// at opacity 1 is want, and toastRed has no green, so the green of the fill
// as drawn over want's is its opacity.
func toastOpacity(tt *Tester, c ui.TestNode, want ui.Color) float64 {
	return float64(toastFill(tt, c).G) / float64(want.G)
}

// Case 6: a form's toast fades in over 200 ms and out over its last 200 ms;
// the pointer resting on it holds it opaque.
func TestToastFades(t *testing.T) {
	f, tt := toastForm(t, 600, 400, VBox(0).Apply(NewStyle().Background(toastRed)))
	th := themeOf(false)
	fill, _, _, _ := statusColorsOf(th, StatusNone)
	want := fill.Over(th.Surface)
	if want.G < 150 {
		t.Fatalf("the neutral fill %v has too little green to tell its opacity", want)
	}
	f.ShowToast(Toast{Title: "Saved"})
	tt.Frame()
	for _, s := range []struct {
		by   time.Duration
		at   string
		want float64
	}{
		{100 * time.Millisecond, "100ms", 0.5},
		{1900 * time.Millisecond, "2s", 1},
		{1900 * time.Millisecond, "3.9s", 0.5},
	} {
		tt.Advance(s.by)
		tt.Frame() // at s.at: Advance builds the frames due before it
		if got := toastOpacity(tt, toastCard(t, tt), want); got < s.want-0.01 || got > s.want+0.01 {
			t.Errorf("at %s: the toast is %.3f opaque, want %v", s.at, got, s.want)
		}
	}
	c := toastCard(t, tt)
	tt.Move(c.Bounds.X+c.Bounds.W/2, c.Bounds.Y+c.Bounds.H/2)
	tt.Frame()
	if got := toastOpacity(tt, toastCard(t, tt), want); got < 0.99 {
		t.Errorf("at 3.9s with the pointer on it: the toast is %.3f opaque, want 1", got)
	}
}

// Review Focus 2: at text scale 1.5 in dark mode, the toast at rest is
// filled as before it faded: the status's fill over Surface.
func TestToastFadesToItsColors(t *testing.T) {
	for _, s := range []Status{StatusNone, StatusSuccess, StatusDanger} {
		f, tt := toastForm(t, 600, 400, VBox(0).Apply(NewStyle().Background(toastRed)))
		tt.SetDark(true)
		tt.SetPreferences(ui.Preferences{TextScale: 1.5})
		f.ShowToast(Toast{Status: s, Title: "Exported", Message: "12 accounts"})
		tt.Frame()
		tt.Advance(2 * time.Second)
		th := themeOf(true)
		fill, _, _, _ := statusColorsOf(th, s)
		if got, want := toastFill(tt, toastCard(t, tt)), fill.Over(th.Surface); !nearColor(got, want, 2) {
			t.Errorf("%v at 2s: the fill is %v, want %v", s, got, want)
		}
	}
}

// Review Focus 3: the allocations of a frame while a toast fades, logged
// and not asserted: a frame asking for animation frames is built anew, and
// one at opacity under 1 paints the toast faded, about 12 frames at each
// end of a toast. At rest, TestToastFrameAllocs asserts them.
func TestToastFadeFrameAllocs(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	plain := toastFrameAllocs(tt.Tester)
	f.ShowToast(Toast{Title: "Exported", Message: "12 accounts", Timeout: -1})
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	toasted := toastFrameAllocs(tt.Tester)
	add := false
	ut := ui.NewTester(func(c *ui.Context) {
		ui.Text(c, "Nội dung")
		if add {
			add = false
			c.AddToast(ui.Toast{Title: "Exported", Description: "12 accounts", Timeout: -1})
		}
	}, 600, 400)
	uiPlain := toastFrameAllocs(ut)
	add = true
	ut.Frame()
	ut.Advance(100 * time.Millisecond)
	uiToasted := toastFrameAllocs(ut)
	t.Logf("allocs a frame, fading: form %v, with a toast %v; ui %v, with a toast %v", plain, toasted, uiPlain, uiToasted)
	// The same form and toast at rest, its content asking for animation
	// frames itself: what a frame built anew costs without Fade.
	f2, tt2 := toastForm(t, 600, 400, Custom(func(c *ui.Context) ui.Element { c.AnimationFrame(); return ui.Text(c, "Nội dung") }))
	f2.ShowToast(Toast{Title: "Exported", Message: "12 accounts", Timeout: -1})
	tt2.Frame()
	tt2.Advance(time.Second)
	t.Logf("allocs a frame, at rest, content animating: %v", toastFrameAllocs(tt2.Tester))
}

// Review Focus 5: during the fade-in, the close button and Escape close the
// toast at once, OnClose once.
func TestToastClosesWhileFading(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var closed counter
	f.ShowToast(Toast{Title: "Saved", OnClose: closed.fn()})
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Find(ByRole(ui.RoleButton).Label("Close")).Click()
	if tt.Has(ByLabel("Saved")) {
		t.Error("the toast shows after its close button, fading in")
	}
	if n := closed.got(t); n != 1 {
		t.Errorf("the close button: OnClose ran %d times, want 1", n)
	}
	f.ShowToast(Toast{Title: "Saved again", Action: "Undo", OnClose: closed.fn()})
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Find(ByRole(ui.RoleButton).Label("Undo")).Focus().Key(0, ui.KeyEscape)
	tt.Frame()
	if tt.Has(ByLabel("Saved again")) {
		t.Error("the toast shows after Escape, fading in")
	}
	if n := closed.got(t); n != 2 {
		t.Errorf("Escape: OnClose ran %d times in all, want 2", n)
	}
}

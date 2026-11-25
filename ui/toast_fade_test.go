package ui

import (
	"math"
	"testing"
	"time"
)

// fadeRig is a window with toasts of a look of their own (ToastViewportBase
// and ToastBase), on a clock the test moves. fade is what Fade returned in
// the last frame, when the look calls it.
type fadeRig struct {
	tt   *Tester
	now  *time.Time
	add  []Toast // added in the next frame
	fade float32
}

// callFade is the look of a fadeRig that fades its toasts.
func callFade(_ *Context, p ToastParts) float32 { return p.Fade() }

// newFadeRig makes a fadeRig whose look calls look, unless nil, on each
// toast.
func newFadeRig(look func(c *Context, p ToastParts) float32) *fadeRig {
	r := &fadeRig{fade: -1}
	now := time.Unix(1_000_000, 0)
	r.now = &now
	r.tt = NewTester(func(c *Context) {
		for _, t := range r.add {
			c.AddToast(t)
		}
		r.add = r.add[:0]
		ToastViewportBase(c, func(_ Element, toasts []Toast) {
			for _, t := range toasts {
				p := ToastBase(c, t)
				if look != nil {
					r.fade = look(c, p)
				}
				p.Root.Children(func() { Text(c, t.Title) })
			}
		})
	}, 400, 300)
	r.tt.rt.clock = func() time.Time { return now }
	r.tt.Frame()
	return r
}

// at moves the clock to d after the start and builds a frame.
func (r *fadeRig) at(start time.Time, d time.Duration) {
	*r.now = start.Add(d)
	r.tt.Frame()
}

// show adds t and builds a frame, returning the time it showed at.
func (r *fadeRig) show(t Toast) time.Time {
	r.add = append(r.add, t)
	r.tt.Frame()
	return *r.now
}

func fadeNear(t *testing.T, what string, got, want float32) {
	t.Helper()
	if !withinEps(got, want, 0.01) {
		t.Errorf("%s: opacity %v, want %v", what, got, want)
	}
}

// Case 1: a toast of a custom look fades in over 200 ms and out over its
// last 200 ms.
func TestToastFadeCustomLook(t *testing.T) {
	r := newFadeRig(callFade)
	start := r.show(Toast{Title: "Saved"})
	fadeNear(t, "at 0", r.fade, 0)
	for _, s := range []struct {
		at   time.Duration
		want float32
	}{
		{100 * time.Millisecond, 0.5},
		{200 * time.Millisecond, 1},
		{2 * time.Second, 1},
		{3800 * time.Millisecond, 1}, // 200 ms left: not yet fading
		{3900 * time.Millisecond, 0.5},
		{3950 * time.Millisecond, 0.25},
	} {
		r.at(start, s.at)
		fadeNear(t, "at "+s.at.String(), r.fade, s.want)
	}
}

// A toast shorter than both fades: past its fade-in, it fades out.
func TestToastFadeShortTimeout(t *testing.T) {
	r := newFadeRig(callFade)
	start := r.show(Toast{Title: "Saved", Timeout: 300 * time.Millisecond})
	r.at(start, 100*time.Millisecond)
	fadeNear(t, "at 100ms", r.fade, 0.5)
	r.at(start, 250*time.Millisecond)
	fadeNear(t, "at 250ms, 50ms left", r.fade, 0.25)
}

// Case 2: a toast shown until closed stays opaque past its fade-in, and
// asks for no frame.
func TestToastFadeNoTimeout(t *testing.T) {
	r := newFadeRig(callFade)
	start := r.show(Toast{Title: "Uploading", Timeout: -1})
	r.at(start, 200*time.Millisecond)
	fadeNear(t, "at 200ms", r.fade, 1)
	r.at(start, time.Minute)
	fadeNear(t, "at 60s", r.fade, 1)
	if r.tt.rt.animating || !r.tt.h.afterAt.IsZero() {
		t.Errorf("at 60s: animating %v, wakes at %v", r.tt.rt.animating, r.tt.h.afterAt)
	}
}

// Case 3: the pointer resting on the toasts holds it opaque; gone, the fade
// goes on from where the time stopped.
func TestToastFadeStopped(t *testing.T) {
	r := newFadeRig(callFade)
	start := r.show(Toast{Title: "Saved"})
	r.at(start, 3900*time.Millisecond)
	fadeNear(t, "at 3.9s", r.fade, 0.5)
	b, ok := r.tt.Find("Saved")
	if !ok {
		t.Fatal("the toast does not show")
	}
	r.tt.Move(b.X+b.W/2, b.Y+b.H/2)
	fadeNear(t, "the pointer on it at 3.9s", r.fade, 1)
	if r.tt.rt.animating {
		t.Error("stopped, the toast asks for animation frames")
	}
	r.at(start, 3900*time.Millisecond+time.Minute)
	fadeNear(t, "the pointer on it a minute later", r.fade, 1)
	r.tt.Move(399, 1)
	fadeNear(t, "the pointer gone", r.fade, 0.5)
	r.at(start, 3950*time.Millisecond+time.Minute)
	fadeNear(t, "50ms after the pointer went", r.fade, 0.25)
}

// Case 4: the frames the fade asks for.
func TestToastFadeFrames(t *testing.T) {
	r := newFadeRig(callFade)
	start := r.show(Toast{Title: "Saved"})
	r.at(start, 100*time.Millisecond)
	if !r.tt.rt.animating {
		t.Error("fading in, no animation frame pending")
	}
	r.at(start, time.Second)
	if r.tt.rt.animating {
		t.Error("at rest, an animation frame pending")
	}
	if got := r.tt.h.afterAt.Sub(*r.now); got != 2800*time.Millisecond {
		t.Errorf("at rest at 1s, wakes in %v, want 2.8s (200ms before its end)", got)
	}
	r.at(start, 3900*time.Millisecond)
	if !r.tt.rt.animating {
		t.Error("fading out, no animation frame pending")
	}
}

// Case 4: a look that never calls Fade asks for no frame more than before.
func TestToastFadeNotCalled(t *testing.T) {
	r := newFadeRig(nil)
	start := r.show(Toast{Title: "Saved"})
	r.at(start, 100*time.Millisecond)
	if r.tt.rt.animating {
		t.Error("in the first 200ms, an animation frame pending")
	}
	r.at(start, time.Second)
	if r.tt.rt.animating {
		t.Error("at rest, an animation frame pending")
	}
	if got := r.tt.h.afterAt.Sub(*r.now); got != 3*time.Second {
		t.Errorf("at rest at 1s, wakes in %v, want 3s (its end)", got)
	}
}

// toastBox returns the theme's toast, the status in the overlay.
func toastBox(t *testing.T, tt *Tester) *node {
	t.Helper()
	var find func(e *node) *node
	find = func(e *node) *node {
		for ; e != nil; e = e.next {
			if e.role == RoleStatus {
				return e
			}
			if f := find(e.first); f != nil {
				return f
			}
		}
		return nil
	}
	if o := tt.rt.c.overlay; o != nil {
		if e := find(o.first); e != nil {
			return e
		}
	}
	t.Fatal("no toast")
	return nil
}

// Case 5: the theme's look fades as case 1, through Fade.
func TestToastFadeThemeLook(t *testing.T) {
	add := false
	now := time.Unix(1_000_000, 0)
	tt := NewTester(func(c *Context) {
		if add {
			c.AddToast(Toast{Title: "Saved"})
			add = false
		}
	}, 400, 300)
	tt.rt.clock = func() time.Time { return now }
	tt.Frame()
	add = true
	tt.Frame()
	start := now
	opacity := func() float32 {
		b := toastBox(t, tt)
		if !b.opacitySet {
			return 1
		}
		return b.opacity
	}
	fadeNear(t, "at 0", opacity(), 0)
	for _, s := range []struct {
		at   time.Duration
		want float32
	}{
		{100 * time.Millisecond, 0.5},
		{2 * time.Second, 1},
		{3900 * time.Millisecond, 0.5},
	} {
		now = start.Add(s.at)
		tt.Frame()
		fadeNear(t, "at "+s.at.String(), opacity(), s.want)
	}
}

// Case 7: a handle with no part fades to nothing, without a panic, as a
// checked handle answers with no part: its type's zero.
func TestToastFadeZeroParts(t *testing.T) {
	if got := (ToastParts{}).Fade(); got != 0 {
		t.Errorf("ToastParts{}.Fade() = %v, want 0", got)
	}
}

// Review Focus 1: a toast replaced by its ID while it shows starts its
// time anew (ui's rule), and so fades in anew, as the theme's look did.
func TestToastFadeReplaced(t *testing.T) {
	r := newFadeRig(callFade)
	start := r.show(Toast{ID: "save", Title: "Saving"})
	r.at(start, time.Second)
	fadeNear(t, "at 1s", r.fade, 1)
	again := r.show(Toast{ID: "save", Title: "Saved"})
	fadeNear(t, "replaced at 1s", r.fade, 0)
	r.at(again, 100*time.Millisecond)
	fadeNear(t, "100ms after the replacement", r.fade, 0.5)
}

// Review Focus 4: in the background, the fade runs on the toasts' clock.
func TestToastFadeInBackground(t *testing.T) {
	r := newFadeRig(callFade)
	r.tt.SetFocused(false)
	start := r.show(Toast{Title: "Downloaded"})
	r.at(start, 100*time.Millisecond)
	fadeNear(t, "in the background at 100ms", r.fade, 0.5)
	r.at(start, time.Second)
	fadeNear(t, "in the background at 1s", r.fade, 1)
}

// Review Focus 3: Fade allocates nothing: a frame in the fade allocates as
// one of a look that asks for animation frames itself, and a frame at rest
// as one of a look that never calls Fade.
func TestToastFadeAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector allocates")
	}
	animates := func(c *Context, _ ToastParts) float32 { c.AnimationFrame(); return 1 }
	frame := func(look func(*Context, ToastParts) float32, d time.Duration, limit float64) float64 {
		r := newFadeRig(look)
		start := r.show(Toast{Title: "Saved"})
		r.at(start, d)
		return allocsPerRun(t, 20, limit, func() { r.tt.Frame() })
	}
	// A stray malloc only raises without, which cannot fail the test: one
	// count. with is over its limit only if every count is.
	without := frame(animates, 100*time.Millisecond, math.Inf(1))
	if with := frame(callFade, 100*time.Millisecond, without); with > without {
		t.Errorf("fading: a frame calling Fade allocates %v, one animating itself %v", with, without)
	}
	without = frame(nil, time.Second, math.Inf(1))
	if with := frame(callFade, time.Second, without); with > without {
		t.Errorf("at rest: a frame calling Fade allocates %v, without %v", with, without)
	}
}

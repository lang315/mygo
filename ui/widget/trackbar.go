package widget

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// TrackBarWidget is a slider choosing a number between a minimum and a
// maximum: the user drags the knob, presses on the track, or uses the
// arrows, Page Up and Page Down, Home and End while it has the keyboard
// focus. It is ui.Slider underneath.
type TrackBarWidget struct {
	edit func() // editedOnce's
	common[TrackBarWidget]
	min, max float64
	step     float64
	value    mirror[float64]
	onChange func()
}

// TrackBar makes a track bar from min to max, at min. A max below min is
// raised to it, as WinForms does.
func TrackBar(lo, hi float64) *TrackBarWidget {
	t := &TrackBarWidget{min: lo, max: math.Max(lo, hi)}
	t.value.set(lo)
	t.init(t)
	return t
}

// Value returns the number the knob is at; Int rounds it.
func (t *TrackBarWidget) Value() float64 { mu.Lock(); defer mu.Unlock(); return t.value.value }
func (t *TrackBarWidget) Int() int       { return int(math.Round(t.Value())) }

// SetValue moves the knob, to the range's end if v is outside it, without
// running OnChange.
func (t *TrackBarWidget) SetValue(v float64) *TrackBarWidget {
	if math.IsNaN(v) {
		return t
	}
	t.change(func() { t.value.set(min(max(v, t.min), t.max)) })
	return t
}

// SetStep snaps the value to the minimum and multiples of step from it,
// with a tick mark at each, and has the arrows move by step. 0 (the
// default) is a continuous track.
func (t *TrackBarWidget) SetStep(step float64) *TrackBarWidget {
	t.change(func() { t.step = step })
	return t
}

// OnChange runs fn, on the event goroutine, after the user moved the knob;
// it runs for every value of a drag. A setter does not run it.
func (t *TrackBarWidget) OnChange(fn func()) *TrackBarWidget {
	t.change(func() { t.onChange = fn })
	return t
}

func (t *TrackBarWidget) build(c *ui.Context) ui.Element {
	p, changed := t.value.bind()
	if changed && t.onChange != nil {
		Post(t.onChange)
	}
	var e ui.Element
	if t.step > 0 {
		e = ui.StepSlider(c, p, t.min, t.max, t.step)
	} else {
		e = ui.Slider(c, p, t.min, t.max)
	}
	return e.OnChange(editedOnce(&t.edit, &t.value, &t.bound, &t.onChange))
}

func (t *TrackBarWidget) bindValue() float64  { return t.Value() }
func (t *TrackBarWidget) bindApply(v float64) { t.SetValue(v) }

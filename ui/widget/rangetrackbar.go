package widget

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// RangeTrackBarWidget is a slider of two knobs choosing a range between a
// minimum and a maximum, as a price range: the user drags a knob, presses
// on the track to move the nearest one, or uses the arrows, Home and End on
// the knob that has the keyboard focus. It is ui.RangeSlider underneath, as
// DevExpress's RangeTrackBarControl.
type RangeTrackBarWidget struct {
	edit func() // editedOnce's
	common[RangeTrackBarWidget]
	min, max float64
	step     float64
	value    mirror[[2]float64] // low, high
	onChange func()
}

// RangeTrackBar makes a range track bar from lo to hi, its knobs at both
// ends. A hi below lo is raised to it, as TrackBar's.
func RangeTrackBar(lo, hi float64) *RangeTrackBarWidget {
	t := &RangeTrackBarWidget{min: lo, max: math.Max(lo, hi)}
	t.value.set([2]float64{t.min, t.max})
	t.init(t)
	return t
}

// Values returns the low and the high end of the range.
func (t *RangeTrackBarWidget) Values() (low, high float64) {
	mu.Lock()
	defer mu.Unlock()
	return t.value.value[0], t.value.value[1]
}

// SetValues moves the knobs, without running OnChange: a value outside the
// track goes to its end, and low and high swap when low is above high. A NaN
// is ignored.
func (t *RangeTrackBarWidget) SetValues(low, high float64) *RangeTrackBarWidget {
	if math.IsNaN(low) || math.IsNaN(high) {
		return t
	}
	if low > high {
		low, high = high, low
	}
	t.change(func() { t.value.set([2]float64{min(max(low, t.min), t.max), min(max(high, t.min), t.max)}) })
	return t
}

// SetStep snaps the user's values to the minimum and multiples of step from
// it, with a tick mark at each, and has the arrows move by step. 0 (the
// default) is a continuous track.
func (t *RangeTrackBarWidget) SetStep(step float64) *RangeTrackBarWidget {
	t.change(func() { t.step = step })
	return t
}

// OnChange runs fn, on the event goroutine, after the user moved a knob; it
// runs for every value of a drag. A setter does not run it.
func (t *RangeTrackBarWidget) OnChange(fn func()) *RangeTrackBarWidget {
	t.change(func() { t.onChange = fn })
	return t
}

func (t *RangeTrackBarWidget) build(c *ui.Context) ui.Element {
	p, changed := t.value.bind()
	if changed && t.onChange != nil {
		Post(t.onChange)
	}
	return ui.RangeSlider(c, &p[0], &p[1], t.min, t.max, t.step).
		OnChange(editedOnce(&t.edit, &t.value, &t.bound, &t.onChange))
}

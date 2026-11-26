package widget

import "github.com/egoist/mygo/ui"

// RatingWidget is a row of stars, of which the first Value are filled: a
// click on a star sets the value, a click on the star set clears it, and
// the arrows, Home and End change it while it has the keyboard focus. It is
// ui.Rating underneath, as DevExpress's RatingControl.
type RatingWidget struct {
	edit func() // editedOnce's
	common[RatingWidget]
	max      int
	value    mirror[int]
	onChange func()
}

// Rating makes a rating of max stars, none filled. A max below 0 is 0.
func Rating(max int) *RatingWidget {
	if max < 0 {
		max = 0
	}
	r := &RatingWidget{max: max}
	r.init(r)
	return r
}

// Value returns the number of stars filled.
func (r *RatingWidget) Value() int { mu.Lock(); defer mu.Unlock(); return r.value.value }

// SetValue fills v stars, 0 to the stars there are, without running
// OnChange.
func (r *RatingWidget) SetValue(v int) *RatingWidget {
	r.change(func() { r.value.set(min(max(v, 0), r.max)) })
	return r
}

// OnChange runs fn, on the event goroutine, after the user changed the
// rating. A setter does not run it.
func (r *RatingWidget) OnChange(fn func()) *RatingWidget {
	r.change(func() { r.onChange = fn })
	return r
}

func (r *RatingWidget) build(c *ui.Context) ui.Element {
	p, changed := r.value.bind()
	if changed && r.onChange != nil {
		Post(r.onChange)
	}
	return ui.Rating(c, p, r.max).OnChange(editedOnce(&r.edit, &r.value, &r.bound, &r.onChange))
}

func (r *RatingWidget) bindValue() int  { return r.Value() }
func (r *RatingWidget) bindApply(v int) { r.SetValue(v) }

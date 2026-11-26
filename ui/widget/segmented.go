package widget

import "github.com/egoist/mygo/ui"

// SegmentedWidget is one choice among a row of joined buttons, as a switch
// between views: a click chooses a segment, as do the arrows while it has
// the keyboard focus. It is ui.Segmented underneath; DevExpress builds it
// from CheckButtons of one GroupIndex.
type SegmentedWidget struct {
	edit func() // editedOnce's
	common[SegmentedWidget]
	labels   []string
	fill     bool
	value    mirror[int]
	onChange func()
}

// Segmented makes a segmented control showing labels, the first chosen.
func Segmented(labels ...string) *SegmentedWidget {
	s := &SegmentedWidget{labels: labels}
	s.init(s)
	return s
}

// Selected returns the index of the segment chosen, -1 for none.
func (s *SegmentedWidget) Selected() int { mu.Lock(); defer mu.Unlock(); return s.value.value }

// SetSelected chooses segment i, or none for -1, without running OnChange.
// An index out of range is ignored.
func (s *SegmentedWidget) SetSelected(i int) *SegmentedWidget {
	s.change(func() {
		if i >= -1 && i < len(s.labels) {
			s.value.set(i)
		}
	})
	return s
}

// SetFillWidth makes the control as wide as the box that holds it, its
// segments sharing the width equally.
func (s *SegmentedWidget) SetFillWidth(on bool) *SegmentedWidget {
	s.change(func() { s.fill = on })
	return s
}

// OnChange runs fn, on the event goroutine, after the user chose a
// segment. A setter does not run it.
func (s *SegmentedWidget) OnChange(fn func()) *SegmentedWidget {
	s.change(func() { s.onChange = fn })
	return s
}

func (s *SegmentedWidget) build(c *ui.Context) ui.Element {
	p, changed := s.value.bind()
	if changed && s.onChange != nil {
		Post(s.onChange)
	}
	e := ui.Segmented(c, p, s.labels...)
	if s.fill {
		e.FillEqually()
	}
	return e.OnChange(editedOnce(&s.edit, &s.value, &s.bound, &s.onChange))
}

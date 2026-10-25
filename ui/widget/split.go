package widget

import "github.com/egoist/mygo/ui"

// SplitContainerWidget shows two panes side by side, or one above the
// other, with a divider between them that the user drags, or moves with
// the arrows once it has the keyboard focus. It is ui.Split underneath.
type SplitContainerWidget struct {
	common[SplitContainerWidget]
	first, second Control
	stacked       bool
	inPercent     bool
	pos           mirror[float32]
	onMoved       func()
}

// SplitContainer makes a container showing first and second side by side,
// each half of the room, with a divider the user moves. Stacked puts first
// above second instead.
func SplitContainer(first, second Control) *SplitContainerWidget {
	s := &SplitContainerWidget{first: first, second: second, inPercent: true}
	s.pos.set(50)
	s.init(s)
	adopt(&s.core, first)
	adopt(&s.core, second)
	return s
}

// Stacked puts the first pane above the second, the divider across.
func (s *SplitContainerWidget) Stacked() *SplitContainerWidget {
	s.change(func() { s.stacked = true })
	return s
}

// SetSplitterDistance makes the first pane d DIPs wide (high when stacked),
// the rest for the second. The divider keeps 40 DIPs from either edge; when
// the window leaves less room, the first pane gives way, and takes its size
// back when there is room again.
func (s *SplitContainerWidget) SetSplitterDistance(d float32) *SplitContainerWidget {
	s.change(func() { s.inPercent = false; s.pos.set(d) })
	return s
}

// SetSplitterPercent makes the first pane take p percent of the room, from
// 0 to 100: both panes then grow and shrink with the window, in proportion.
func (s *SplitContainerWidget) SetSplitterPercent(p float32) *SplitContainerWidget {
	s.change(func() { s.inPercent = true; s.pos.set(p) })
	return s
}

// SplitterDistance returns the first pane's size in DIPs, as the user left
// it; 0 while the container is set in percent.
func (s *SplitContainerWidget) SplitterDistance() float32 {
	mu.Lock()
	defer mu.Unlock()
	if s.inPercent {
		return 0
	}
	return s.pos.value
}

// SplitterPercent returns the share of the room the first pane takes, as
// the user left it; 0 while the container is set in DIPs.
func (s *SplitContainerWidget) SplitterPercent() float32 {
	mu.Lock()
	defer mu.Unlock()
	if !s.inPercent {
		return 0
	}
	return s.pos.value
}

// OnSplitterMoved runs fn, on the event goroutine, after the user moved
// the divider. A setter does not run it.
func (s *SplitContainerWidget) OnSplitterMoved(fn func()) *SplitContainerWidget {
	s.change(func() { s.onMoved = fn })
	return s
}

func (s *SplitContainerWidget) build(c *ui.Context) ui.Element {
	p, changed := s.pos.bind()
	pane := func(w Control) func() {
		return func() {
			outer := axis
			axis = vertical
			defer func() { axis = outer }()
			if w != nil {
				render(c, w).Grow(1)
			}
		}
	}
	first, second := pane(s.first), pane(s.second)
	var e ui.Element
	switch {
	case s.inPercent && s.stacked:
		e = ui.SplitVerticalPercent(c, p, first, second)
	case s.inPercent:
		e = ui.SplitPercent(c, p, first, second)
	case s.stacked:
		e = ui.SplitVertical(c, p, first, second)
	default:
		e = ui.Split(c, p, first, second)
	}
	if e.Changed() {
		s.pos.value, s.pos.dirty = s.pos.shown, false
		changed = true
	}
	if changed && s.onMoved != nil {
		Post(s.onMoved)
	}
	return e.Grow(1)
}

func (s *SplitContainerWidget) fillsAcross() bool { return true }

func (s *SplitContainerWidget) children() []Control { return []Control{s.first, s.second} }

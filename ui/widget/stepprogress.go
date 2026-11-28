package widget

import (
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// The step bar's measures, in DIPs: the circle's diameter, the room between
// it and the title, and the thickness of the lines joining the circles.
const (
	stepDot      = 24
	stepTitleGap = 4
	stepLine     = 2
)

// StepProgressBarWidget shows the steps of a process as numbered circles
// joined by lines, each with its title, as DevExpress's StepProgressBar:
// the steps before the current one are done, with a check, and the current
// one is in the accent. The steps run from the start edge, or from the top
// after SetVertical(true).
type StepProgressBarWidget struct {
	common[StepProgressBarWidget]
	steps    []string
	nums     []string // the circles' numbers, made as the steps change, so a frame formats none
	edges    []func(p *ui.Painter, r ui.Rect)
	step     int
	vertical bool

	// What build last saw, which the drawing callbacks read without mu:
	// only the drawing thread touches them.
	drawn struct {
		n, step       int
		vertical, rtl bool
	}
	pad         float32                        // a vertical step's room under its title
	done, ahead ui.Color                       // the lines' colors
	ink         ui.Color                       // the check's
	check       func(p *ui.Painter, r ui.Rect) // drawCheck, made once
	pts         [3][2]float32                  // the check's points, kept so drawing it allocates nothing
}

// StepProgressBar makes a bar of steps, titled steps, at the first.
func StepProgressBar(steps ...string) *StepProgressBarWidget {
	s := &StepProgressBarWidget{}
	s.init(s)
	s.setSteps(steps)
	return s
}

// Step returns the index of the current step.
func (s *StepProgressBarWidget) Step() int { mu.Lock(); defer mu.Unlock(); return s.step }

// SetStep makes step i the current one, the steps before it done; an index
// out of range is the nearest step.
func (s *StepProgressBarWidget) SetStep(i int) *StepProgressBarWidget {
	s.change(func() { s.setStep(i) })
	return s
}

// SetSteps replaces the steps. The current step stays, or is the last where
// there are fewer steps now.
func (s *StepProgressBarWidget) SetSteps(steps ...string) *StepProgressBarWidget {
	steps = slices.Clone(steps)
	s.change(func() { s.setSteps(steps) })
	return s
}

// SetVertical stacks the steps from the top, each circle before its title.
func (s *StepProgressBarWidget) SetVertical(on bool) *StepProgressBarWidget {
	s.change(func() { s.vertical = on })
	return s
}

// setStep is SetStep's, under mu.
func (s *StepProgressBarWidget) setStep(i int) { s.step = max(0, min(i, len(s.steps)-1)) }

// setSteps sets the steps, which it owns, with their numbers and the
// callbacks drawing their lines. Under mu.
func (s *StepProgressBarWidget) setSteps(steps []string) {
	for i := len(s.nums); i < len(steps); i++ {
		s.nums = append(s.nums, strconv.Itoa(i+1))
		s.edges = append(s.edges, func(p *ui.Painter, r ui.Rect) { s.drawEdges(p, r, i) })
	}
	s.steps = steps
	s.setStep(s.step)
}

func (s *StepProgressBarWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	if s.check == nil {
		s.check = s.drawCheck
	}
	s.drawn.n, s.drawn.step, s.drawn.vertical, s.drawn.rtl = len(s.steps), s.step, s.vertical, c.Direction() == ui.RightToLeft
	s.pad, s.done, s.ahead, s.ink = t.Space(4), t.Accent, t.Border, t.AccentText
	var bar ui.Element
	if s.vertical {
		bar = ui.Column(c)
	} else {
		bar = ui.Row(c).AlignItems(ui.Start)
	}
	return bar.Children(func() {
		for i, title := range s.steps {
			s.buildStep(c, i, title)
		}
	})
}

// buildStep builds step i: its circle and its title, over the halves of the
// lines to the steps beside it. Under mu, while building.
func (s *StepProgressBarWidget) buildStep(c *ui.Context, i int, title string) {
	t := c.Theme()
	var cell ui.Element
	if s.vertical {
		// The room under the title carries the line down to the next circle.
		cell = ui.Row(c).AlignItems(ui.Center).Gap(stepTitleGap).Padding(0, 0, s.pad, 0)
	} else {
		cell = ui.Column(c).AlignItems(ui.Center).Gap(stepTitleGap).Grow(1).Basis(0).PaddingX(t.Space(1))
	}
	cell.Draw(s.edges[i])
	if i == s.step {
		cell.Current(ui.CurrentStep)
	}
	cell.Children(func() {
		dot := ui.Row(c).Width(stepDot).Height(stepDot).Radius(stepDot / 2).Shrink(0).
			AlignItems(ui.Center).Justify(ui.Center)
		switch {
		case i < s.step:
			dot.Background(t.Accent).Draw(s.check)
		case i == s.step:
			dot.Background(t.Accent).TextColor(t.AccentText)
		default:
			dot.Background(t.Background).BorderWidth(stepLine).BorderColor(t.Border).TextColor(t.TextMuted)
		}
		dot.Children(func() {
			if i >= s.step {
				ui.Text(c, s.nums[i]).SingleLine().FontWeight(600)
			}
		})
		text := ui.Text(c, title)
		switch {
		case i == s.step:
			text.TextColor(t.Text).FontWeight(600)
		case i > s.step:
			text.TextColor(t.TextMuted)
		}
		if !s.vertical {
			text.TextAlign(ui.Center)
		}
	})
}

// drawEdges draws, behind the circle of step i in its cell r, the halves of
// the lines to the steps before and after it, in the accent where the step
// they lead to is done or current.
func (s *StepProgressBarWidget) drawEdges(p *ui.Painter, r ui.Rect, i int) {
	d := s.drawn
	n := d.n // the steps the frame built: a cell i is one of them
	before, after := s.ahead, s.ahead
	if i <= d.step {
		before = s.done
	}
	if i+1 <= d.step {
		after = s.done
	}
	if d.vertical {
		x := r.X + stepDot/2
		if d.rtl {
			x = r.X + r.W - stepDot/2
		}
		mid := r.Y + (r.H-s.pad)/2 // the circle is centered on the title, above the room under it
		if i > 0 {
			p.Line(x, r.Y, x, mid, stepLine, before)
		}
		if i < n-1 {
			p.Line(x, mid, x, r.Y+r.H, stepLine, after)
		}
		return
	}
	y, mid := r.Y+stepDot/2, r.X+r.W/2
	start, end := r.X, r.X+r.W
	if d.rtl {
		start, end = end, start
	}
	if i > 0 {
		p.Line(start, y, mid, y, stepLine, before)
	}
	if i < n-1 {
		p.Line(mid, y, end, y, stepLine, after)
	}
}

// drawCheck draws a check in the circle r of a done step.
func (s *StepProgressBarWidget) drawCheck(p *ui.Painter, r ui.Rect) {
	u := r.W / 24
	s.pts = [3][2]float32{
		{r.X + 7*u, r.Y + 12.5*u},
		{r.X + 10.5*u, r.Y + 16*u},
		{r.X + 17*u, r.Y + 8.5*u},
	}
	p.StrokePolyline(s.pts[:], 2*u, s.ink)
}

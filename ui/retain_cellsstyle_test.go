package ui

import "testing"

// CellsStyle is read as the rows build: its colors are what a Table's rows are
// built with, so a change of one rebuilds the rows a list keeps, and they draw
// as the twin that builds every row draws them.
func TestRetainCellsStyleChangeRebuildsKeptRows(t *testing.T) {
	var states []*ListState
	p := newPair(t, 400, 300, func() func(c *Context) {
		s := &ListState{CellSelect: true, Selection: &Selection[int]{}, Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		states = append(states, s)
		cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
		return func(c *Context) {
			Table(c, s, cols, 8, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
		}
	})
	p.idle(2)
	p.both(func(tt *Tester) { tt.Click("r1c1") })
	p.idle(3)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test shows nothing")
	}
	th := LightTheme()
	base := CellsStyle{Cell: th.Success, CellText: th.AccentText, Row: th.Warning.Alpha(0.2), Stripe: th.Danger.Alpha(0.1)}
	for _, step := range []struct {
		what string
		set  func(s *CellsStyle)
	}{
		{"all four colors", func(s *CellsStyle) { *s = base }},
		{"Cell", func(s *CellsStyle) { s.Cell = th.Accent }},
		{"CellText", func(s *CellsStyle) { s.CellText = th.Text }},
		{"Row", func(s *CellsStyle) { s.Row = th.Success.Alpha(0.3) }},
		{"Stripe", func(s *CellsStyle) { s.Stripe = th.Accent.Alpha(0.1) }},
	} {
		for _, s := range states {
			step.set(&s.CellsStyle)
		}
		p.frame(step.what)
		p.idle(2)
	}
}

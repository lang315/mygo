package ui

import "testing"

// A call to ScrollTo made before the width changed the columns scrolls to the
// item in the rows the next build lays out, with or without headers.
func TestGridViewScrollAfterTheColumnsChanged(t *testing.T) {
	for _, headed := range []bool{false, true} {
		name := "no headers"
		if headed {
			name = "headers"
		}
		t.Run(name, func(t *testing.T) {
			s := &GridState{}
			if headed {
				s.Header = func(i int) bool { return i%10 == 0 }
			}
			tt := coreNewTester(func(c *context) {
				coreColumn(c).Fill().Children(func() {
					coreGridView(c, s, 400, 100, 40, func(i int) {
						if s.Header != nil && s.Header(i) {
							coreTextf(c, "H%d", i)
						} else {
							coreTextf(c, "Item %d", i)
						}
					}).Grow(1)
				})
			}, 448, 300)
			tt.Frame()
			s.ScrollTo(301, Start)
			tt.SetSize(230, 300)
			tt.Frame()
			tt.Frame()
			tt.Frame()
			if s.cols != 2 {
				t.Fatalf("the narrow grid has %d columns, want 2", s.cols)
			}
			first, _ := s.rows.Visible()
			at, _, _ := s.rowItems(first, 400)
			if at < 290 || at > 301 {
				t.Fatalf("ScrollTo(301, Start) before the grid got narrower shows from item %d, want 301 at the top", at)
			}
		})
	}
}

// The user scrolling after the call is not undone by it: the request made in
// the build after the grid was applied in that frame, and the build that
// follows does not apply it again.
func TestGridViewScrollIsNotAppliedAgainOverTheUsersScrolling(t *testing.T) {
	s := &GridState{}
	flag := false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, 1000, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
			if flag {
				flag = false
				s.ScrollTo(901, Start)
			}
		})
	}, 448, 300)
	tt.Frame()
	tt.Update(func() { flag = true })
	tt.Frame()
	before, _ := s.rows.Visible()
	tt.Move(200, 150)
	tt.Scroll(200, 150, 0, 120)
	after, _ := s.rows.Visible()
	if after <= before {
		t.Fatalf("the user scrolled from row %d to %d: the request was applied again over it", before, after)
	}
}

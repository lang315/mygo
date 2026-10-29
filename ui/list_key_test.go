package ui

import "testing"

// Key is asked once for a row in a pass: the build, the choice around the row,
// the checks of a Table and the places the list remembers share the answer
// (an integer of 256 or more, a string or a struct returned as an any is
// allocated each time Key is asked). Counted for each row, by the calls of the
// app's Key, in frames that idle, hover, scroll by a pixel and by a page.
func TestListAsksKeyOncePerRowPerPass(t *testing.T) {
	for _, mode := range []string{"plain", "selection", "selected and selection", "table with checks"} {
		t.Run(mode, func(t *testing.T) {
			type at struct {
				frame uint64
				pass  int
				row   int
			}
			calls := map[at]int{}
			var tt *Tester
			s := &ListState{RowVersion: func(int) uint64 { return 0 }}
			var sel, checked Selection[int]
			var cur int
			switch mode {
			case "selection":
				s.Selection = &sel
			case "selected and selection":
				s.Selection, s.Selected = &sel, &cur
			case "table with checks":
				s.Selection, s.Selected, s.Checked = &sel, &cur, &checked
			}
			s.Key = func(i int) any {
				if tt != nil { // not in the Tester's first frame
					calls[at{tt.rt.frame, tt.rt.pass, i}]++
				}
				return i + 1000
			}
			tt = NewTester(func(c *Context) {
				Column(c).Fill().Children(func() {
					if mode == "table with checks" {
						Table(c, s, []TableColumn{{Title: "Name"}}, 5000, func(r, col int) { Textf(c, "row %d", r) }).Grow(1)
						return
					}
					List(c, s, 5000, func(i int) { Row(c).Children(func() { Textf(c, "row %d", i) }) }).Grow(1)
				})
			}, 300, 400)
			for range 8 {
				tt.Frame()
			}
			steps := []struct {
				what string
				do   func()
			}{
				{"idle", func() { tt.Frame() }},
				{"hover", func() { tt.Move(150, 200) }},
				{"hover another row", func() { tt.Move(150, 240) }},
				{"scroll 1 px", func() { tt.Scroll(150, 200, 0, 1) }},
				{"scroll half a pixel", func() { tt.Scroll(150, 200, 0, 0.5) }},
				{"scroll a page", func() { tt.Scroll(150, 200, 0, 300) }},
				{"idle after the scroll", func() { tt.Frame() }},
			}
			for _, st := range steps {
				clear(calls)
				st.do()
				worst, asked := 0, 0
				for _, n := range calls {
					asked++
					worst = max(worst, n)
				}
				if asked == 0 {
					t.Fatalf("%s: Key was not asked for any row", st.what)
				}
				// In each frame and pass: once for a row, and once more where the
				// callback of the row above ran in between (the choice around a row
				// reads its neighbours before it, and a callback may change the
				// data, so the identity of a row is asked again after it).
				const limit = 2
				if worst > limit {
					t.Errorf("%s: a row was asked for %d times in a pass, at most %d", st.what, worst, limit)
				}
			}
		})
	}
}

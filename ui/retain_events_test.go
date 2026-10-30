package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// The events of a Table that sits outside a list (SelectionChanged, SortChanged,
// ColumnsChanged, CheckedChanged, Activated) reach the rows that ask of it: a
// pass clears an event, so the pictures cannot tell, but the rows that saw it
// can be counted, in the twin that keeps rows and in the one that builds all.
func TestRetainRowsSeeTheEventsOfATable(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End}}
	for _, tc := range []struct {
		name string
		ask  func(h Element) bool
		do   func(t *testing.T, tt *Tester)
	}{
		{"SelectionChanged", func(h Element) bool { return h.SelectionChanged() }, func(t *testing.T, tt *Tester) { _ = tt.Click("b.txt") }},
		{"SortChanged", func(h Element) bool { return h.SortChanged() }, func(t *testing.T, tt *Tester) { _ = tt.Click("Name") }},
		{"Activated", func(h Element) bool { _, ok := h.Activated(); return ok }, func(t *testing.T, tt *Tester) {
			_ = tt.Click("c.txt")
			tt.Key(0, KeyEnter)
		}},
		{"CheckedChanged", func(h Element) bool { return h.CheckedChanged() }, func(t *testing.T, tt *Tester) {
			_ = tt.Click("b.txt")
			tt.Key(0, KeySpace)
		}},
		{"ColumnsChanged", func(h Element) bool { return h.ColumnsChanged() }, func(t *testing.T, tt *Tester) {
			kind := header(t, tt, "Kind")
			edge, y := float32(kind.X+kind.W)-2, float32(kind.Y+kind.H/2)
			tt.Press(edge, y)
			tt.Move(edge+30, y)
			tt.Release(edge+30, y)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 500, 400, func() func(c *Context) {
				sort := SortOrder{Column: "Size"}
				sel := -1
				var checked Selection[string]
				ts := &ListState{Sort: &sort, Selected: &sel, Checked: &checked, Key: func(i int) any { return names[i] }}
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						h := Table(c, ts, cols, len(names), func(row, col int) {
							if col == 0 {
								Text(c, names[row]).SingleLine()
							}
						}).Height(120)
						List(c, s, 30, func(i int) {
							r := Row(c).Padding(4)
							if tc.ask(h) {
								r.Background(Hex("#ff0000"))
								countSeen(c)
							}
							r.Children(func() { Textf(c, "row %d", i) })
						}).Grow(1)
					})
				}
			})
			p.idle(3)
			p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
			p.idle(2)
			seenKept.Store(0)
			seenBuilt.Store(0)
			p.both(func(tt *Tester) { tc.do(t, tt) })
			p.compare("do")
			p.frame("after")
			if seenKept.Load() != seenBuilt.Load() {
				t.Errorf("rows that saw the event: %d with kept rows, %d with every row built", seenKept.Load(), seenBuilt.Load())
			}
			if seenBuilt.Load() == 0 {
				t.Errorf("no row saw the event: the test does not test")
			}
		})
	}
}

// The Dismissed of a Modal that sits outside the list: Escape sets it as the
// modal builds, before the rows, and the rows that ask must see it.
func TestRetainRowsSeeTheDismissedOfAnOutsideModal(t *testing.T) {
	p := newPair(t, 400, 400, func() func(c *Context) {
		open := true
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				m := Modal(c, &open, func() { Textf(c, "dialog") })
				List(c, s, 30, func(i int) {
					if m.Dismissed() {
						countSeen(c)
					}
					Textf(c, "row %d", i)
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	seenKept.Store(0)
	seenBuilt.Store(0)
	p.both(func(tt *Tester) { tt.Key(0, KeyEscape) })
	p.compare("escape")
	p.frame("after")
	if seenKept.Load() != seenBuilt.Load() || seenBuilt.Load() == 0 {
		t.Errorf("rows that saw Dismissed: %d with kept rows, %d with every row built", seenKept.Load(), seenBuilt.Load())
	}
}

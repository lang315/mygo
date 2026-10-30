package ui

import "testing"

// Rows that show whether their list has the focus (ListRow.ListFocused): kept
// while idle, and following the focus as it goes in, out and from one list to
// another, in the frame it moves (a click gives the focus in the first pass,
// and the rows are linked or built in the passes after).
func TestRetainRowsShowingTheFocusOfTheirList(t *testing.T) {
	p := newPair(t, 600, 400, func() func(c *Context) {
		mk := func() *ListState {
			sel := 0
			return &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Selected: &sel}
		}
		a, b := mk(), mk()
		return func(c *Context) {
			Row(c).Fill().Children(func() {
				for k, s := range []*ListState{a, b} {
					Column(c).Key(k).Grow(1).Children(func() {
						List(c, s, 30).Rows(func(r ListRow) {
							Textf(c, "list %d row %d focused=%v", k, r.Index, r.ListFocused())
						}).Grow(1)
					})
				}
			})
		}
	})
	p.idle(4)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
		t.Errorf("idle: %d rows built, %d kept; want none built", b, k)
	}
	p.both(func(tt *Tester) { tt.ClickAt(100, 60) }) // list 0 takes the focus
	p.compare("click on list 0")
	p.frame("list 0 focused")
	p.frame("list 0 focused 2")
	built, kept = p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle with the focus")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
		t.Errorf("idle with the focus: %d rows built, %d kept; want none built", b, k)
	}
	p.both(func(tt *Tester) { tt.ClickAt(400, 60) }) // the focus goes to list 1
	p.compare("click on list 1")
	p.frame("list 1 focused")
	p.both(func(tt *Tester) { tt.SetFocused(false) }) // the window loses the focus
	p.compare("window blurred")
	p.frame("window blurred 2")
	p.both(func(tt *Tester) { tt.SetFocused(true) })
	p.compare("window focused")
	p.both(func(tt *Tester) { tt.Key(0, KeyTab) }) // out of the lists
	p.compare("tab")
	p.frame("tab 2")
	p.both(func(tt *Tester) { tt.Key(Shift, KeyTab) })
	p.compare("shift tab")
	p.frame("shift tab 2")
}

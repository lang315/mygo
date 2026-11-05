package ui

import "testing"

// ObserveInput is counted at the commit (engine.observers) that a kept row
// skips: a row observing its input is built, as the twin that builds every
// row shows.
func TestRetainRowsObservingInputAreBuilt(t *testing.T) {
	var got [2]int
	p := newPair(t, 400, 200, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			n := 0
			if c.rt.noRetain {
				n = 1
			}
			List(c, s, 5, func(i int) {
				Box(c).Size(200, 30).ObserveInput(func(InputEvent) { got[n]++ })
			}).Grow(1)
		}
	})
	p.idle(3)
	p.both(func(tt *Tester) { tt.ClickAt(50, 15) })
	p.frame("click")
	if got[0] == 0 || got[0] != got[1] {
		t.Errorf("the kept list's rows observed %d events, the twin's %d", got[0], got[1])
	}
}

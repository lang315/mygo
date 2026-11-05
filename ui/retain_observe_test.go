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

// FocusTakesText reads the focus of the window, which no state of the row
// versions: a row that asks is built in every frame, and shows what the
// twin that builds every row does.
func TestRetainRowsAskingFocusTakesTextAreBuilt(t *testing.T) {
	p := newPair(t, 400, 400, func() func(c *Context) {
		var text string
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				TextInput(c, &text)
				List(c, s, 30, func(i int) {
					Textf(c, "row %d %v", i, c.FocusTakesText())
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	built := p.kept.rt.rowsBuilt
	p.frame("idle")
	if p.kept.rt.rowsBuilt == built {
		t.Error("rows that ask FocusTakesText were kept")
	}
	p.both(func(tt *Tester) { tt.ClickAt(50, 15) })
	p.frame("focus")
	p.frame("focus 2")
}

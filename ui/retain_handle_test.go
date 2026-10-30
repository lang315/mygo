package ui

import (
	"fmt"
	"testing"
)

// The queries of a Handle name an element of the window that may not be built in
// this pass: a row that asks (here whether a field outside the list has the
// focus, and where it is) is built, and shows what a build shows.
func TestRetainRowsAskingAHandleAreBuilt(t *testing.T) {
	for _, tc := range []struct {
		name string
		ask  func(c *Context, h *Handle) string
	}{
		{"Focused", func(c *Context, h *Handle) string { return fmt.Sprint(h.Focused(c)) }},
		{"FocusWithin", func(c *Context, h *Handle) string { return fmt.Sprint(h.FocusWithin(c)) }},
		{"Bounds", func(c *Context, h *Handle) string { return fmt.Sprint(h.Bounds(c).H) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 400, 400, func() func(c *Context) {
				var text string
				var field Handle
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						TextInput(c, &text).Bind(&field)
						List(c, s, 30, func(i int) {
							Textf(c, "row %d %s", i, tc.ask(c, &field))
						}).Grow(1)
					})
				}
			})
			p.idle(3)
			built := p.kept.rt.rowsBuilt
			p.frame("idle")
			if p.kept.rt.rowsBuilt == built {
				t.Errorf("rows that ask a Handle were kept")
			}
			p.both(func(tt *Tester) { tt.ClickAt(50, 15) })
			p.frame("focus")
			p.frame("focus 2")
			p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
			p.frame("tab")
			p.frame("tab 2")
		})
	}
}

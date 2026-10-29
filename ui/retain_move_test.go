package ui

import (
	"fmt"
	"testing"
)

// A Table of rows with a RowVersion that scrolls, by whole and fractional
// pixels, steps that are not exact in float32, pages and back, at scales 1 and
// 2: a kept Tester and one that builds every row draw alike after every step
// (P4: a row that moved keeps its layout, and replays its paint shifted).
func TestRetainMoveScenarios(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		t.Run(fmt.Sprintf("scale %v", scale), func(t *testing.T) {
			type app struct {
				rows []string
				vers []uint64
				list ListState
			}
			p := newPair(t, 400, 600, func() func(c *Context) {
				a := &app{rows: make([]string, 3000), vers: make([]uint64, 3000)}
				for i := range a.rows {
					a.rows[i] = fmt.Sprintf("acc%06d", i)
				}
				cols := []TableColumn{{Title: "Name"}, {Title: "Status", Width: 80}}
				a.list.Key = func(i int) any { return i }
				a.list.RowVersion = func(i int) uint64 { return a.vers[i] }
				return func(c *Context) {
					Table(c, &a.list, cols, len(a.rows), func(r, col int) {
						if col == 0 {
							Text(c, a.rows[r])
						} else {
							Textf(c, "%d", r%7)
						}
					}).Grow(1)
				}
			})
			p.both(func(tt *Tester) { tt.SetScale(scale) })
			for range 10 {
				p.both(func(tt *Tester) { tt.Frame() })
			}
			p.compare("warm")
			p.idle(4)
			p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, 2000) })
			p.compare("scroll far")
			p.idle(4)
			step := func(what string, dy float32) {
				t.Helper()
				p.both(func(tt *Tester) { tt.Scroll(150, 300, 0, dy) })
				p.compare(what)
			}
			for i := range 10 {
				step(fmt.Sprintf("scroll %d", i), float32(1-2*(i%2)))
			}
			if n := p.kept.rt.rowsShifted; n == 0 {
				t.Errorf("a scroll by a pixel shifted the layout of %d rows, want > 0", n)
			}
			if n := p.kept.rt.rowsShiftReplayed; n == 0 {
				t.Errorf("a scroll by a pixel replayed the paint of %d rows shifted, want > 0", n)
			}
			for i, dy := range []float32{0.5, 0.5, -0.5, -0.5, 0.25, 0.25, -0.25, -0.25} {
				step(fmt.Sprintf("scroll half %d", i), dy)
			}
			for i, dy := range []float32{0.37, 1.13, -0.37, -1.13} {
				step(fmt.Sprintf("scroll odd %d", i), dy)
			}
			step("scroll page", 500)
			step("scroll page back", -500)
			p.idle(3)
			step("scroll again", 3)
			p.idle(3)
		})
	}
}

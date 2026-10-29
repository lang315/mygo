package ui

import (
	"fmt"
	"testing"
	"time"
)

// Two nested Match transitions: the inner (registered first, as its Transition is
// set inside the outer's Children) keeps its size and moves (showAt: shift), the
// outer changes size at the same place (showAt: layout and place again).
func TestRetainLayoutNestedMatchShiftThenPlace(t *testing.T) {
	type mo struct{}
	type mi struct{}
	for _, twin := range []bool{false, true} {
		t.Run(fmt.Sprint("twin ", twin), func(t *testing.T) {
			show := false
			p := newPair(t, 300, 500, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					to := ElementTransition{Duration: 300 * time.Millisecond, Ease: Linear, Match: mo{}}
					ti := ElementTransition{Duration: 300 * time.Millisecond, Ease: Linear, Match: mi{}}
					Column(c).Fill().Children(func() {
						if !show {
							Column(c).Key("oa").Height(200).Children(func() {
								Column(c).Key("ia").Height(250).Transition(ti).Children(func() { Text(c, "a") })
							}).Transition(to)
							return
						}
						Column(c).Key("ob").Height(400).Children(func() {
							Text(c, "spacer").Height(60)
							Column(c).Key("ib").Height(250).Transition(ti).Children(func() {
								List(c, s, 100, func(i int) { Row(c).Children(func() { Textf(c, "row %d", i) }) }).Grow(1)
							})
						}).Transition(to)
					})
				}
			})
			if twin {
				p.kept.rt.layoutCheck = true
			}
			p.idle(2)
			show = true
			p.both(func(tt *Tester) { tt.Frame() })
			if twin {
				t.Logf("show: layoutMismatch %d", p.kept.rt.layoutMismatch)
			}
			p.compare("show")
			kept := 0
			for i := range 6 {
				p.both(func(tt *Tester) { tt.Advance(16 * time.Millisecond) })
				kept += p.kept.rt.rowsLaidKept
				if twin {
					t.Logf("advance %d: layoutMismatch %d", i, p.kept.rt.layoutMismatch)
				}
				p.compare(fmt.Sprintf("advance %d", i))
				if twin && p.kept.rt.layoutMismatch != 0 {
					t.Fatalf("advance %d: layoutMismatch %d", i, p.kept.rt.layoutMismatch)
				}
			}
			t.Logf("kept or shifted %d", kept)
		})
	}
}

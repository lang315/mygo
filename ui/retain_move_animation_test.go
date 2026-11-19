package ui

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

// A list with a MoveAnimation whose rows are kept (RowVersion): as rows go and
// come around them, the kept rows glide as the rebuilt twin's do, frame by
// frame of the Testers' clocks.
func TestRetainRowsMoveAnimation(t *testing.T) {
	ids := make([]int, 60)
	for i := range ids {
		ids[i] = i
	}
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{
			Key:           func(i int) any { return ids[i] },
			RowVersion:    func(i int) uint64 { return uint64(ids[i]) + 1 },
			MoveAnimation: &Animation{Duration: 200 * time.Millisecond},
		}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, len(ids), func(i int) {
					Box(c).Height(40).Children(func() { Textf(c, "Item %d", ids[i]) })
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test tests nothing")
	}
	glide := func(what string) {
		t.Helper()
		if !p.kept.rt.animating {
			t.Fatalf("%s: no row glides: the step tests nothing", what)
		}
		kept := p.kept.rt.rowsKept
		defer func() {
			if p.kept.rt.rowsKept == kept {
				t.Errorf("%s: no row was kept while the rows glided", what)
			}
		}()
		for i := range 16 {
			p.both(func(tt *Tester) { tt.Advance(16 * time.Millisecond) })
			p.frame(fmt.Sprintf("%s %d", what, i))
		}
	}
	ids = slices.Delete(ids, 1, 3) // two rows go near the top
	p.frame("removed")
	glide("closing")
	ids = slices.Insert(ids, 3, 100, 101, 102) // three come among them
	p.frame("inserted")
	glide("opening")
	ids[4], ids[5] = ids[5], ids[4] // two rows trade places
	p.frame("moved")
	glide("moving")
	p.idle(2)
}

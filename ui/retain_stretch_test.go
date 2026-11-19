package ui

import (
	"fmt"
	"testing"
	"time"
)

// A list whose rows are kept, pulled past its top while Android's stretch
// grows it (contentStretch), then let go to settle: the kept rows' worlds and
// hits are the rebuilt twin's at each frame, stretched and not.
func TestRetainRowsStretch(t *testing.T) {
	defer func(k overscroll) { overscrollKind = k }(overscrollKind)
	overscrollKind = overscrollStretch
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, 200, func(i int) {
					Row(c).Padding(4).Children(func() { Textf(c, "row %d", i) })
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test tests nothing")
	}
	p.both(func(tt *Tester) { tt.FingerDown(150, 50) })
	p.both(func(tt *Tester) { tt.Advance(time.Millisecond) })
	p.both(func(tt *Tester) { tt.FingerMove(150, 60) })
	p.both(func(tt *Tester) { tt.Advance(16 * time.Millisecond) })
	p.both(func(tt *Tester) { tt.FingerMove(150, 200) })
	p.frame("stretched")
	if s := p.kept.rt.c.root; listOver(s) >= 0 {
		t.Fatalf("the list is not pulled past its top (%v)", listOver(s))
	}
	p.frame("stretched again")
	p.both(func(tt *Tester) { tt.Advance(200 * time.Millisecond) })
	p.both(func(tt *Tester) { tt.FingerUp(150, 200) })
	for i := range 60 {
		p.both(func(tt *Tester) { tt.Advance(16 * time.Millisecond) })
		p.frame(fmt.Sprintf("settling %d", i))
	}
	p.idle(2)
}

// listOver is how far the first list of the tree shows past its ends, down.
func listOver(n *node) float32 {
	if n.list != nil && n.st != nil {
		return n.st.over[1]
	}
	for c := n.first; c != nil; c = c.next {
		if o := listOver(c); o != 0 {
			return o
		}
	}
	return 0
}

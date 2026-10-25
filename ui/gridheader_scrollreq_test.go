package ui

import (
	"fmt"
	"testing"
)

// scrollGrid shows a grid of n items (read at each frame), a header every
// ten of them if headed.
func scrollGrid(n *int, headed *bool) (*Tester, *GridState) {
	s := &GridState{}
	s.Header = func(i int) bool { return *headed && i%10 == 0 }
	tt := coreNewTester(func(c *context) {
		if *headed {
			s.Header = func(i int) bool { return i%10 == 0 }
		} else {
			s.Header = nil
		}
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, *n, 100, 40, func(i int) {
				if s.Header != nil && s.Header(i) {
					coreTextf(c, "H%d", i)
				} else {
					coreTextf(c, "Item %d", i)
				}
			}).Grow(1)
		})
	}, 448, 300)
	tt.Frame()
	return tt, s
}

// ScrollTo and ScrollIntoView called between two frames, after the number of
// items changed, scroll to the item in the grid the next frame builds, not in
// the rows of the last.
func TestGridViewScrollAfterTheItemsChanged(t *testing.T) {
	n, headed := 40, true
	tt, s := scrollGrid(&n, &headed)
	n = 200
	s.ScrollTo(199, End)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("Item 199"); !ok {
		t.Fatalf("ScrollTo(199, End) after the items grew from 40 to 200 shows %q, not Item 199", tt.Texts())
	}
	n = 400
	s.ScrollIntoView(399)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("Item 399"); !ok {
		t.Fatalf("ScrollIntoView(399) after the items grew to 400 does not show Item 399")
	}
}

// ... and after Header was set on a grid that was laid out without it.
func TestGridViewScrollAfterHeaderWasSet(t *testing.T) {
	n, headed := 200, false
	tt, s := scrollGrid(&n, &headed)
	headed = true
	s.ScrollTo(150, Start)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("Item 151"); !ok {
		t.Fatalf("ScrollTo(150, Start) after Header was set shows %v, not Item 151", fmt.Sprint(tt.Texts()))
	}
}

// The same without headers: the grid scrolls to the item asked for before it
// is first built, with the columns it will have.
func TestGridViewScrollBeforeTheFirstFrame(t *testing.T) {
	s := &GridState{}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, 400, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 300)
	s.ScrollTo(301, Start)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("Item 301"); !ok {
		t.Fatalf("ScrollTo(301, Start) before the first frame shows %v", tt.Texts())
	}
}

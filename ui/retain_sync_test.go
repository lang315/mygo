package ui

import (
	"fmt"
	"testing"
)

// The tests of this file cover upstream changes that meet the rows a list
// keeps across frames.

// A ListState moved to another window (upstream #1664) drops the rows it kept:
// their states are the old window's, which the new one does not hold.
func TestRetainListMovedToAnotherWindowBuildsItsRows(t *testing.T) {
	list := ListState{
		Key:        func(i int) any { return i },
		RowVersion: func(int) uint64 { return 0 },
	}
	// One theme of the app's for both windows, as their default themes would
	// tell the windows apart.
	theme := DarkTheme()
	view := func(c *context) {
		c.SetTheme(theme)
		coreList(c, &list, 20, func(i int) { coreText(c, fmt.Sprint("row ", i)).Height(20) }).Grow(1)
	}
	inFirst := true
	first := coreNewTester(func(c *context) {
		if inFirst {
			view(c)
		}
	}, 200, 200)
	for range 4 {
		first.Frame()
	}
	inFirst = false
	// The new window counts as many frames as the old one did before it shows
	// the list, as the copies' last frame would let them be linked.
	inSecond := false
	second := coreNewTester(func(c *context) {
		if inSecond {
			view(c)
		}
	}, 200, 200)
	for second.rt.frame < first.rt.frame {
		second.Frame()
	}
	inSecond = true
	kept := second.rt.rowsKept
	second.Frame()
	if got := second.rt.rowsKept - kept; got != 0 {
		t.Errorf("the new window linked %d rows the old one kept", got)
	}
	if bad := danglingStates(second.rt, &list); bad > 0 {
		t.Errorf("%d states of the rows linked in the new window are the old window's", bad)
	}
	if _, ok := rowBox(second, &list, 0); !ok {
		t.Fatal("the moved list shows no row 0")
	}
}

// A kept row whose element reads InView follows a single scroll that brings
// the element in (upstream #1661): the commit asks for another frame, and the
// row, whose answer changed, is built in it.
func TestRetainKeptRowReadsInViewAfterOneScroll(t *testing.T) {
	list := ListState{
		Key:        func(i int) any { return i },
		RowVersion: func(int) uint64 { return 0 },
	}
	in := map[int]bool{}
	tt := coreNewTester(func(c *context) {
		coreList(c, &list, 3, func(i int) {
			coreColumn(c).Children(func() {
				coreBox(c).Size(100, 300)
				in[i] = coreBox(c).Key("tail").Size(100, 40).InView()
			})
		}).Grow(1)
	}, 200, 200)
	for range 4 {
		tt.Frame()
	}
	if in[0] {
		t.Fatal("row 0's tail is in view before the scroll")
	}
	kept := tt.rt.rowsKept
	tt.Frame()
	if tt.rt.rowsKept == kept {
		t.Fatal("no row kept in a steady frame: the test shows nothing")
	}
	tt.Scroll(50, 100, 0, 200)
	if !in[0] {
		t.Error("after one scroll, row 0's tail reads out of view")
	}
}

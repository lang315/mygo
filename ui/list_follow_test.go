package ui

import "testing"

// At the end, FollowEnd turned on with rows added before the next frame:
// ui's own semantics, which the widget setter now exposes.
func TestListFollowEndTurnedOnAtTheEndFollowsRowsOfTheSameFrame(t *testing.T) {
	n := 100
	s := ListState{}
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, n, func(i int) { coreBox(c).Height(40) }).Grow(1)
	}, 300, 400)
	s.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if !s.AtEnd() {
		t.Fatal("not at the end")
	}
	s.FollowEnd = true
	n += 20
	tt.Frame()
	if !s.AtEnd() {
		t.Error("at the end, FollowEnd turned on with rows in the same frame does not follow")
	}
}

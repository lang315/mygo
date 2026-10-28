package ui

import "testing"

// The app changes a row's data, and bumps its version, after the list built
// the row in this frame (a later part of the view, or a goroutine updating a
// RowSource between the view and the capture): the copy of the row is recorded
// with the version the row was built with, not the one at the end of the frame.
func TestRetainTheVersionOfACopyIsTheOneTheRowWasBuiltWith(t *testing.T) {
	for _, keep := range []bool{false, true} {
		names := []string{"a", "b", "c", "d", "e"}
		vers := make([]uint64, len(names))
		poke := false
		s := ListState{RowVersion: func(i int) uint64 { return vers[i] }}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				if poke {
					names[2], vers[2] = "first", vers[2]+1 // the row is built with this
				}
				coreList(c, &s, len(names), func(i int) { coreText(c, names[i]) }).Grow(1)
				if poke {
					names[2], vers[2] = "second", vers[2]+1 // after the row was built
					poke = false
				}
			})
		}, 300, 300)
		tt.rt.noRetain = !keep
		for range 3 {
			tt.Frame()
		}
		poke = true
		tt.Frame()
		for range 3 {
			tt.Frame()
		}
		if !tt.HasText("second") {
			t.Errorf("keep=%v: row 2 shows %q, its data is %q", keep, "first", names[2])
		}
	}
}

package ui

import (
	"fmt"
	"sync"
	"testing"
)

// Testers run on goroutines of their own, as the tests of an app may, and
// the first list one builds links the layout of lists: the functions it sets
// are shared by every engine in the process, so setting them is a write that
// the others' reads must not meet. Run with -race.
func TestTestersBuildingListsInParallelDoNotRace(t *testing.T) {
	run := func(g int) {
		for range 10 {
			if g == 0 {
				// No list: lays out and paints what compares paint
				// only below a list's row.
				tt := coreNewTester(func(c *context) {
					coreColumn(c).Children(func() {
						coreText(c, "no list")
						coreButton(c, "a button")
					})
				}, 300, 200)
				tt.Frame()
				tt.Image()
				continue
			}
			var s ListState
			tt := coreNewTester(func(c *context) {
				coreList(c, &s, 50, func(i int) { coreText(c, fmt.Sprint("row ", i)) }).Grow(1)
			}, 300, 200)
			tt.Frame()
			tt.Scroll(150, 100, 0, 200)
			tt.Image()
		}
	}
	// This tests the hook only: the glyph atlas is warmed first to avoid the
	// race documented on Tester.
	run(0)
	run(1)
	// Unlinked again, so that the first list built in the goroutines links.
	comparePaint, inheritsPaint, rowPainter, layoutListFn, relayoutListFn = nil, nil, nil, nil, nil
	paintCompareOnce = sync.Once{}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			run(g)
		}()
	}
	close(start)
	wg.Wait()
}

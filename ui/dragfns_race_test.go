package ui

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/transfer"
)

// Testers run on goroutines of their own, as the tests of an app may, and
// the first source or target of a drag links the engine's side of dragging,
// which every engine of the process reads at each pointer event, whether or
// not its app drags: setting it is a write that those reads must not meet.
// Run with -race.
func TestTestersDraggingInParallelDoNotRace(t *testing.T) {
	data := transfer.New(transfer.NewItem(transfer.Bytes(transfer.Text, []byte("x"))))
	opts := transfer.DropOptions{Formats: []transfer.Format{transfer.Text}, Operations: transfer.Copy}
	run := func(g int) {
		for range 10 {
			var view func(c *context)
			switch g {
			case 0, 1: // no drag
				view = func(c *context) {
					coreColumn(c).Children(func() {
						coreText(c, "no drag")
						coreButton(c, "a button")
					})
				}
			case 2:
				view = func(c *context) {
					coreColumn(c).Gap(20).Children(func() {
						coreBox(c).Size(60, 30).Label("Source").Drag("v")
						coreDrop[string](coreBox(c).Size(100, 60).Label("Target"))
					})
				}
			case 3:
				view = func(c *context) {
					coreColumn(c).Gap(20).Children(func() {
						coreBox(c).Size(60, 30).Label("Source").DragData(data)
						coreDropData(coreBox(c).Size(100, 60).Label("Target"), opts)
					})
				}
			default:
				view = func(c *context) {
					s := &ListState{Reorder: func(from []int, to int) {}}
					coreList(c, s, 5, func(i int) { coreText(c, "row") }).Grow(1)
				}
			}
			tt := coreNewTester(view, 300, 200)
			tt.Frame()
			tt.Press(10, 10)
			tt.Move(30, 30)
			tt.Move(60, 100)
			tt.Release(60, 100)
			tt.Key(0, KeyEscape)
			tt.Image()
		}
	}
	// This tests the hook only: the glyph atlas is warmed first to avoid the
	// race documented on Tester.
	for g := range 5 {
		run(g)
	}
	// Unlinked again, so that the first source or target made in the
	// goroutines links.
	clearDragFns()
	var wg sync.WaitGroup
	start := make(chan struct{})
	for g := range 5 {
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

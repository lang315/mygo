package ui

import (
	"sync"
	"testing"
)

// Testers run on goroutines of their own, as the tests of an app may, and
// the first editor one makes links the editor's functions, which are shared
// by every engine in the process: setting them is a write that the others'
// reads must not meet. Run with -race.
func TestTestersMakingEditorsInParallelDoNotRace(t *testing.T) {
	run := func(g int) {
		for range 10 {
			value := "abc"
			var view func(c *context)
			switch g {
			case 0: // no editor
				view = func(c *context) {
					coreColumn(c).Children(func() {
						coreText(c, "no input")
						coreButton(c, "a button")
					})
				}
			case 1:
				view = func(c *context) { coreTextInput(c, &value) }
			case 2:
				view = func(c *context) { coreTextArea(c, &value) }
			default:
				view = func(c *context) { coreText(c, "select me").Selectable() }
			}
			tt := coreNewTester(view, 300, 200)
			tt.Frame()
			if g != 0 {
				tt.ClickAt(20, 12)
				tt.Key(0, KeyEnd)
			}
			tt.Image()
		}
	}
	// This tests the hook only: the glyph atlas is warmed first to avoid the
	// race documented on Tester.
	for g := range 5 {
		run(g)
	}
	// Unlinked again, so that the first editor made in the goroutines links.
	clearEditorFns()
	prepareEditorTextFn = nil
	editorOnce, selectableTextOnce = sync.Once{}, sync.Once{}
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

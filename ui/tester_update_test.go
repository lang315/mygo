package ui_test

import (
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestTesterUpdateRunsOnTheNextFrame(t *testing.T) {
	status := "idle"
	tt := ui.NewTester(func(c *ui.Context) { ui.Text(c, status) }, 200, 100)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tt.Update(func() { status = "done" })
		}()
	}
	wg.Wait()
	if !tt.HasText("idle") {
		t.Fatalf("Update ran before a frame: %q", tt.Texts())
	}
	tt.Frame()
	if !tt.HasText("done") {
		t.Fatalf("after a frame, texts %q, want done", tt.Texts())
	}
}

// An update queued while the tester settles, as by another update, runs
// before it returns, as Window.Update asks a window for a frame.
func TestTesterUpdateQueuedWhileSettlingRuns(t *testing.T) {
	status := "idle"
	var tt *ui.Tester
	tt = ui.NewTester(func(c *ui.Context) { ui.Text(c, status) }, 200, 100)
	tt.Update(func() {
		status = "first"
		tt.Update(func() { status = "second" })
	})
	tt.Frame()
	if !tt.HasText("second") {
		t.Fatalf("texts %q, want second", tt.Texts())
	}
}

// An update queued while the view only repaints, as an animation drawn
// by a Painter, builds the next frame Advance runs, as Window.Update
// asks a window for a frame built anew.
func TestTesterUpdateBuildsWhileRepainting(t *testing.T) {
	status := "idle"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Text(c, status)
		ui.Box(c).Size(20, 20).Draw(func(p *ui.Painter, r ui.Rect) { p.AnimationFrame() })
	}, 200, 100)
	tt.Advance(20 * time.Millisecond) // repainting, not building
	tt.Update(func() { status = "done" })
	tt.Advance(20 * time.Millisecond)
	if !tt.HasText("done") {
		t.Fatalf("after an update and a frame, texts %q, want done", tt.Texts())
	}
}

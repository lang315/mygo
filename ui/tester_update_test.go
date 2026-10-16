package ui_test

import (
	"sync"
	"testing"

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

//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// DRAFT (stress): the cycle of TestCallbacksDoNotGrowWithWindows, 20 x 11
// times, with a guard on the wait that hangs, and a diagnosis when it does.
func TestZZStressHiddenContentWindows(t *testing.T) {
	var hangs, closedLate int
	var diag []string
	cycle := func(rep, c int) {
		drawn, closed := make(chan struct{}, 1), make(chan struct{}, 1)
		var w *mygo.Window
		mygo.RunOnMain(func() {
			w = mygo.NewWindow(mygo.WindowOptions{Width: 300, Height: 200, Hidden: true, Content: ui.View(func(c *ui.Context) {
				ui.Text(c, "callbacks")
			})})
			w.OnReadyToShow(func() { drawn <- struct{}{} })
			w.OnClosed(func() { closed <- struct{}{} })
		})
		w.Show()
		start := time.Now()
		select {
		case <-drawn:
		case <-time.After(wait.Guard):
			hangs++
			var undrawn int
			var cloaked bool
			mygo.RunOnMain(func() { undrawn, cloaked = win.TestUndrawnCloaked() })
			diag = append(diag, fmt.Sprintf("rep %d cycle %d: no ready-to-show after %v; windows undrawn %d (all cloaked: %v); %s", rep, c, time.Since(start).Round(time.Second), undrawn, cloaked, stressState()))
		}
		web := mygo.NewWindow(mygo.WindowOptions{Hidden: true, URL: "app://localhost/"})
		if _, err := web.Page().Eval("1"); err != nil {
			t.Errorf("rep %d cycle %d: web page: %v", rep, c, err)
		}
		web.Destroy()
		w.Destroy()
		select {
		case <-closed:
		case <-time.After(wait.Guard):
			closedLate++
			diag = append(diag, fmt.Sprintf("rep %d cycle %d: not closed after the guard; %s", rep, c, stressState()))
		}
	}
	for rep := 0; rep < 20 && hangs < 3 && closedLate < 3; rep++ {
		for c := 0; c < 11 && hangs < 3 && closedLate < 3; c++ {
			cycle(rep, c)
		}
	}
	t.Fatalf("DEBUG stress summary: hangs=%d notClosed=%d; final %s\n%s", hangs, closedLate, stressState(), strings.Join(diag, "\n"))
}

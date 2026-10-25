//go:build !js

package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// callbackBudget is how many callbacks the whole run of the GUI tests may
// make: half of the 2000 purego and syscall.NewCallback give a process, which
// are never freed. The tests open every kind of window, menu, tray icon and
// dialog an app does (hello and widget-tour use no more); an app that
// makes callbacks per window or per call runs out, and panics, after a while.
const callbackBudget = 1000

// checkCallbackBudget fails the run, once the tests are done, if they made
// more callbacks than the budget.
func checkCallbackBudget(code int) int {
	n, exact := callbackCount()
	kind := "since the first count"
	if exact {
		kind = "in all"
	}
	line := fmt.Sprintf("e2e: %d callbacks made %s (budget %d)", n, kind, callbackBudget)
	fmt.Fprintln(os.Stderr, line)
	// go test hides the stderr of a package that passes, unless it is run with -v:
	// CI names a file for the number, which a step after the tests prints.
	if p := os.Getenv("MYGO_E2E_CALLBACKS_FILE"); p != "" {
		if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintln(f, line)
			f.Close()
		}
	}
	if n > callbackBudget {
		fmt.Fprintf(os.Stderr, "e2e: FAIL: more callbacks than the budget of %d\n", callbackBudget)
		return 1
	}
	return code
}

// TestCallbackCounter checks the counter itself: a callback is counted.
func TestCallbackCounter(t *testing.T) {
	before, _ := callbackCount()
	for range 3 {
		makeCallback()
	}
	after, _ := callbackCount()
	if after-before != 3 {
		t.Fatalf("3 callbacks made, the counter went from %d to %d", before, after)
	}
}

// Windows opened and closed make no callbacks of their own: those the
// backend needs it made once, whatever the number of windows.
func TestCallbacksDoNotGrowWithWindows(t *testing.T) {
	cycle := func() {
		w := mygo.NewWindow(mygo.WindowOptions{Width: 300, Height: 200, Hidden: true, Content: ui.View(func(c *ui.Context) {
			ui.Text(c, "callbacks")
		})})
		drawn, closed := make(chan struct{}, 1), make(chan struct{}, 1)
		w.OnReadyToShow(func() { drawn <- struct{}{} })
		w.OnClosed(func() { closed <- struct{}{} })
		w.Show()
		<-drawn
		web := mygo.NewWindow(mygo.WindowOptions{Hidden: true, URL: "app://localhost/"})
		if _, err := web.Page().Eval("1"); err != nil {
			t.Fatal(err)
		}
		web.Destroy()
		w.Destroy()
		<-closed
	}
	cycle() // the first of each kind makes what the backend keeps
	before, _ := callbackCount()
	for range 10 {
		cycle()
	}
	if after, _ := callbackCount(); after != before {
		t.Fatalf("10 more windows made %d more callbacks, want none", after-before)
	}
}

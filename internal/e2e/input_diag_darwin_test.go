package e2e

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
)

// lastInput is the last click or drag the tests sent, for what a wait for its
// effect reports when it times out: a click that did nothing leaves nothing
// else to look at, and the state of the window and the app when it was sent
// tells whether it could have been taken.
var lastInput struct {
	sync.Mutex
	what  string
	at    time.Time
	win   *mygo.Window
	state string
}

// noteInput records an input about to be sent to w.
func noteInput(what string, w *mygo.Window) {
	var state string
	mygo.RunOnMain(func() { state = darwin.TestWindowState(w.NativeHandle()) })
	lastInput.Lock()
	defer lastInput.Unlock()
	lastInput.what, lastInput.at, lastInput.win, lastInput.state = what, time.Now(), w, state
}

// frontmostBundleID is the bundle identifier of the frontmost app.
func frontmostBundleID() (id string) {
	mygo.RunOnMain(func() { id = darwin.TestFrontmostBundleID() })
	return id
}

// inputDiag describes the last input and, if its window is still there, the
// state of the window and the app now; empty when the tests sent none or its
// window is gone. It says first if the desktop is locked, which loses inputs.
func inputDiag() string {
	hint := ""
	if why := desktopUnavailable(frontmostBundleID()); why != "" {
		hint = "; " + why
	}
	lastInput.Lock()
	what, at, w, state := lastInput.what, lastInput.at, lastInput.win, lastInput.state
	lastInput.Unlock()
	// The window of an earlier test is gone: its input is not this wait's.
	if w == nil || w.IsDestroyed() {
		return hint
	}
	var now string
	mygo.RunOnMain(func() { now = darwin.TestWindowState(w.NativeHandle()) })
	return hint + fmt.Sprintf("; the last input was a %s %v ago to %s; now %s", what, time.Since(at).Round(time.Millisecond), state, now)
}

// TestInputDiagDescribesTheLastInput checks what a failing wait for the
// effect of a click or drag says about the window the input went to.
func TestInputDiagDescribesTheLastInput(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Width: 200, Height: 100})
	click(w, 10, 10)
	d := inputDiag()
	for _, want := range []string{"click", "key=", "occlusion", "app active=", "frontmost app"} {
		if !strings.Contains(d, want) {
			t.Errorf("inputDiag() = %q, missing %q", d, want)
		}
	}
	drag(w, [][2]float64{{10, 10}, {20, 20}})
	if d := inputDiag(); !strings.Contains(d, "drag") {
		t.Errorf("after a drag, inputDiag() = %q", d)
	}
	// A later test must not report the input of an earlier one.
	w.Destroy()
	if d := inputDiag(); strings.Contains(d, "last input") {
		t.Errorf("after the window is destroyed, inputDiag() = %q, want no input", d)
	}
}

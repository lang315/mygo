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

// keepFront brings the window of the last input back to the front if another
// app took it: what the input did needs the window key, so a test waiting
// for it fails when a user, or a notification, switches apps. A window of an
// active app that is not key stays so, as a flyout that takes no keyboard.
func keepFront() {
	lastInput.Lock()
	w := lastInput.win
	lastInput.Unlock()
	if w == nil || w.IsDestroyed() {
		return
	}
	if active, _ := appActive(); !active && !w.IsFocused() {
		keepFronted.Lock()
		keepFronted.n, keepFronted.title = keepFronted.n+1, w.Title()
		keepFronted.Unlock()
		w.Focus()
	}
}

// keepFronted is what keepFront brought to the front, for a wait that times
// out: each window it focuses takes the key from the one the wait is for.
var keepFronted struct {
	sync.Mutex
	n     int
	title string
}

// keepFrontFocused returns how many times keepFront focused a window, and the
// title of the last.
func keepFrontFocused() (int, string) {
	keepFronted.Lock()
	defer keepFronted.Unlock()
	return keepFronted.n, keepFronted.title
}

// windowDiag describes w and the app for a wait for w that timed out, when no
// input was sent to it: the window's state (key, whether it can be, shown),
// which window is key instead, whether the app is active, what keepFront did
// since the wait began (focusedBefore is keepFrontFocused's count then) and
// how long the test's activation and deactivation of the app took.
func windowDiag(t *testing.T, w *mygo.Window, focusedBefore int) string {
	var state string
	mygo.RunOnMain(func() { state = darwin.TestWindowState(w.NativeHandle()) })
	n, title := keepFrontFocused()
	n -= focusedBefore
	if n == 0 {
		title = ""
	}
	return fmt.Sprintf("%s; keepFront focused %d times (last %q); %s; %s", state, n, title,
		activation.describe(t, "activated", "the app was not activated"),
		deactivation.describe(t, "deactivated", "the app was not deactivated"))
}

// A stopwatch is the time the last activated or deactivated took, and the
// test it was for.
type stopwatch struct {
	sync.Mutex
	test string
	took time.Duration
}

var activation, deactivation stopwatch

// describe says how long the step took in test t, or none if t did not do it.
func (s *stopwatch) describe(t *testing.T, did, none string) string {
	s.Lock()
	defer s.Unlock()
	if s.test != t.Name() {
		return none
	}
	return fmt.Sprintf("%s in %v", did, s.took.Round(time.Millisecond))
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

// TestWindowDiagSaysWhatKeepsAPanelFromBeingKey checks what a wait for a
// window to be key says when it times out, for a panel nothing had sent input
// to: whether the panel can be key and is shown, which window is key instead,
// and whether the app is active.
func TestWindowDiagSaysWhatKeepsAPanelFromBeingKey(t *testing.T) {
	other := newWindow(t, mygo.WindowOptions{Title: "Alpha", Width: 200, Height: 100})
	activated(t, other)
	panel := newWindow(t, mygo.WindowOptions{Title: "Beta", Width: 200, Height: 100, Panel: true, Hidden: true})
	before, _ := keepFrontFocused()
	d := windowDiag(t, panel, before)
	for _, want := range []string{`"Beta"`, "canKey=true", "visible=false", `key window`, `"Alpha"`, "app active=", "keepFront focused 0 times", "activated in ", "the app was not deactivated"} {
		if !strings.Contains(d, want) {
			t.Errorf("windowDiag(t, panel, before) = %q, missing %q", d, want)
		}
	}
	deactivated(t)
	if d := windowDiag(t, panel, before); !strings.Contains(d, "deactivated in ") {
		t.Errorf("after the app was deactivated, windowDiag(t, panel, before) = %q, missing the time it took", d)
	}
}

// TestActivatedLeavesTheAppActiveAndInFront: a window is key a moment before
// its app is active and frontmost, and what a test does next (deactivate the
// app, show a panel while another app is in front) needs the app to have been:
// a runner showed the app not active, nor frontmost, just before it was
// deactivated ("deactivated in 1ms"), and a panel shown after it never became
// key.
func TestActivatedLeavesTheAppActiveAndInFront(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Epsilon", Width: 200, Height: 100})
	deactivated(t) // another app in front: activated brings the app back
	activated(t, w)
	if front, _ := appActive(); !front {
		t.Error("the app is not the frontmost one")
	}
	if !appIsActive() {
		t.Error("the app is not active")
	}
	if !w.IsFocused() {
		t.Error("the window is not key")
	}
	if d := windowDiag(t, w, 0); !strings.Contains(d, "activated in ") {
		t.Errorf("windowDiag = %q, missing the time the activation took", d)
	}
}

// TestKeepFrontCountsWhatItFocuses: a wait that timed out says how many times
// keepFront brought a window to the front, and which, because every one takes
// the key from the window the wait is for. Only a window still there is
// brought: that of an earlier test is gone.
func TestKeepFrontCountsWhatItFocuses(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Gamma", Width: 200, Height: 100, Hidden: true})
	noteInput("click", w)
	deactivated(t)
	before, _ := keepFrontFocused()
	keepFront()
	if n, title := keepFrontFocused(); n != before+1 || title != "Gamma" {
		t.Errorf("after keepFront on a window not key in an inactive app: %d focuses of %q, want %d of %q", n, title, before+1, "Gamma")
	}
	w.Destroy()
	keepFront()
	if n, _ := keepFrontFocused(); n != before+1 {
		t.Errorf("keepFront focused a destroyed window: %d focuses, want %d", n, before+1)
	}
}

// TestGuardLineSaysWhichWaitRanOut: a wait that runs out wait.Guard without
// failing the test (waitKey, and sendFront, which sends the input then) leaves
// a minute in the run and no trace; its line names the wait and says what the
// window and the app were.
func TestGuardLineSaysWhichWaitRanOut(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Delta", Width: 200, Height: 100})
	w.Focus()
	eventually(t, "the window key", w.IsFocused)
	d := guardLine("waitKey", w)
	for _, want := range []string{"waitKey", "wait.Guard", `"Delta"`, "key=", "app active=", "frontmost app", "frontmost bundle "} {
		if !strings.Contains(d, want) {
			t.Errorf("guardLine(waitKey, w) = %q, missing %q", d, want)
		}
	}
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

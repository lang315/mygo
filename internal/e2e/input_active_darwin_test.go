package e2e

import (
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// deactivated makes another app the active one, as a user does by switching
// to it while a test runs, and waits for it.
func deactivated(t *testing.T) {
	t.Helper()
	if !deactivateApp() {
		t.Skip("another app cannot be activated")
	}
	// Not eventually: the waits of the tests bring their window back.
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if active, _ := appActive(); !active {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the app deactivated")
		}
	}
}

// TestClickWhenAnotherAppTookTheFront: an input sent to a window of an app
// that is no longer active is lost, as AppKit drops a click for a window that
// is not key. A user sees the window and clicks it, which brings it back; so
// does click.
func TestClickWhenAnotherAppTookTheFront(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Width: 400, Height: 300})
	w.Page().LoadHTML(`<body style="margin:0"><button id="b" style="width:200px;height:100px" onclick="window.clicked=(window.clicked||0)+1">x</button></body>`, "")
	waitFor(t, w, "document.getElementById('b')")
	deactivated(t)
	click(w, 50, 50)
	waitFor(t, w, "window.clicked === 1")
}

// TestInputBringsTheWindowBack: click and drag leave the window key and its
// app active when another app had taken the front.
func TestInputBringsTheWindowBack(t *testing.T) {
	for name, input := range map[string]func(*mygo.Window){
		"click": func(w *mygo.Window) { click(w, 20, 20) },
		"drag":  func(w *mygo.Window) { drag(w, [][2]float64{{20, 20}, {40, 40}}) },
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindow(t, mygo.WindowOptions{Width: 200, Height: 100})
			eventually(t, "the window key", w.IsFocused)
			deactivated(t)
			input(w)
			if active, _ := appActive(); !active || !w.IsFocused() {
				t.Errorf("after the %s: app active %v, window key %v", name, active, w.IsFocused())
			}
		})
	}
}

// TestFrontmostBundleIDNamesTheFrontApp checks what TestMain and the timeout
// diagnostic read to tell a locked screen, whose frontmost app is the login
// window, from a usable desktop.
func TestFrontmostBundleIDNamesTheFrontApp(t *testing.T) {
	deactivated(t) // the Finder
	eventually(t, "the Finder frontmost", func() bool { return frontmostBundleID() == "com.apple.finder" })
	if why := desktopUnavailable(frontmostBundleID()); why != "" {
		t.Errorf("with the Finder frontmost, the desktop is unavailable: %s", why)
	}
}

// TestWaitBringsTheWindowBack: another app may take the front after an input
// too, while a test waits for what it did, which needs the window key.
func TestWaitBringsTheWindowBack(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Width: 200, Height: 100})
	click(w, 20, 20)
	deactivated(t)
	eventually(t, "the window key again", w.IsFocused)
}

// TestHooksBringTheWindowBack: every hook that sends the window a key, a menu
// item or a click leaves it key in an active app, as click and drag do, when
// another app took the front. A menu item chosen with another app in front
// finds no focused window: File > Revert to Saved did nothing, and the test
// waited five seconds for its sheet.
func TestHooksBringTheWindowBack(t *testing.T) {
	for name, hook := range map[string]func(*mygo.Window){
		"activateMenu": func(w *mygo.Window) { activateMenu(w, "No such menu", "No such item") },
		"pressKey":     func(w *mygo.Window) { pressKey(w, 0, "a") },
		"pressTab":     func(w *mygo.Window) { pressTab(w, false) },
		"compose":      func(w *mygo.Window) { compose(w, "x", 1, false) },
		"clickAndType": func(w *mygo.Window) { clickAndType(w, 20, 20, "x") },
		"sideButton":   func(w *mygo.Window) { sideButton(w, true) },
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindow(t, mygo.WindowOptions{Width: 200, Height: 100})
			eventually(t, "the window key", w.IsFocused)
			deactivated(t)
			hook(w)
			if active, _ := appActive(); !active || !w.IsFocused() {
				t.Errorf("after %s: app active %v, window key %v", name, active, w.IsFocused())
			}
		})
	}
}

// TestClickSurvivesTheFrontChangingJustBeforeIt: the front may change after
// click waited for it, before the event is posted; AppKit drops a click to a
// window of an app that is not active, and the test waited for what it did.
func TestClickSurvivesTheFrontChangingJustBeforeIt(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Width: 400, Height: 300})
	w.Page().LoadHTML(`<body style="margin:0"><button id="b" style="width:200px;height:100px" onclick="window.clicked=(window.clicked||0)+1">x</button></body>`, "")
	waitFor(t, w, "document.getElementById('b')")
	var once sync.Once
	beforeSend = func() { once.Do(func() { deactivated(t) }) }
	defer func() { beforeSend = nil }()
	click(w, 50, 50)
	waitFor(t, w, "window.clicked === 1")
}

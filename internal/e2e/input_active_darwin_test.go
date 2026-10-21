package e2e

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// deactivated makes another app the active one, as a user does by switching
// to it while a test runs, and waits for it: for the workspace to name the
// other app frontmost, and for NSApp to say it is not active, which comes a
// moment after. A window shown in between becomes key, and the deactivation
// still on its way takes the key from it: on a runner a panel shown right
// after never became key, with NSApp active when it was shown.
func deactivated(t *testing.T) {
	t.Helper()
	if !deactivateApp() {
		t.Skip("another app cannot be activated")
	}
	// Not eventually: the waits of the tests bring their window back.
	start := time.Now()
	if !wait.Poll(func() bool { front, _ := appActive(); return !front && !nsAppActive() }) {
		t.Fatalf("the app deactivated never happened (waited %v)", wait.Guard)
	}
	deactivation.Lock()
	deactivation.test, deactivation.took = t.Name(), time.Since(start)
	deactivation.Unlock()
}

// activated brings w's app to the front and waits until it is the frontmost and
// the active app with w key, as it is when a user has clicked its window. A
// window is key a moment before that, so a wait for the window alone starts
// what comes next too soon: on a runner the app was not frontmost when the
// test deactivated it ("deactivated in 1ms"), and a panel shown after it never
// became key. The time it took is in windowDiag. A desktop that never gives
// the activation, as macOS 14 may when it is cooperative, skips the test.
func activated(t *testing.T, w *mygo.Window) {
	t.Helper()
	start := time.Now()
	if !wait.Poll(func() bool {
		// appActive is whether the app is the frontmost one; nsAppActive is what
		// NSApp says, which comes a moment after.
		if front, _ := appActive(); front && nsAppActive() && w.IsFocused() {
			return true
		}
		w.Focus()
		return false
	}) {
		t.Skipf("the app could not become active: the system refused the activation (waited %v); %s", wait.Guard, windowDiag(t, w, 0))
	}
	activation.Lock()
	activation.test, activation.took = t.Name(), time.Since(start)
	activation.Unlock()
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

// keyWithin reports whether w is key within d, polling without keepFront: the
// window is key a moment after its app is active, so a check of both at once
// fails by a few milliseconds on a slow machine.
func keyWithin(w *mygo.Window, d time.Duration) bool {
	for end := time.Now().Add(d); ; time.Sleep(20 * time.Millisecond) {
		if w.IsFocused() {
			return true
		}
		if time.Now().After(end) {
			return false
		}
	}
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
			if active, _ := appActive(); !active {
				t.Errorf("after the %s: app not active", name)
			}
			if !keyWithin(w, time.Second) {
				t.Errorf("after the %s: window not key a second later", name)
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
			// The app is in front when the hook returns; the window is key a
			// moment later, which a poll without keepFront sees.
			if active, _ := appActive(); !active {
				t.Errorf("after %s: app not active", name)
			}
			if !keyWithin(w, time.Second) {
				t.Errorf("after %s: window not key a second later", name)
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

// TestActivateMenuSurvivesTheFrontChangingJustBeforeIt: File > Revert to Saved
// of an app that is not in front does nothing, and the front may change after
// activateMenu waited for it.
func TestActivateMenuSurvivesTheFrontChangingJustBeforeIt(t *testing.T) {
	revertJustAfter(t, func() { deactivated(t) })
}

// TestActivateMenuSurvivesAnotherWindowTakingTheKeyJustBeforeIt: a menu item
// acts on the focused window, which another window of the app may become after
// activateMenu waited for the front. An app just activated has none for a
// moment too: a runner showed Revert to Saved doing nothing then.
func TestActivateMenuSurvivesAnotherWindowTakingTheKeyJustBeforeIt(t *testing.T) {
	other := newWindow(t, mygo.WindowOptions{Width: 200, Height: 100})
	revertJustAfter(t, func() {
		other.Focus()
		eventually(t, "the other window key", func() bool { return mygo.FocusedWindow() == other })
	})
}

// revertJustAfter chooses File > Revert to Saved for an edited document's
// window, and before runs once between activateMenu's wait for the front and
// the menu's activation (beforeSend); the sheet asking to revert must show.
func revertJustAfter(t *testing.T, before func()) {
	t.Helper()
	mygo.App.SetPath(mygo.PathUserData, t.TempDir())
	defer mygo.App.SetPath(mygo.PathUserData, "")
	docs := notes()
	old := mygo.App.Menu()
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, {Role: mygo.RoleFileMenu}}))
	defer mygo.App.SetMenu(old)
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	path := filepath.Join(dir, "front.e2enote")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := docs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	w := doc.Window()
	t.Cleanup(func() {
		doc.SetEdited(false)
		if !w.IsDestroyed() {
			w.Destroy()
		}
	})
	eventually(t, "the window key", w.IsFocused)
	doc.Set("unwanted")
	eventually(t, "the edited mark", func() bool { _, _, _, edited, _ := documentChrome(w); return edited })
	var once sync.Once
	beforeSend = func() { once.Do(before) }
	defer func() { beforeSend = nil }()
	if err := activateMenu(w, "File", "Revert to Saved"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the sheet asking to revert", func() bool {
		_, buttons, ok, _ := sheet(w)
		return ok && len(buttons) == 2
	})
	endSheetButton(w, 1)
}

// TestClickAndTypeSurvivesTheFrontChangingJustBeforeIt is the same for a click
// and the text typed after it.
func TestClickAndTypeSurvivesTheFrontChangingJustBeforeIt(t *testing.T) {
	var frames atomic.Int32
	st := &editor.State{Buffer: editor.NewBuffer("first\ncafe")}
	w := newWindow(t, mygo.WindowOptions{Title: "Code editor", Width: 500, Height: 240, Content: ui.View(func(c *ui.Context) {
		editor.View(c, st).Fill().AutoFocus()
		frames.Add(1)
	})})
	eventually(t, "an editor frame", func() bool { return frames.Load() > 0 })
	var once sync.Once
	beforeSend = func() { once.Do(func() { deactivated(t) }) }
	defer func() { beforeSend = nil }()
	if !clickAndType(w, 400, 30, "!") {
		t.Skip("typing automation unavailable")
	}
	eventually(t, "the text typed", func() bool {
		var text string
		mygo.RunOnMain(func() { text = st.Buffer.String() })
		return text == "first\ncafe!"
	})
}

// TestClickOnAKeyPanelDoesNotWaitForTheFront: a panel that takes no activation
// is key while another app is in front; a click on it is sent at once, not
// after waiting for an app that stays in the background.
func TestClickOnAKeyPanelDoesNotWaitForTheFront(t *testing.T) {
	focused, _ := keepFrontFocused()
	other := newWindow(t, mygo.WindowOptions{Title: "Other", Width: 300, Height: 200})
	// The app is active and in front before another app takes the front, as
	// when a user switches away: its window being key is not enough.
	activated(t, other)
	panel := newWindow(t, mygo.WindowOptions{Title: "Panel", Width: 400, Height: 200, Panel: true, Hidden: true})
	panel.Page().LoadHTML(`<body style="margin:0"><button id="b" style="width:200px;height:100px" onclick="window.clicked=(window.clicked||0)+1">x</button></body>`, "")
	waitFor(t, panel, "document.getElementById('b')")
	deactivated(t)
	panel.Show()
	// Without keepFront: the test wants the app in the background, and
	// keepFront would focus a window of it, which takes the key from the
	// panel and may activate the app. windowDiag counts what it did in the
	// waits before.
	if !wait.Poll(panel.IsFocused) {
		t.Fatalf("the panel key never happened (waited %v); %s", wait.Guard, windowDiag(t, panel, focused))
	}
	if active, _ := appActive(); active {
		t.Skip("showing the panel activated the app")
	}
	// Counted, not timed: sendFront tries once for each wait for the front.
	tries := 0
	beforeSend = func() { tries++ }
	defer func() { beforeSend = nil }()
	click(panel, 50, 50)
	if tries != 1 {
		t.Fatalf("a click on a key panel was tried %d times: it waited for the app to come to the front", tries)
	}
	waitFor(t, panel, "window.clicked === 1")
}

// TestInFront: input reaches a window as a user's does when its app is in
// front and active, or when it is a panel that takes no activation, which is
// key while another app is in front. A window that is key in an app that was
// just deactivated is not: a CI runner showed it resigning key a moment later.
// Nor is one of an app that is frontmost and not yet active, which a runner
// showed too: AppKit drops a click then, and the window is key a moment later.
func TestInFront(t *testing.T) {
	for _, tt := range []struct {
		frontmost, active, key, panel, want bool
	}{
		{true, true, true, false, true},
		{true, true, false, false, true}, // a flyout that takes no keyboard
		{false, true, true, true, true},  // a key panel, another app in front
		{true, false, true, true, true},
		{false, true, false, true, false},
		{true, false, false, false, false}, // frontmost, not active yet: the click is dropped
		{true, false, true, false, false},
		{false, true, true, false, false}, // the stale key window of an app just deactivated
		{false, false, false, false, false},
	} {
		if got := inFront(tt.frontmost, tt.active, tt.key, tt.panel); got != tt.want {
			t.Errorf("inFront(frontmost %v, active %v, key %v, panel %v) = %v, want %v", tt.frontmost, tt.active, tt.key, tt.panel, got, tt.want)
		}
	}
}

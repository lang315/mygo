//go:build darwin

package e2e

import (
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/internal/wait"
)

// recentDocumentRecorded reports whether path is among the app's recent
// documents. macOS 15 keeps none for a process without a bundle identifier,
// as this test binary ("Invalid bundle identifier: <nil>"); macOS 26 does.
func recentDocumentRecorded(path string) (recorded, supported bool) {
	version, _ := syscall.Sysctl("kern.osproductversion")
	major, _, _ := strings.Cut(version, ".")
	if n, err := strconv.Atoi(major); err == nil && n < 26 {
		return false, false
	}
	var paths []string
	mygo.RunOnMain(func() { paths = darwin.TestRecentDocuments() })
	return slices.Contains(paths, path), true
}

func activateMenu(w *mygo.Window, path ...string) (err error) {
	sendFocused(w, "activateMenu", true, func() { err = darwin.TestPerformMenuItem(path...) })
	return err
}

func pressShortcut(key string, shift bool) (handled bool, supported bool) {
	mygo.RunOnMain(func() { handled = darwin.TestPerformKeyEquivalent(key, shift) })
	return handled, true
}

func endSheet(w *mygo.Window) (ok bool, supported bool) {
	mygo.RunOnMain(func() { ok = darwin.TestEndSheet(w.NativeHandle()) })
	return ok, true
}

// beforeSend, when set, runs in sendFront between the wait for the front and
// the input's posting: a test makes another app take the front then, as a
// user's switch may at any instant.
var beforeSend func()

// inFront says whether input reaches a window as a user's does: its app is in
// front and active, or it is a panel that takes no activation, which is key
// while another app is in front. A window that is key in an app that was just
// deactivated does not count: it resigns key a moment later. Nor does one of
// an app that is frontmost and not active yet: AppKit drops a click to it.
func inFront(frontmost, active, key, panel bool) bool { return frontmost && active || key && panel }

// sendFront gives w its input: send runs on the main thread. The front may
// change between the wait for it and the input, and AppKit drops a click to
// an app that is not active, a menu of one has no focused window: so the
// check and send are one step on the main thread, and an input that found
// the window not in front (inFront) waits for the front again. Each try
// brings the window back (waitKey), so the front comes; wait.Guard only keeps
// a desktop that never gives it from hanging the test: the input is sent then
// anyway, which is why this is a loop and not wait.Poll. The guard bounds
// each wait: the deadline here runs from the start of the loop, so waitKey
// waiting its guard out sends the input at once, but a desktop that never
// gives the front still holds an input about a minute here, and click's own
// waits (waitKey, waitOnScreen) add theirs. TestMain stops at a locked screen.
func sendFront(w *mygo.Window, what string, send func()) { sendFocused(w, what, false, send) }

// sendFocused is sendFront, and with focused it also waits for w to be the
// app's focused window (FocusedWindow), which a menu item acts on: an app just
// activated is active a moment before it has one, as a runner showed, and
// another of its windows may be it. Each try that found another focuses w.
func sendFocused(w *mygo.Window, what string, focused bool, send func()) {
	panel := w.IsPanel()
	for deadline := time.Now().Add(wait.Guard); ; time.Sleep(20 * time.Millisecond) {
		waitKey(w)
		noteInput(what, w)
		if f := beforeSend; f != nil {
			f()
		}
		var sent, late, other bool
		mygo.RunOnMain(func() {
			other = focused && mygo.FocusedWindow() != w
			front := !other && inFront(darwin.TestIsFrontmost(), darwin.TestIsActive(), darwin.TestIsKeyWindow(w.NativeHandle()), panel)
			late = !front && time.Now().After(deadline)
			if sent = front || late; sent {
				send()
			}
		})
		if late {
			guardReached("sendFront", w)
		}
		if sent {
			return
		}
		if other {
			w.Focus()
		}
	}
}

func click(w *mygo.Window, x, y float64) bool {
	waitKey(w)
	waitOnScreen(w)
	// The wait for the screen took long enough for another app to take the
	// front again: sendFront waits for it once more.
	sendFront(w, "click", func() { darwin.TestClick(w.NativeHandle(), x, y) })
	return true
}

// appClick clicks through the application, whose event monitors see it.
func appClick(w *mygo.Window, x, y float64) bool {
	waitOnScreen(w)
	mygo.RunOnMain(func() { darwin.TestAppClick(w.NativeHandle(), x, y) })
	return true
}

// heldAppClick is appClick with the button held down, as a user clicks:
// what the press did runs before the release, the work it left for the
// main thread too.
func heldAppClick(w *mygo.Window, x, y float64) bool {
	waitOnScreen(w)
	mygo.RunOnMain(func() { darwin.TestAppPress(w.NativeHandle(), x, y, true) })
	mygo.RunOnMain(func() {}) // after what the press posted
	mygo.RunOnMain(func() { darwin.TestAppPress(w.NativeHandle(), x, y, false) })
	return true
}

// waitOnScreen waits for a window to show on screen, as a user sees a
// window before clicking it: a window shows a while after it is ordered
// front and made key, longer on a busy runner, and AppKit puts a click
// made for it before then elsewhere in it, as outside its content, where
// it clicks nothing. wait.Guard is a hang guard: it says so if the window
// did not show by then, before the click that will miss.
func waitOnScreen(w *mygo.Window) {
	if !wait.Poll(func() (on bool) {
		mygo.RunOnMain(func() { on = darwin.TestOnScreen(w.NativeHandle()) })
		return on
	}) {
		log.Printf("e2e: the window %q did not show on screen in %v: a click made for it misses", w.Title(), wait.Guard)
	}
}

// waitKey brings a window back to the front, until it is (wait.Guard only
// keeps a desktop that never gives it from hanging the test), if another app
// took it: AppKit drops a click or drag for a window that is not key while
// its app is not active, as when a user switches to another app during a
// test, or a notification does. A window of an active app that is not key
// stays so, as a flyout that takes no keyboard.
func waitKey(w *mygo.Window) {
	if !wait.Poll(func() bool {
		if active, _ := appActive(); active || w.IsFocused() {
			return true
		}
		w.Focus()
		return false
	}) {
		guardReached("waitKey", w)
	}
}

// guardReached logs a wait that ran out wait.Guard without failing the test:
// the window did not come to the front for a minute, and the input goes on.
// The line tells which wait, and what the window and the app were.
func guardReached(what string, w *mygo.Window) { log.Print(guardLine(what, w)) }

// guardLine is the line guardReached logs.
func guardLine(what string, w *mygo.Window) string {
	var state string
	mygo.RunOnMain(func() { state = darwin.TestWindowState(w.NativeHandle()) })
	return fmt.Sprintf("e2e: %s ran out wait.Guard (%v) without the front: %s; frontmost bundle %q", what, wait.Guard, state, frontmostBundleID())
}

func sideButton(w *mygo.Window, back bool) (ok bool) {
	sendFront(w, "sideButton", func() { ok = darwin.TestSideButton(w.NativeHandle(), back) })
	return ok
}

func drag(w *mygo.Window, points [][2]float64) bool {
	sendFront(w, "drag", func() { darwin.TestDrag(w.NativeHandle(), points) })
	return true
}

func webViewAttached(w *mygo.Window) (attached bool, supported bool) {
	mygo.RunOnMain(func() { attached = darwin.TestWebViewAttached(w.NativeHandle()) })
	return attached, true
}

// deactivateApp activates the Finder, as the user would.
func deactivateApp() bool {
	return exec.Command("open", "-a", "Finder").Run() == nil
}

func appActive() (active bool, supported bool) {
	mygo.RunOnMain(func() { active = darwin.TestIsFrontmost() })
	return active, true
}

// nsAppActive reports whether NSApp says the app is active. appActive says
// whether it is the frontmost one, which the workspace names a moment before
// NSApp says it is active (see inFront).
func nsAppActive() (active bool) {
	mygo.RunOnMain(func() { active = darwin.TestIsActive() })
	return active
}

func collectionBehavior(w *mygo.Window) (behavior uint, supported bool) {
	mygo.RunOnMain(func() { behavior = darwin.TestCollectionBehavior(w.NativeHandle()) })
	return behavior, true
}

func dockDevTools(w *mygo.Window) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestDockInspector(w.NativeHandle()) })
	return ok
}

func dockedDevToolsPlace(w *mygo.Window) (place string) {
	mygo.RunOnMain(func() { place = darwin.TestInspectorPlace(w.NativeHandle()) })
	return place
}

// Context menus track the mouse in a modal loop; not automated.
func dismissPopups() (int, bool) { return 0, false }

func setDroppedFiles(w *mygo.Window, paths []string) bool {
	mygo.RunOnMain(func() { darwin.TestSetDroppedFiles(w.NativeHandle(), paths) })
	return true
}

func dockTileImage() (png []byte, supported bool) {
	mygo.RunOnMain(func() { png = darwin.TestDockTileImage() })
	return png, true
}

func defaultURLHandler(string) (string, bool) { return "", false }

func dockMenu(click int) (titles []string, supported bool) {
	mygo.RunOnMain(func() { titles = darwin.TestDockMenu(click) })
	return titles, true
}

// Posting key events needs the accessibility permission on macOS.
func pressCtrlShiftK() bool { return false }

// Only Linux binds global shortcuts through a desktop portal.
func usePortalShortcuts() (func(), bool) { return nil, false }

func pressKeys(...string) bool               { return false }
func markedText(*mygo.Window) (string, bool) { return "", false }

func fullScreenHidesToolbar(w *mygo.Window) (hides bool, supported bool) {
	mygo.RunOnMain(func() { hides = darwin.TestFullScreenHidesToolbar(w.NativeHandle()) })
	return hides, true
}

func trafficLights(w *mygo.Window) (x, y float64, supported bool) {
	mygo.RunOnMain(func() { x, y = darwin.TestTrafficLights(w.NativeHandle()) })
	return x, y, true
}

func setWindowAppearance(w *mygo.Window, dark bool) {
	mygo.RunOnMain(func() { darwin.TestSetWindowAppearance(w.NativeHandle(), dark) })
}

// Frameless windows keep native resize borders here.
func movePointer(int, int) bool                { return false }
func pressButton(bool) bool                    { return false }
func resizeCursor(*mygo.Window) (string, bool) { return "", false }

// The menu bar belongs to the application on macOS.
func menuBarShown(*mygo.Window) (bool, bool)                { return false, false }
func activateAccelerator(*mygo.Window, string) (bool, bool) { return false, false }
func enterMenuBar(*mygo.Window, string) (bool, bool, bool)  { return false, false, false }
func openMenus(*mygo.Window, byte) bool                     { return false }
func closeMenus()                                           {}

// The traffic lights are AppKit's own.
func titleButtons(*mygo.Window) ([]string, bool) { return nil, false }
func pressTitleButton(*mygo.Window, string) bool { return false }

func topNonClient(*mygo.Window) (int32, int32, bool) { return 0, 0, false }

func menuBarColors(*mygo.Window) (uint32, uint32, bool) { return 0, 0, false }

func titleButtonColor(*mygo.Window, string) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

// AppKit composes the window: there is no redirection bitmap to go without.
func composition(*mygo.Window) (bool, bool, bool) { return false, false, false }
func loseComposition(bool) bool                   { return false }

// WebKit creates web views with their windows.
func failWebViews(int) bool { return false }

func clickAndType(w *mygo.Window, x, y float64, text string) (ok bool) {
	sendFront(w, "clickAndType", func() { ok = darwin.TestClickAndType(w.NativeHandle(), x, y, text) })
	return ok
}

// pressKey presses the key of a virtual key code in a window of native UI,
// through interpretKeyEvents. A real input method does not see the key:
// darwin.TestPostKey's keys it does.
func pressKey(w *mygo.Window, code uint16, chars string) (ok bool) {
	sendFront(w, "pressKey", func() { ok = darwin.TestKey(w.NativeHandle(), code, chars) })
	return ok
}

// AppKit routes a drag to the view under the pointer, which
// dropFiles, dropping on the content's, skips.
func dragFiles(*mygo.Window, float64, float64, []string) (bool, bool) { return false, false }

// pressTab presses Tab in a window of native UI, with Shift when back, as
// the keyboard does: in the content or in the page of a web view.
func pressTab(w *mygo.Window, back bool) (ok bool) {
	sendFront(w, "pressTab", func() { ok = darwin.TestKeyShift(w.NativeHandle(), 48, "\t", back) })
	return ok
}

func compose(w *mygo.Window, text string, caret int, commit bool) (ok bool) {
	sendFront(w, "compose", func() { ok = darwin.TestCompose(w.NativeHandle(), text, caret, commit) })
	return ok
}

func controlClick(w *mygo.Window, x, y float64) bool {
	mygo.RunOnMain(func() { darwin.TestControlClick(w.NativeHandle(), x, y) })
	return true
}

func composeOver(w *mygo.Window, text string, caret int, commit bool, from, length int) (ok bool) {
	sendFront(w, "composeOver", func() { ok = darwin.TestComposeOver(w.NativeHandle(), text, caret, commit, from, length) })
	return ok
}

func inputClient(w *mygo.Window) (selected [2]int, document string, ok bool) {
	mygo.RunOnMain(func() { selected, document = darwin.TestInputClient(w.NativeHandle()) })
	return selected, document, true
}

func dropFiles(w *mygo.Window, x, y float64, paths []string) (over, dropped, ok bool) {
	mygo.RunOnMain(func() { over, dropped = darwin.TestDropFiles(w.NativeHandle(), x, y, paths) })
	return over, dropped, true
}

// The roles of elements in the accessibility API of AppKit.
const roleText, roleButton, roleCheckBox, roleTextField, roleSlider = "AXStaticText", "AXButton", "AXCheckBox", "AXTextField", "AXSlider"

// roleListItem is the role of the rows of a List.
const roleListItem = "AXRow/AXTableRow"

func accessibility(w *mygo.Window) (nodes []accessNode, ok bool) {
	mygo.RunOnMain(func() {
		for _, n := range darwin.TestAccessibility(w.NativeHandle()) {
			role := n.Role
			if n.Subrole != "" {
				role += "/" + n.Subrole
			}
			nodes = append(nodes, accessNode{role, n.Label, n.Value})
		}
	})
	return nodes, true
}

// axAttribute reads an attribute of an element through AppKit's older
// accessibility API, which only macOS has.
func axAttribute(w *mygo.Window, label, attr string) (value string, settable, named, ok bool) {
	mygo.RunOnMain(func() { value, settable, named, ok = darwin.TestAccessibilityAttribute(w.NativeHandle(), label, attr) })
	return value, settable, named, ok
}

func accessPerform(w *mygo.Window, label, action, value string) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestAccessibilityPerform(w.NativeHandle(), label, action, value) })
	return ok
}

// observe has key-value observing watch the view of a window showing
// native UI and its elements, until stop.
func observe(w *mygo.Window) (stop func(), supported bool) {
	var end func()
	mygo.RunOnMain(func() { end = darwin.TestObserve(w.NativeHandle()) })
	return func() { mygo.RunOnMain(end) }, true
}

// Only Linux draws native UI in a GtkGLArea, nor waits to load the GPU's
// driver.
func glSurface(*mygo.Window) (string, []byte, int, int, bool) { return "", nil, 0, 0, false }
func memoryUI(*testing.T) bool                                { return false }
func surfaceOnScreen(*mygo.Window) ([]byte, bool)             { return nil, false }

// Only Windows reads the screen over native UI.
func screenColor(*mygo.Window, float64, float64) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

// Context menus are not automated on macOS: one shown waits for the user.
func rightClick(*mygo.Window, float64, float64) bool { return false }
func popupMenus() ([][]string, bool)                 { return nil, false }
func choosePopupItem(string) bool                    { return false }

// pressEscape presses Escape in the key window, as the keyboard does.
func pressEscape(*mygo.Window) bool {
	mygo.RunOnMain(func() { darwin.TestAppKey(53, "\x1b") })
	return true
}

func documentChrome(w *mygo.Window) (title, file string, icon, edited, supported bool) {
	mygo.RunOnMain(func() { title, file, icon, edited = darwin.TestDocumentChrome(w.NativeHandle()) })
	return title, file, icon, edited, true
}

func sheet(w *mygo.Window) (texts, buttons []string, ok, supported bool) {
	mygo.RunOnMain(func() { texts, buttons, ok = darwin.TestSheet(w.NativeHandle()) })
	return texts, buttons, ok, true
}

func endSheetButton(w *mygo.Window, i int) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestEndSheetButton(w.NativeHandle(), i) })
	return ok
}

// Panels' window styles are Windows'.
func panelStyles(*mygo.Window) (bool, bool, bool, bool) { return false, false, false, false }

func menuIcon(w *mygo.Window, path ...string) (has, template, supported bool, err error) {
	mygo.RunOnMain(func() { has, template, err = darwin.TestMenuItemImage(path...) })
	return has, template, true, err
}

func shortcutDrawn(*mygo.Window, ...string) (bool, []byte, bool, error) {
	return false, nil, false, nil
}

// trayMenuIcons returns the items of the menu the tray's status item holds
// by title, true for those with an image.
func trayMenuIcons() (icons map[string]bool, supported bool, err error) {
	mygo.RunOnMain(func() { icons = darwin.TestTrayMenuImages() })
	return icons, true, nil
}

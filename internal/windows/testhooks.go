//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/accelerator"
)

// The functions in this file drive native UI the way a user would, for the
// GUI tests in internal/e2e. They must run on the main thread.

// TestActivateMenuItem activates an item of a window's menu bar, found by
// the labels along its path, e.g. TestActivateMenuItem(hwnd, "File", "New").
// plugins/automation's menu request chooses items with it too.
func TestActivateMenuItem(hwnd uintptr, path ...string) error {
	w := theBackend.windows[hwnd]
	if w == nil || w.menu == nil {
		return fmt.Errorf("mygo: the window has no menu bar")
	}
	items := w.menu.Items
	for i, label := range path {
		found := false
		for _, it := range items {
			if it.Label != label {
				continue
			}
			found = true
			if i == len(path)-1 {
				for cmd, e := range theBackend.menus.entries {
					if e.uid == it.ID && e.owner == w.owner {
						theBackend.menuCommand(cmd, w)
						return nil
					}
				}
				return fmt.Errorf("mygo: menu item %q is not a command", label)
			}
			if it.Submenu == nil {
				return fmt.Errorf("mygo: menu %q has no submenu", label)
			}
			items = it.Submenu.Items
			break
		}
		if !found {
			return fmt.Errorf("mygo: no menu item %q", label)
		}
	}
	return nil
}

// menuItemHasIcon reports whether the item at pos of menu shows an icon: a
// bitmap, or the icon of an item MyGo draws (menudraw.go).
func menuItemHasIcon(menu uintptr, pos int) (bool, error) {
	mi := menuItemInfo{Mask: miimBitmap | miimData}
	mi.Size = uint32(unsafe.Sizeof(mi))
	if ok, _, err := procGetMenuItemInfoW.Call(menu, uintptr(pos), 1, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return false, err
	}
	l := theBackend.menus.looks[mi.ItemData]
	return mi.BmpItem != 0 || l != nil && l.icon != nil, nil
}

// TestMenuBarShown reports whether a window shows its menu bar.
func TestMenuBarShown(hwnd uintptr) bool {
	bar, _, _ := user32.NewProc("GetMenu").Call(hwnd)
	return bar != 0
}

// TestEnterMenuBar takes the keyboard to a window's menu bar, as Alt and
// F10 do, and returns once it leaves the menus. It reports false when the
// window cannot come to the front, which the keyboard needs.
func TestEnterMenuBar(hwnd uintptr) bool {
	if !TestForeground(hwnd) {
		return false
	}
	procSendMessageW.Call(hwnd, wmSysCommand, scKeyMenu, 0)
	return true
}

// TestForeground brings a window to the front, where the keyboard types,
// and reports whether it came.
func TestForeground(hwnd uintptr) bool {
	procSetForegroundWindow.Call(hwnd)
	fg, _, _ := procGetForegroundWindow.Call()
	return fg == hwnd
}

// TestForegroundState describes who is in front, for a test that finds a
// window not where it should be: the window in front (its class, title and
// process), how long ago the last input came (the right to take the front
// depends on it), and whether the window is visible or minimized. It asks for
// nothing, so it leaves the front as it is.
func TestForegroundState(hwnd uintptr) string {
	fg, _, _ := procGetForegroundWindow.Call()
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	iconic, _, _ := procIsIconic.Call(hwnd)
	return fmt.Sprintf("in front: %s; last input %v ago; the window %#x is visible %v, minimized %v",
		describeWindow(fg), power{}.IdleTime(), hwnd, visible != 0, iconic != 0)
}

// TestForegroundDiag describes what a window that does not come to the front
// is up against: TestForegroundState, and what SetForegroundWindow returns for
// the window asked again now, and who is in front after. The first ask is
// Focus's, which discards the result.
func TestForegroundDiag(hwnd uintptr) string {
	state := TestForegroundState(hwnd)
	ok, _, errno := procSetForegroundWindow.Call(hwnd)
	fg, _, _ := procGetForegroundWindow.Call()
	return fmt.Sprintf("%s; SetForegroundWindow asked again returned %v (%v), then in front: %s",
		state, ok != 0, errno, describeWindow(fg))
}

// TestGrantForeground makes this process the one that received the last
// input, by sending F24 down and up, a key nothing uses. Windows lets that
// process take the front from the app in front (SetForegroundWindow), as it
// does the app a user is working in. A process nobody has used does not have
// it: a window of one started a while ago, as the tests' is once the webview
// warmed up, stays behind the app in front, and the right stays until
// another process gets input.
func TestGrantForeground() {
	const vkF24, keyUp = 0x87, 0x0002
	key := user32.NewProc("keybd_event")
	key.Call(vkF24, 0, 0, 0)
	key.Call(vkF24, 0, keyUp, 0)
}

// TestEndMenu leaves the menus, as Escape does.
func TestEndMenu() { procEndMenu.Call() }

// TestPanelStyles reports whether a window is a topmost tool window
// without a window menu, owned by the hidden application window, as
// panels are.
func TestPanelStyles(hwnd uintptr) (tool, topmost, owned bool) {
	ex := windowLong(hwnd, gwlExStyle)
	owner, _, _ := procGetNextWindow.Call(hwnd, gwOwner)
	menu := windowLong(hwnd, gwlStyle)&wsSysMenu != 0
	return ex&wsExToolWindow != 0 && !menu, ex&wsExTopmost != 0, owner != 0 && owner == theBackend.appHwnd
}

// TestActivateAccelerator runs the menu item of a window's shortcut, as its
// keys do in the page, and reports whether there is one.
func TestActivateAccelerator(hwnd uintptr, acc string) bool {
	w := theBackend.windows[hwnd]
	a, err := accelerator.Parse(acc, "windows")
	if w == nil || err != nil {
		return false
	}
	vk, ok := virtualKey(a.Key)
	if !ok {
		return false
	}
	cmd, ok := w.pageAccelerator(accelKey{vk: vk, mods: a.Modifiers})
	if ok {
		theBackend.menuCommand(cmd, w)
	}
	return ok
}

// TestSetDroppedFiles makes paths the files of the next drop on a window's
// page, as if they had been dragged there.
func TestSetDroppedFiles(hwnd uintptr, paths []string) {
	if w := theBackend.windows[hwnd]; w != nil {
		w.dropped = paths
	}
}

// TestPressKeys presses virtual keys together and releases them, like a
// keyboard would.
func TestPressKeys(keys ...byte) {
	TestKeysDown(keys...)
	TestKeysUp(keys...)
}

// TestKeysDown presses virtual keys, in order, and holds them.
func TestKeysDown(keys ...byte) {
	proc := user32.NewProc("keybd_event")
	for _, k := range keys {
		proc.Call(uintptr(k), 0, 0, 0)
	}
}

// TestKeysUp releases virtual keys, the last first.
func TestKeysUp(keys ...byte) {
	proc := user32.NewProc("keybd_event")
	const keyUp = 0x2
	for i := len(keys) - 1; i >= 0; i-- {
		proc.Call(uintptr(keys[i]), 0, keyUp, 0)
	}
}

// TestKeyToggled reports the toggle bit of a virtual key's state on the
// calling thread, the main one, which flips each time the thread reads
// the key pressed.
func TestKeyToggled(vk byte) bool {
	r, _, _ := procGetKeyState.Call(uintptr(vk))
	return r&1 != 0
}

// TestCaptionButtons returns the window controls of a window with a hidden
// title bar, left to right, named after what WM_NCHITTEST answers over
// their middles, which Windows' snap layouts rely on.
func TestCaptionButtons(hwnd uintptr) []string {
	w := theBackend.windows[hwnd]
	if w == nil || w.caption == nil {
		return nil
	}
	var names []string
	for i := range w.caption.shown {
		hit, _, _ := procSendMessageW.Call(w.caption.buttons, wmNCHitTest, 0, captionButtonPoint(w.caption, i))
		switch hit {
		case htMinButton:
			names = append(names, "minimize")
		case htMaxButton:
			names = append(names, "maximize")
		case htClose:
			names = append(names, "close")
		default:
			names = append(names, fmt.Sprint(hit))
		}
	}
	return names
}

// TestPressCaptionButton clicks a window control of a window with a hidden
// title bar: "minimize", "maximize" or "close".
func TestPressCaptionButton(hwnd uintptr, name string) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.caption == nil {
		return false
	}
	for i, b := range w.caption.shown {
		if [...]string{"minimize", "maximize", "close"}[b] != name {
			continue
		}
		lp := captionButtonPoint(w.caption, i)
		procSendMessageW.Call(w.caption.buttons, wmNCLButtonDown, b.hitTest(), lp)
		procSendMessageW.Call(w.caption.buttons, wmNCLButtonUp, b.hitTest(), lp)
		return true
	}
	return false
}

// captionButtonPoint is the screen position of the middle of the i-th
// button, as the lParam of mouse messages.
func captionButtonPoint(c *captionBar, i int) uintptr {
	var r rect
	procGetWindowRect.Call(c.buttons, uintptr(unsafe.Pointer(&r)))
	bw := (r.Right - r.Left) / int32(len(c.shown))
	x, y := r.Left+int32(i)*bw+bw/2, r.Top+(r.Bottom-r.Top)/2
	return uintptr(uint16(x)) | uintptr(uint16(y))<<16
}

// TestCaptionColor puts the pointer over a window control of a hidden
// title bar, "minimize", "maximize" or "close", as far as the control
// knows, and returns the color on the screen beside its glyph once Windows
// has composed it: what the user sees, through UpdateLayeredWindow or
// DirectComposition. The pointer leaves after.
func TestCaptionColor(hwnd uintptr, name string) (r, g, b uint8, ok bool) {
	w := theBackend.windows[hwnd]
	if w == nil || w.caption == nil {
		return 0, 0, 0, false
	}
	c := w.caption
	for i, btn := range c.shown {
		if [...]string{"minimize", "maximize", "close"}[btn] != name {
			continue
		}
		c.setHot(i)
		defer c.setHot(-1)
		dwmapi.NewProc("DwmFlush").Call() // until the next frame shows what was drawn
		var rc rect
		procGetWindowRect.Call(c.buttons, uintptr(unsafe.Pointer(&rc)))
		bw := (rc.Right - rc.Left) / int32(len(c.shown))
		x, y := rc.Left+int32(i)*bw+3, rc.Top+(rc.Bottom-rc.Top)/2
		screen, _, _ := procGetDC.Call(0)
		defer procReleaseDC.Call(0, screen)
		px, _, _ := gdi32.NewProc("GetPixel").Call(screen, uintptr(x), uintptr(y))
		if px == 0xFFFFFFFF { // CLR_INVALID
			return 0, 0, 0, false
		}
		return uint8(px), uint8(px >> 8), uint8(px >> 16), true
	}
	return 0, 0, 0, false
}

// TestComposed reports whether a window has no redirection bitmap, for the
// material behind its page, and whether the controls of its hidden title
// bar show through the DirectComposition device in use.
func TestComposed(hwnd uintptr) (noRedirect, composed bool) {
	w := theBackend.windows[hwnd]
	if w == nil {
		return false, false
	}
	c := w.caption
	return w.noRedirect, c != nil && c.comp != nil && c.comp.dev != nil && c.comp.dev == theBackend.comp
}

// TestCanCompose reports whether DirectComposition is available, which
// windows without a redirection bitmap need.
func TestCanCompose() bool { return theBackend.composition() != nil }

// TestLoseComposition takes the DirectComposition device away, as a draw
// that finds it invalid does, or, with removed, signals that its GPU device
// was removed. It reports false without a device, or, with removed,
// without the event (Windows 10 before 1607).
func TestLoseComposition(removed bool) bool {
	c := theBackend.comp
	switch {
	case c == nil, removed && c.removed == 0:
		return false
	case removed:
		procSetEvent.Call(c.removed)
	default:
		theBackend.loseComposition(errors.New("taken away by a test"))
	}
	return true
}

// testFail is what TestFailWebViews armed: n creations of webviews still fail,
// of the first window that asks for its webview after the arming (owner), and
// of no other: a window that asked before, retrying on its timer, or one made
// meanwhile, creates its webview as WebView2 does. webViewSeq numbers the
// windows by their first creation. Main thread only.
var (
	testFail struct {
		n       int
		armedAt uint64
		owner   *window
	}
	webViewSeq uint64
)

// TestFailWebViews makes the next n creations of webviews of the next window
// that asks for one fail, as WebView2 reports a failure, when it would have
// created them; 0 lets them be.
func TestFailWebViews(n int) {
	testFail.n, testFail.armedAt, testFail.owner = n, webViewSeq, nil
}

// failsWebView reports whether this creation of w's webview is to fail, and
// counts it.
func (w *window) failsWebView() bool {
	if w.webViewSeq == 0 {
		webViewSeq++
		w.webViewSeq = webViewSeq
	}
	if testFail.n <= 0 || w.webViewSeq <= testFail.armedAt {
		return false
	}
	if testFail.owner == nil {
		testFail.owner = w
	}
	if testFail.owner != w {
		return false
	}
	testFail.n--
	return true
}

// TestTopNonClient returns how many pixels at the top of a window are not
// its client area, and how many a hidden title bar keeps there on this
// version of Windows: one on Windows 11, none on Windows 10.
func TestTopNonClient(hwnd uintptr) (px, want int32) {
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	var origin point
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&origin)))
	if windows11() {
		want = 1
	}
	return origin.Y - r.Top, want
}

// TestClickSurface clicks (x, y), in DIPs, in a window showing native UI,
// with the messages a mouse sends. It reports false for a window showing a
// web page.
func TestClickSurface(hwnd uintptr, x, y float64) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return false
	}
	scale := float64(w.surface.dpi()) / 96
	lp := uintptr(uint16(int16(y*scale)))<<16 | uintptr(uint16(int16(x*scale)))
	const mkLButton = 1
	procSendMessageW.Call(w.surface.hwnd, wmMouseMove, 0, lp)
	procSendMessageW.Call(w.surface.hwnd, wmLButtonDown, mkLButton, lp)
	procSendMessageW.Call(w.surface.hwnd, wmLButtonUp, 0, lp)
	return true
}

// screenPoint is a point of a window showing native UI, in DIPs from its
// top-left corner, on the screen in physical pixels.
func screenPoint(w *window, x, y float64) point {
	scale := float64(w.surface.dpi()) / 96
	pt := point{int32(x * scale), int32(y * scale)}
	procClientToScreen.Call(w.surface.hwnd, uintptr(unsafe.Pointer(&pt)))
	return pt
}

// TestOwnWindowAt reports whether the top-level window at a point of a window
// showing native UI belongs to this process: the window itself, or another
// of the app's, rather than another app's over it.
func TestOwnWindowAt(hwnd uintptr, x, y float64) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return false
	}
	return ownWindow(rootAt(screenPoint(w, x, y)))
}

// ownWindow reports whether a top-level window belongs to this process.
func ownWindow(h uintptr) bool { return h != 0 && windowProcess(h) == uint32(syscall.Getpid()) }

// TestInputClick clicks a point of a window showing native UI with the
// system's input, as a mouse does: the window it lands on activates as
// for the user. The system delivers the click later, through the thread's
// queue: TestInputClick handles the messages that come until the release
// reaches a window of the app, so that the app handled the click when it
// returns. It reports false for a window showing a web page.
//
// missed says why the click did not reach the app, and is empty when it
// did. Another app's window over the point (after bringing the window to the
// front) would take the click, so none is sent: missed describes that window
// and the one in front. A window of the app that takes no click, a parent its
// modal child disables, is clicked all the same and returns at once, as no
// release will come. If no release came within the given time, missed
// describes the window under the point and the one in front, and what may
// keep the click from coming back (diagnoseClick).
//
// The system sends a button to the window it last found under the cursor, which
// it updates as the mouse moves, and which lags behind a window that appeared
// under a cursor that stayed: the button went to the window that was there
// (the runner's terminal), or to none, and never came back. So the cursor
// moves off the point and onto it, and the click waits for that move to reach a
// window of the app. moves says how many times it moved: more than one means
// the first move did not reach it.
func TestInputClick(hwnd uintptr, x, y float64, within time.Duration) (ok bool, missed string, moves int) {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return false, "", 0
	}
	pt := screenPoint(w, x, y)
	// A window shown while another app is in front can stay behind it,
	// whose window over the point then takes the click: bring w to the
	// front first, as a user does. A window of the app there, such as a
	// parent its modal child disables, is left as it is.
	var raised string
	if under := rootAt(pt); !ownWindow(under) {
		procSetForegroundWindow.Call(w.hwnd)
		fg, _, _ := procGetForegroundWindow.Call()
		raised = fmt.Sprintf("; before the click %s was there, and after bringing the window to the front %s, with %s in front",
			describeWindow(under), describeWindow(rootAt(pt)), describeWindow(fg))
	}
	// The window under the point is read where it is reported: one that
	// appeared after the click shows.
	describe := func() string {
		fg, _, _ := procGetForegroundWindow.Call()
		return fmt.Sprintf("%s, with %s in front%s", describeWindow(rootAt(pt)), describeWindow(fg), raised)
	}
	under := rootAt(pt)
	if !ownWindow(under) {
		return true, describe(), 0
	}
	enabled, _, _ := procIsWindowEnabled.Call(under)
	deadline := time.Now().Add(within)
	// A window its modal child disables takes no mouse messages: there is
	// no move to wait for.
	if enabled != 0 {
		reached, quit := false, false
		if moves, reached, quit = moveCursorTo(pt, deadline); quit {
			return true, "", moves
		}
		if !reached {
			return true, describe() + fmt.Sprintf("; the cursor moved %d times to (%d, %d) and no move reached a window of the app", moves, pt.X, pt.Y), moves
		}
	}
	placed, _, _ := user32.NewProc("SetCursorPos").Call(uintptr(pt.X), uintptr(pt.Y))
	mouse := user32.NewProc("mouse_event")
	const leftDown, leftUp = 0x0002, 0x0004
	mouse.Call(leftDown, 0, 0, 0, 0)
	mouse.Call(leftUp, 0, 0, 0, 0)
	if enabled == 0 {
		return true, "", moves
	}
	downs := 0 // for the diagnosis of a click that did not come back
	for time.Now().Before(deadline) {
		procMsgWaitForMultipleObjectsEx.Call(0, 0, 100, qsAllInput, mwmoInputAvailable)
		var m msg
		for {
			if r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove); r == 0 {
				break
			}
			if m.Message == wmQuit {
				procPostQuitMessage.Call(m.WParam) // for the loop running outside
				return true, "", moves
			}
			if m.Message == wmLButtonDown {
				downs++
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
			if m.Message == wmLButtonUp {
				return true, "", moves
			}
		}
	}
	return true, describe() + diagnoseClick(pt, placed != 0, downs) + fmt.Sprintf("; the cursor moved %d times to reach the window", moves), moves
}

// moveCursorTo moves the cursor off pt and onto it with the system's input,
// as a mouse does, and handles messages until the move onto pt reaches a
// window of the app, which says the system found the app's window under the
// cursor. It moves again when the move did not come within a second, and
// returns how many moves it made, and whether one reached the app before the
// deadline. quit is set when the app is quitting.
func moveCursorTo(pt point, deadline time.Time) (moves int, reached, quit bool) {
	for time.Now().Before(deadline) {
		moves++
		absoluteMove(point{pt.X + 1, pt.Y})
		absoluteMove(pt)
		attempt := time.Now().Add(time.Second)
		for time.Now().Before(attempt) && time.Now().Before(deadline) {
			procMsgWaitForMultipleObjectsEx.Call(0, 0, 100, qsAllInput, mwmoInputAvailable)
			var m msg
			for {
				if r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove); r == 0 {
					break
				}
				if m.Message == wmQuit {
					procPostQuitMessage.Call(m.WParam) // for the loop running outside
					return moves, false, true
				}
				procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
				procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
				// The point of a message is where the cursor was as it was posted.
				if m.Message == wmMouseMove && abs32(m.Pt.X-pt.X) <= 1 && abs32(m.Pt.Y-pt.Y) <= 1 && ownWindow(rootOf(m.HWnd)) {
					return moves, true, false
				}
			}
		}
	}
	return moves, false, false
}

func abs32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

// absoluteMove moves the cursor to a point of the screen, in physical
// pixels, with an input event, which the system routes to the window under
// it (SetCursorPos sends none).
func absoluteMove(pt point) {
	const (
		move, absolute, virtualDesk        = 0x0001, 0x8000, 0x4000
		smXVirtual, smYVirtual, smCX, smCY = 76, 77, 78, 79
	)
	metric := func(i uintptr) int32 { r, _, _ := user32.NewProc("GetSystemMetrics").Call(i); return int32(r) }
	x0, y0, cx, cy := metric(smXVirtual), metric(smYVirtual), metric(smCX), metric(smCY)
	// 0..65535 over the virtual screen, rounded up so that the pixel is the point's.
	norm := func(v, from, size int32) uintptr {
		return uintptr((int64(v-from)*65536 + int64(size) - 1) / int64(size))
	}
	user32.NewProc("mouse_event").Call(move|absolute|virtualDesk, norm(pt.X, x0, cx), norm(pt.Y, y0, cy), 0, 0)
}

// rootOf returns the top-level window of h.
func rootOf(h uintptr) uintptr {
	const gaRoot = 2
	if root, _, _ := user32.NewProc("GetAncestor").Call(h, gaRoot); root != 0 {
		return root
	}
	return h
}

// diagnoseClick describes what may keep a click from coming back, for the
// message of one that did not: whether the button went down in the loop
// (and since no release came, it never came up), whether the system took
// the cursor to the point, whether the desktop that takes input is the
// one of this thread, and who is in front. It runs only then.
func diagnoseClick(pt point, placed bool, downs int) string {
	var at point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&at)))
	desktop := func(h uintptr) string {
		if h == 0 {
			return "none"
		}
		name := make([]uint16, 64)
		var n uint32
		const uoiName = 2
		if r, _, _ := user32.NewProc("GetUserObjectInformationW").Call(h, uoiName, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)*2), uintptr(unsafe.Pointer(&n))); r == 0 {
			return fmt.Sprintf("%#x", h)
		}
		return fmt.Sprintf("%q", syscall.UTF16ToString(name))
	}
	// The desktop of the user's input is opened for switching to it, which
	// fails where another desktop (the lock screen's) has it.
	const desktopSwitch = 0x0100
	input, _, errno := user32.NewProc("OpenInputDesktop").Call(0, 0, desktopSwitch)
	inputDesktop := fmt.Sprintf("the input desktop %s", desktop(input))
	if input == 0 {
		inputDesktop = fmt.Sprintf("the input desktop cannot be opened (%v)", errno)
	} else {
		defer user32.NewProc("CloseDesktop").Call(input)
	}
	tid, _, _ := procGetCurrentThreadId.Call()
	own, _, _ := user32.NewProc("GetThreadDesktop").Call(tid)
	fg, _, _ := procGetForegroundWindow.Call()
	var pid uint32
	fgThread, _, _ := user32.NewProc("GetWindowThreadProcessId").Call(fg, uintptr(unsafe.Pointer(&pid)))
	return fmt.Sprintf("; the button went down %d times in the loop; the cursor was set to (%d, %d): %v, and is at (%d, %d); %s, this thread's is %s; the thread in front is %d (this one %d)",
		downs, pt.X, pt.Y, placed, at.X, at.Y, inputDesktop, desktop(own), fgThread, tid)
}

// rootAt returns the top-level window at a point of the screen, in
// physical pixels, or 0.
func rootAt(pt point) uintptr {
	h, _, _ := user32.NewProc("WindowFromPoint").Call(uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32)
	if h == 0 {
		return 0
	}
	const gaRoot = 2
	if root, _, _ := user32.NewProc("GetAncestor").Call(h, gaRoot); root != 0 {
		h = root
	}
	return h
}

func windowProcess(h uintptr) (pid uint32) {
	user32.NewProc("GetWindowThreadProcessId").Call(h, uintptr(unsafe.Pointer(&pid)))
	return pid
}

// describeWindow describes a top-level window: its class, title and
// process.
func describeWindow(h uintptr) string {
	if h == 0 {
		return "no window"
	}
	class := make([]uint16, 256)
	user32.NewProc("GetClassNameW").Call(h, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	title := make([]uint16, 256)
	procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
	return fmt.Sprintf("%#x, class %q, title %q, process %d (this one %d)", h, syscall.UTF16ToString(class), syscall.UTF16ToString(title), windowProcess(h), syscall.Getpid())
}

// TestSideButton clicks a mouse's back or forward button (XBUTTON1 and
// XBUTTON2) in a window showing native UI, with the messages a mouse
// sends.
func TestSideButton(hwnd uintptr, back bool) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return false
	}
	button, held := uintptr(2), uintptr(0x40) // XBUTTON2, MK_XBUTTON2
	if back {
		button, held = 1, 0x20 // XBUTTON1, MK_XBUTTON1
	}
	scale := float64(w.surface.dpi()) / 96
	lp := uintptr(uint16(int16(20*scale)))<<16 | uintptr(uint16(int16(20*scale)))
	procSendMessageW.Call(w.surface.hwnd, wmXButtonDown, button<<16|held, lp)
	procSendMessageW.Call(w.surface.hwnd, wmXButtonUp, button<<16, lp)
	return true
}

// TestDragSurface presses the primary button at the first of points, in
// DIPs, in a window showing native UI, moves the pointer through the
// others, and releases it at the last, with the messages a mouse sends.
func TestDragSurface(hwnd uintptr, points [][2]float64) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil || len(points) < 2 {
		return false
	}
	scale := float64(w.surface.dpi()) / 96
	lp := func(p [2]float64) uintptr {
		return uintptr(uint16(int16(p[1]*scale)))<<16 | uintptr(uint16(int16(p[0]*scale)))
	}
	const mkLButton = 1
	procSendMessageW.Call(w.surface.hwnd, wmMouseMove, 0, lp(points[0]))
	procSendMessageW.Call(w.surface.hwnd, wmLButtonDown, mkLButton, lp(points[0]))
	for _, p := range points[1:] {
		procSendMessageW.Call(w.surface.hwnd, wmMouseMove, mkLButton, lp(p))
	}
	procSendMessageW.Call(w.surface.hwnd, wmLButtonUp, 0, lp(points[len(points)-1]))
	return true
}

// TestRightClickSurface clicks (x, y), in DIPs, in a window showing native
// UI with the secondary button, as TestClickSurface does with the primary.
func TestRightClickSurface(hwnd uintptr, x, y float64) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return false
	}
	scale := float64(w.surface.dpi()) / 96
	lp := uintptr(uint16(int16(y*scale)))<<16 | uintptr(uint16(int16(x*scale)))
	const mkRButton = 2
	procSendMessageW.Call(w.surface.hwnd, wmMouseMove, 0, lp)
	procSendMessageW.Call(w.surface.hwnd, wmRButtonDown, mkRButton, lp)
	procSendMessageW.Call(w.surface.hwnd, wmRButtonUp, 0, lp)
	return true
}

// TestPopups returns the labels of the items of the popup menus the main
// thread shows, "-" for separators.
func TestPopups() [][]string {
	var menus [][]string
	for _, h := range threadPopups() {
		menus = append(menus, popupLabels(h))
	}
	return menus
}

// TestChoosePopupItem chooses the item labeled label of a popup menu the
// main thread shows, with the keys a keyboard presses in it, and reports
// whether there is one.
func TestChoosePopupItem(label string) bool {
	const vkReturn, vkUp, vkDown = 0x0D, 0x26, 0x28
	for _, h := range threadPopups() {
		labels := popupLabels(h)
		target := -1
		for i, l := range labels {
			if l == label {
				target = i
			}
		}
		if target < 0 {
			continue
		}
		// The arrows move to the next item past separators, from the one
		// highlighted; Down to the first when none is.
		hmenu, _, _ := procSendMessageW.Call(h, mnGetHMenu, 0, 0)
		from := -1
		for i := range labels {
			if state, _, _ := procGetMenuState.Call(hmenu, uintptr(i), mfByPosition); state&mfHilite != 0 {
				from = i
			}
		}
		for i := from + 1; i <= target; i++ {
			if labels[i] != "-" {
				postMessage(h, wmKeyDown, vkDown, 0)
			}
		}
		for i := target; i < from; i++ {
			if labels[i] != "-" {
				postMessage(h, wmKeyDown, vkUp, 0)
			}
		}
		postMessage(h, wmKeyDown, vkReturn, 0)
		return true
	}
	return false
}

const (
	mnGetHMenu = 0x01E1
	mfHilite   = 0x0080
)

// threadPopups returns the windows of the popup menus the main thread
// shows: menus of other threads and processes are not the app's.
func threadPopups() []uintptr {
	findWindowEx := user32.NewProc("FindWindowExW")
	threadOf := user32.NewProc("GetWindowThreadProcessId")
	class := u16("#32768")
	thread := uintptr(currentThreadID())
	var popups []uintptr
	for after := uintptr(0); ; {
		h, _, _ := findWindowEx.Call(0, after, uintptr(unsafe.Pointer(class)), 0)
		if h == 0 {
			return popups
		}
		after = h
		if t, _, _ := threadOf.Call(h, 0); t != thread {
			continue
		}
		if visible, _, _ := procIsWindowVisible.Call(h); visible != 0 {
			popups = append(popups, h)
		}
	}
}

// popupLabels returns the labels of the items of the popup menu of a
// menu window, without the accelerators they show.
func popupLabels(h uintptr) []string {
	hmenu, _, _ := procSendMessageW.Call(h, mnGetHMenu, 0, 0)
	if hmenu == 0 {
		return nil
	}
	n, _, _ := procGetMenuItemCount.Call(hmenu)
	labels := []string{}
	for i := range int(int32(n)) {
		if state, _, _ := procGetMenuState.Call(hmenu, uintptr(i), mfByPosition); state&mfSeparator != 0 {
			labels = append(labels, "-")
			continue
		}
		buf := make([]uint16, 256)
		procGetMenuStringW.Call(hmenu, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), mfByPosition)
		label, _, _ := strings.Cut(syscall.UTF16ToString(buf), "\t")
		labels = append(labels, label)
	}
	return labels
}

var (
	procPrintWindow            = user32.NewProc("PrintWindow")
	procGetPixel               = gdi32.NewProc("GetPixel")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
)

// TestMenuBarColors draws a window as it shows and returns the colors, as
// COLORREFs, of its menu bar past the last item and of the line below it.
func TestMenuBarColors(hwnd uintptr) (bar, line uint32, ok bool) {
	w := theBackend.windows[hwnd]
	if w == nil {
		return 0, 0, false
	}
	r, ok := w.menuBarRect()
	if !ok {
		return 0, 0, false
	}
	var wr rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	dc, _, _ := procCreateCompatibleDC.Call(screen)
	defer procDeleteDC.Call(dc)
	bmp, _, _ := procCreateCompatibleBitmap.Call(screen, uintptr(wr.Right-wr.Left), uintptr(wr.Bottom-wr.Top))
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(dc, bmp)
	defer procSelectObject.Call(dc, old)
	const pwRenderFullContent = 2
	procPrintWindow.Call(hwnd, dc, pwRenderFullContent)
	x := uintptr(r.Right - 4)
	b, _, _ := procGetPixel.Call(dc, x, uintptr((r.Top+r.Bottom)/2))
	l, _, _ := procGetPixel.Call(dc, x, uintptr(r.Bottom))
	return uint32(b), uint32(l), true
}

// TestRecentDocuments returns the paths in the Recent category of the app's
// jump list, newest first.
func TestRecentDocuments() []string {
	var (
		clsid = guid("86bec222-30f2-47e0-9f25-60d11cd75c28") // ApplicationDocumentLists
		iid   = guid("3c594f9f-9f30-47a1-979a-c9e83d3d0a06") // IApplicationDocumentLists
	)
	const adlGetList, adltRecent = 4, 0
	lists, err := coCreate(clsid, iid)
	if err != nil {
		return nil
	}
	defer release(lists)
	var arr uintptr
	if failed(comCall(lists, adlGetList, adltRecent, 0, uintptr(unsafe.Pointer(&iidIObjectArray)), uintptr(unsafe.Pointer(&arr)))) {
		return nil
	}
	defer release(arr)
	var n uint32
	comCall(arr, oaGetCount, uintptr(unsafe.Pointer(&n)))
	var paths []string
	for i := range n {
		var item uintptr
		if failed(comCall(arr, oaGetAt, uintptr(i), uintptr(unsafe.Pointer(&iidIShellItem)), uintptr(unsafe.Pointer(&item)))) {
			continue
		}
		paths = append(paths, itemPath(item))
		release(item)
	}
	return paths
}

// TestWindowTitle returns the text of a window's title bar.
func TestWindowTitle(hwnd uintptr) string { return (&window{hwnd: hwnd}).Title() }

// TestTrayMenuImages builds the menu of a tray as PopUpMenu does and
// returns its items by text, true for those showing an icon: a bitmap, or
// the icon an item MyGo draws has.
func TestTrayMenuImages() map[string]bool {
	items := map[string]bool{}
	for _, t := range theBackend.trays {
		if t.menu == nil {
			continue
		}
		id := t.b.menus.newOwner()
		menu := t.b.buildMenu(t.menu, false, id, nil, dpiOf(0))
		n, _, _ := procGetMenuItemCount.Call(menu)
		for j := range int(int32(n)) {
			buf := make([]uint16, 256)
			procGetMenuStringW.Call(menu, uintptr(j), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), mfByPosition)
			text, _, _ := strings.Cut(syscall.UTF16ToString(buf), "\t")
			items[text], _ = menuItemHasIcon(menu, j)
		}
		procDestroyMenu.Call(menu)
		t.b.menus.drop(id)
	}
	return items
}

// TestMaximizeButton reports what Windows' hit test answers in the middle
// of the maximize button the content draws (SetMaximizeButton), and, when
// press, presses and releases it there as the mouse does: with the
// non-client messages that a window answering HTMAXBUTTON gets. It reports
// 0 when no window is over the button.
func TestMaximizeButton(hwnd uintptr, press bool) uintptr {
	w := theBackend.windows[hwnd]
	if w == nil || w.caption == nil || w.caption.snap == 0 || w.surface == nil {
		return 0
	}
	if v, _, _ := procIsWindowVisible.Call(w.caption.snap); v == 0 {
		return 0
	}
	var r rect
	procGetWindowRect.Call(w.caption.snap, uintptr(unsafe.Pointer(&r)))
	lp := uintptr(uint16(int16((r.Top+r.Bottom)/2)))<<16 | uintptr(uint16(int16((r.Left+r.Right)/2)))
	hit, _, _ := procSendMessageW.Call(w.caption.snap, wmNCHitTest, 0, lp)
	if press {
		procSendMessageW.Call(w.caption.snap, wmNCMouseMove, hit, lp)
		procSendMessageW.Call(w.caption.snap, wmNCLButtonDown, hit, lp)
		procSendMessageW.Call(w.caption.snap, wmNCLButtonUp, hit, lp)
	}
	return hit
}

// TestSurfaceKey presses and releases a key in a window showing native UI,
// with the modifiers held, as the thread's keyboard state tells them while
// it handles the messages: the window needs not be in front.
func TestSurfaceKey(hwnd uintptr, vk byte, mods ...byte) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return false
	}
	getState, setState := user32.NewProc("GetKeyboardState"), user32.NewProc("SetKeyboardState")
	var saved, state [256]byte
	getState.Call(uintptr(unsafe.Pointer(&saved)))
	state = saved
	for _, m := range mods {
		state[m] = 0x80
	}
	setState.Call(uintptr(unsafe.Pointer(&state)))
	defer setState.Call(uintptr(unsafe.Pointer(&saved)))
	down, up := uint32(wmKeyDown), uint32(wmKeyUp)
	if state[vkMenu] != 0 {
		down, up = wmSysKeyDown, wmSysKeyUp
	}
	procSendMessageW.Call(w.surface.hwnd, uintptr(down), uintptr(vk), 1)
	procSendMessageW.Call(w.surface.hwnd, uintptr(up), uintptr(vk), 1|3<<30)
	return true
}

// TestBrowserAcceleratorKeys reports whether WebView2's own keys (F5 and
// Ctrl+R reloading, Ctrl+P printing, ...) work in a window's page; ok is
// false without a web view or a runtime with ICoreWebView2Settings3.
func TestBrowserAcceleratorKeys(hwnd uintptr) (on, ok bool) {
	w := theBackend.windows[hwnd]
	if w == nil || w.settings == 0 {
		return false, false
	}
	s3 := queryInterface(w.settings, &iidICoreWebView2Settings3)
	if s3 == 0 {
		return false, false
	}
	defer release(s3)
	var v int32
	comCall(s3, set3GetAreBrowserAcceleratorKeysEnabled, uintptr(unsafe.Pointer(&v)))
	return v != 0, true
}

// TestHoldClipboardData takes a reference to the data object the app put on
// the clipboard, as an app that reads the clipboard may keep one, and returns
// the function that releases it.
func TestHoldClipboardData() func() {
	s := winClipboard
	if s == nil {
		return func() {}
	}
	object := s.object
	addRef(object)
	return func() { release(object) }
}

// TestPageGestures reports whether a swipe navigates and a pinch zooms
// the page of a window, or with webView 0 or more, of its web view of that
// index, as its WebView2 settings say. ok is false without that page's web
// view or a runtime with ICoreWebView2Settings6.
func TestPageGestures(hwnd uintptr, webView int) (swipe, pinch, ok bool) {
	w := theBackend.windows[hwnd]
	if w != nil && webView >= 0 {
		if webView >= len(w.webViews) {
			return false, false, false
		}
		w = w.webViews[webView]
	}
	if w == nil || w.settings == 0 {
		return false, false, false
	}
	s6 := queryInterface(w.settings, &iidICoreWebView2Settings6)
	if s6 == 0 {
		return false, false, false
	}
	defer release(s6)
	var sw, pz int32
	comCall(s6, set6GetIsSwipeNavigationEnabled, uintptr(unsafe.Pointer(&sw)))
	comCall(s6, set5GetIsPinchZoomEnabled, uintptr(unsafe.Pointer(&pz)))
	return sw != 0, pz != 0, true
}

// TestPinchSurface pinches a touchpad at (x, y), in DIPs, in a window
// showing native UI, by scale, with the message Windows sends for it: the
// wheel with Ctrl held, by less than a notch at a time. It reports the
// scale the message makes, which rounds scale to a 360th of a doubling.
func TestPinchSurface(hwnd uintptr, x, y, scale float64) (float64, bool) {
	w := theBackend.windows[hwnd]
	if w == nil || w.surface == nil {
		return 0, false
	}
	delta := int(math.Round(360 * math.Log2(scale)))
	if delta%120 == 0 {
		delta++ // a notch is a mouse wheel's
	}
	dpi := float64(w.surface.dpi()) / 96
	pt := point{int32(x * dpi), int32(y * dpi)}
	procClientToScreen.Call(w.surface.hwnd, uintptr(unsafe.Pointer(&pt)))
	const mkControl = 0x0008
	wp := uintptr(uint16(int16(delta)))<<16 | mkControl
	lp := uintptr(uint16(int16(pt.Y)))<<16 | uintptr(uint16(int16(pt.X)))
	procSendMessageW.Call(w.surface.hwnd, wmMouseWheel, wp, lp)
	return math.Exp(float64(delta) / 120 * math.Ln2 / 3), true
}

// TestInMenuMode reports whether the foreground thread is in menu mode (a menu
// bar item highlighted or a menu open), as after an Alt key the window did not
// take.
func TestInMenuMode() bool {
	type guiThreadInfo struct {
		size, flags                                        uint32
		active, focus, capture, menuOwner, moveSize, caret uintptr
		caretLeft, caretTop, caretRight, caretBottom       int32
	}
	info := guiThreadInfo{size: uint32(unsafe.Sizeof(guiThreadInfo{}))}
	r, _, _ := user32.NewProc("GetGUIThreadInfo").Call(0, uintptr(unsafe.Pointer(&info)))
	const guiInMenuMode, guiPopupMenuMode = 0x1, 0x10
	return r != 0 && info.flags&(guiInMenuMode|guiPopupMenuMode) != 0
}

// TestCloak cloaks a window, or shows it again, as Windows does a window of
// another virtual desktop: it stays visible to IsWindowVisible and shows
// nowhere.
func TestCloak(hwnd uintptr, on bool) error {
	v := uint32(0)
	if on {
		v = 1
	}
	if hr, _, _ := procDwmSetWindowAttribute.Call(hwnd, dwmwaCloak, uintptr(unsafe.Pointer(&v)), 4); hr != 0 {
		return fmt.Errorf("mygo: DwmSetWindowAttribute(DWMWA_CLOAK) failed: %#x", uint32(hr))
	}
	return nil
}

// TestMaximizedRects returns, in pixels, a window's rectangle, the part of
// it DWM shows (without its invisible borders), its client area on the
// screen, and the monitor and work area it is on, and whether it is
// zoomed.
func TestMaximizedRects(hwnd uintptr) (window, shown, client, monitor, work [4]int32, zoomed bool) {
	var r, s, c rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	procDwmGetWindowAttribute.Call(hwnd, dwmwaExtendedFrameBounds, uintptr(unsafe.Pointer(&s)), unsafe.Sizeof(s))
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&c)))
	var origin point
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&origin)))
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNearest)
	mi := monitorInfo(mon)
	z, _, _ := procIsZoomed.Call(hwnd)
	four := func(r rect) [4]int32 { return [4]int32{r.Left, r.Top, r.Right, r.Bottom} }
	return four(r), four(s), [4]int32{origin.X, origin.Y, origin.X + c.Right, origin.Y + c.Bottom}, four(mi.Monitor), four(mi.Work), z != 0
}

// TestHitTest reports what Windows' hit test answers at a point of a
// window, in pixels from the top-left of its client area (negative from
// the right or the bottom): the answer of the topmost visible window
// there, the top edge above the webview or the surface, or the window
// itself.
func TestHitTest(hwnd uintptr, x, y int32) uintptr {
	w := theBackend.windows[hwnd]
	if w == nil {
		return 0
	}
	var c rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&c)))
	if x < 0 {
		x += c.Right
	}
	if y < 0 {
		y += c.Bottom
	}
	const cwpSkipInvisible = 1
	child, _, _ := user32.NewProc("ChildWindowFromPointEx").Call(hwnd, uintptr(uint32(x))|uintptr(uint32(y))<<32, cwpSkipInvisible)
	if child == 0 {
		child = hwnd
	}
	pt := point{x, y}
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	lp := uintptr(uint16(int16(pt.Y)))<<16 | uintptr(uint16(int16(pt.X)))
	hit, _, _ := procSendMessageW.Call(child, wmNCHitTest, 0, lp)
	return hit
}

// TestMenuItemIcon reports the size, in DIPs, of the icon an item of a
// window's menu bar shows, 0 by 0 for none, and whether it is checked.
func TestMenuItemIcon(hwnd uintptr, path ...string) (width, height int, checked bool, err error) {
	w := theBackend.windows[hwnd]
	if w == nil || w.menu == nil {
		return 0, 0, false, fmt.Errorf("mygo: the window has no menu bar")
	}
	items := w.menu.Items
	var uid int
	for i, label := range path {
		found := false
		for _, it := range items {
			if it.Label != label {
				continue
			}
			found = true
			if i == len(path)-1 {
				uid = it.ID
			} else if it.Submenu != nil {
				items = it.Submenu.Items
			}
			break
		}
		if !found {
			return 0, 0, false, fmt.Errorf("mygo: no menu item %q", label)
		}
	}
	for cmd, e := range theBackend.menus.entries {
		if e.uid != uid || e.owner != w.owner {
			continue
		}
		mi := menuItemInfo{Mask: miimBitmap | miimState}
		mi.Size = uint32(unsafe.Sizeof(mi))
		user32.NewProc("GetMenuItemInfoW").Call(e.menu, uintptr(cmd), 0, uintptr(unsafe.Pointer(&mi)))
		checked = mi.State&mfsChecked != 0
		dpi := dpiOf(hwnd)
		switch {
		case e.look != nil && e.look.icon != nil && e.look.icons && drawMenuIcon != nil:
			if img := iconAt(e.look.icon, toPx(16, dpi), 0); img != nil {
				width, height = toDIP(int32(img.W), dpi), toDIP(int32(img.H), dpi)
			}
		case e.look == nil && mi.BmpItem != 0:
			var ds dibSection
			procGetObjectW.Call(mi.BmpItem, unsafe.Sizeof(ds), uintptr(unsafe.Pointer(&ds)))
			width, height = toDIP(ds.bmWidth, dpi), toDIP(ds.bmHeight, dpi)
		}
		return width, height, checked, nil
	}
	return 0, 0, false, fmt.Errorf("mygo: menu item %q is not a command", path[len(path)-1])
}

// TestGDIObjects returns how many GDI objects the process holds, which a
// menu's bitmaps count in.
func TestGDIObjects() int {
	const grGDIObjects = 0
	process, _ := syscall.GetCurrentProcess()
	n, _, _ := user32.NewProc("GetGuiResources").Call(uintptr(process), grGDIObjects)
	return int(n)
}

// TestRightClickPage right-clicks (x, y), in CSS pixels, in the page of a
// window's web view, with the mouse events of the DevTools protocol, as a
// mouse does. The page's context menu opens later. It reports false for a
// window without a web view.
func TestRightClickPage(hwnd uintptr, x, y float64) bool {
	w := theBackend.windows[hwnd]
	if w == nil || w.webview == 0 {
		return false
	}
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		w.devtools("Input.dispatchMouseEvent", fmt.Sprintf(`{"type":%q,"x":%g,"y":%g,"button":"right","buttons":2,"clickCount":1}`, typ, x, y), nil)
	}
	return true
}

// TestPageMenu returns the labels of the items of the popup menu the main
// thread shows, without their mnemonics' ampersands, "-" for separators,
// and false when it shows none: a page's context menu MyGo draws. The
// menu WebView2 draws itself is not MyGo's to read.
func TestPageMenu() ([]string, bool) {
	popups := threadPopups()
	if len(popups) == 0 {
		return nil, false
	}
	labels := popupLabels(popups[0])
	for i, l := range labels {
		labels[i] = withoutAmpersands(l)
	}
	return labels, true
}

// TestChoosePageMenuItem chooses the item labeled label, without its
// mnemonic's ampersand, of that menu, and reports whether it has one.
func TestChoosePageMenuItem(label string) bool {
	popups := threadPopups()
	if len(popups) == 0 {
		return false
	}
	for _, l := range popupLabels(popups[0]) {
		if withoutAmpersands(l) == label {
			return TestChoosePopupItem(l)
		}
	}
	return false
}

// TestBackgroundThrottlingOff reports whether the page of a window, or of
// its first web view, keeps running while it can't be seen: WebView2's
// browser runs with unthrottledArguments, the one setting, which all
// pages share.
func TestBackgroundThrottlingOff(hwnd uintptr, webView bool) (off bool, settings int) {
	w := theBackend.windows[hwnd]
	if w != nil && webView {
		if len(w.webViews) == 0 {
			return false, 0
		}
		w = w.webViews[0]
	}
	if w == nil || w.controller == 0 {
		return false, 0
	}
	return theBackend.envUnthrottled, 1
}

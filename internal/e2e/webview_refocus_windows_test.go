//go:build windows

package e2e

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// refocusPage is a page whose editable element counts the copies made in it.
const refocusPage = `<body style="margin:0;height:100vh">
<div id=e contenteditable style="height:100px">hello</div><script>
window.copies = 0;
document.addEventListener("copy", () => window.copies++);
window.ready = true;
</script></body>`

// pageHasKeyboard is true in a page whose editable element has the keyboard.
const pageHasKeyboard = "document.hasFocus() && document.activeElement && document.activeElement.id === 'e'"

// webViewHasKeyboard is in foregroundState while a web view of the window
// has the keyboard.
const webViewHasKeyboard = "a web view in it has the keyboard: true"

// refocusWindow makes a window of native UI with a text input above a web
// view showing refocusPage, and another window to take the front from it.
func refocusWindow(t *testing.T) (w, other *mygo.Window, v *mygo.WebView, input func() bool) {
	t.Helper()
	var view *mygo.WebView
	var text string
	var focused atomic.Bool
	w = newWindow(t, mygo.WindowOptions{Title: "Web view refocus", Width: 400, Height: 300, X: 40, Y: 40, Content: ui.View(func(c *ui.Context) {
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			focused.Store(ui.TextInput(c, &text).Label("Native").Focused())
			if view != nil {
				ui.WebView(c, view).Grow(1)
			}
		})
	})})
	other = newWindow(t, mygo.WindowOptions{Title: "Other", Width: 300, Height: 200, X: 480, Y: 40, Content: ui.View(func(c *ui.Context) {
		ui.Text(c, "other")
	})})
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(refocusPage, "")
	waitForPage(t, v.Page(), "window.ready")
	input = func() bool {
		w.Invalidate()
		return focused.Load()
	}
	return w, other, v, input
}

// toFront focuses a window and waits until it is in front, saying who is
// when it never comes.
func toFront(t *testing.T, w *mygo.Window) {
	t.Helper()
	w.Focus()
	if !wait.Poll(func() bool {
		if w.IsFocused() {
			return true
		}
		keepFront()
		return false
	}) {
		t.Fatalf("%s in front never happened (waited %v); %s", w.Title(), wait.Guard, foregroundDiag(w))
	}
}

// TestContentWindowWebViewRefocus takes the front from a window of native
// UI whose web view has the keyboard, and gives it back, as Alt+Tab does:
// the page has the keyboard again, and the Edit menu's Copy goes to it,
// not to the native UI.
func TestContentWindowWebViewRefocus(t *testing.T) {
	w, other, v, _ := refocusWindow(t)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleEditMenu}}))
	defer mygo.App.SetMenu(nil)
	clipboard := mygo.Clipboard.ReadText()
	defer mygo.Clipboard.WriteText(clipboard)

	toFront(t, w)
	v.Focus()
	if _, err := v.Page().Eval("document.getElementById('e').focus()"); err != nil {
		t.Fatal(err)
	}
	// The page and the window both know: WebView2 tells the window after
	// the page has the keyboard.
	if !wait.Poll(func() bool {
		ok, _ := mygo.EvalAs[bool](v.Page(), pageHasKeyboard)
		return ok && strings.Contains(foregroundState(w), webViewHasKeyboard)
	}) {
		t.Fatalf("the page to have the keyboard never happened (waited %v); %s", wait.Guard, foregroundState(w))
	}

	toFront(t, other)
	toFront(t, w)
	if !wait.Poll(func() bool {
		ok, _ := mygo.EvalAs[bool](v.Page(), pageHasKeyboard)
		return ok && strings.Contains(foregroundState(w), webViewHasKeyboard)
	}) {
		t.Fatalf("the page to have the keyboard again once its window came back to the front never happened (waited %v); %s", wait.Guard, foregroundState(w))
	}

	menu := func(item string) {
		t.Helper()
		if err := activateMenu(w, "Edit", item); err != nil {
			t.Skipf("menu automation: %v", err)
		}
	}
	menu("Select All")
	waitForPage(t, v.Page(), "getSelection().toString() === 'hello'")
	sent := foregroundState(w)
	menu("Copy")
	if !wait.Poll(func() bool { n, _ := mygo.EvalAs[float64](v.Page(), "window.copies"); return n > 0 }) {
		t.Fatalf("Copy to copy in the page never happened (waited %v); as Copy was sent, %s", wait.Guard, sent)
	}
}

// TestContentWindowNativeRefocus takes the front from a window of native
// UI whose text input has the keyboard, and gives it back: the native UI
// keeps the keyboard, which its web view does not take.
func TestContentWindowNativeRefocus(t *testing.T) {
	w, other, v, input := refocusWindow(t)
	toFront(t, w)
	if !click(w, 100, 20) {
		t.Skip("click automation not available on this platform")
	}
	if !wait.Poll(input) {
		t.Fatalf("the text input to have the keyboard never happened (waited %v); %s", wait.Guard, foregroundState(w))
	}

	toFront(t, other)
	toFront(t, w)
	if !wait.Poll(input) {
		t.Fatalf("the text input to keep the keyboard never happened (waited %v); %s", wait.Guard, foregroundState(w))
	}
	// What would give the page the keyboard has had time to: a script runs
	// after the focus change that came before it.
	if _, err := v.Page().Eval("1"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := mygo.EvalAs[bool](v.Page(), "document.hasFocus()"); ok {
		t.Errorf("the page took the keyboard as its window came back to the front; %s", foregroundState(w))
	}
	if s := foregroundState(w); strings.Contains(s, webViewHasKeyboard) {
		t.Errorf("the web view took the keyboard as its window came back to the front; %s", s)
	}
}

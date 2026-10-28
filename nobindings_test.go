//go:build !js

package mygo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/fake"
)

// noBindingsAndNormalViews returns a window with a web view without
// bindings (WebViewOptions.NoBindings, as the web browser widget makes) and
// a normal one, both with their DOM ready.
func noBindingsAndNormalViews(t *testing.T) (w *Window, bare, normal *WebView, fbare, fnormal *fake.WebView) {
	t.Helper()
	show := true
	w, bare, s, fbare := webViewWindowOpts(t, &show, WebViewOptions{NoBindings: true})
	normal, err := w.NewWebView(WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	views := s.WebViews()
	if len(views) != 2 {
		t.Fatalf("%d web views", len(views))
	}
	fnormal = views[1]
	onMain(func() { fnormal.H.NavigationCommitted("about:blank") })
	page(fnormal.Window, `{"t":"dom-ready"}`)
	return w, bare, normal, fbare, fnormal
}

// A page without bindings hears only the events sent to it: it shows any
// site, which must not learn what the app tells its own pages.
func TestNoBindingsEvents(t *testing.T) {
	ev := newEventForTest[string](t, "nobindings-test")
	w, bare, _, fbare, fnormal := noBindingsAndNormalViews(t)
	if err := ev.Broadcast("to all"); err != nil {
		t.Fatal(err)
	}
	if err := ev.Emit(w, "to the window"); err != nil {
		t.Fatal(err)
	}
	// Events reach a page in the order they were sent.
	if err := ev.EmitPage(bare.Page(), "to the page"); err != nil {
		t.Fatal(err)
	}
	received(t, fbare.Window, func(m map[string]any) bool { return m["t"] == "event" && m["p"] == "to the page" })
	for _, s := range fbare.Scripts() {
		if strings.Contains(s, "to all") || strings.Contains(s, "to the window") {
			t.Errorf("a page without bindings heard %s", s)
		}
	}
	for _, want := range []string{"to all", "to the window"} {
		received(t, fnormal.Window, func(m map[string]any) bool { return m["t"] == "event" && m["p"] == want })
	}
}

// A page without bindings calls no Go method, and posts no message larger
// than the bridge's: those are dropped before they are decoded.
func TestNoBindingsCalls(t *testing.T) {
	bindForTest(t, "Pages", pageService{})
	_, _, _, fbare, _ := noBindingsAndNormalViews(t)
	m := call(t, fbare.Window, 1, "Pages.Where")
	if m["ok"] != false {
		t.Errorf("a call from a page without bindings: %v", m)
	}
	big := strings.Repeat("a", maxUntrustedMessage)
	page(fbare.Window, fmt.Sprintf(`{"t":"call","id":10,"k":"tok","m":"Pages.Where","a":[%q]}`, big))
	call(t, fbare.Window, 11, "Pages.Where")
	for _, s := range fbare.Scripts() {
		if strings.Contains(s, `"id":10`) {
			t.Errorf("a message above the limit was decoded: %.200s", s)
		}
	}
}

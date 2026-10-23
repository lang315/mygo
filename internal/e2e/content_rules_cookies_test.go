//go:build !js

package e2e

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// TestContentRulesBlockCookies applies a block-cookies rule to /track/ and
// checks that a matching request goes without the cookies the jar has, and
// that the cookie its response sets is not kept, while the same requests
// to /open/ (no rule) send and keep cookies.
func TestContentRulesBlockCookies(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("WebView2 adds cookies after WebResourceRequested and shows no response, so block-cookies keeps none off (docs/windows.md)")
	}
	prefix := fmt.Sprintf("mygo%d", time.Now().UnixNano())
	open, track := prefix+"open", prefix+"track"
	var mu sync.Mutex
	sent := map[string]string{} // path -> the Cookie header of the last request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		sent[r.URL.Path] = r.Header.Get("Cookie")
		mu.Unlock()
		switch r.URL.Path {
		case "/open/set":
			http.SetCookie(w, &http.Cookie{Name: open, Value: "1", Path: "/"})
		case "/track/set":
			http.SetCookie(w, &http.Cookie{Name: track, Value: "1", Path: "/"})
		case "/open/echo", "/track/echo":
		default:
			fmt.Fprint(w, "<!doctype html><title>cookies</title><body>cookies</body>")
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	rules, err := mygo.CompileContentRules("e2e-cookies", `[{"trigger":{"url-filter":"/track/"},"action":{"type":"block-cookies"}}]`)
	if errors.Is(err, errors.ErrUnsupported) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Block cookies", Width: 300, Height: 200, Page: mygo.PageOptions{ContentRules: []*mygo.ContentRules{rules}}})
	if err := w.Page().LoadURL(server.URL + "/"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "document.title === 'cookies'")
	get := func(path string) {
		t.Helper()
		if _, err := mygo.EvalAs[string](w.Page(), `fetch("`+path+`").then(r => r.text())`); err != nil {
			t.Fatalf("fetch %s: %v", path, err)
		}
	}
	cookie := func(path string) string {
		mu.Lock()
		defer mu.Unlock()
		return sent[path]
	}
	stored := func(name string) bool {
		list, err := mygo.Cookies.List(mygo.CookieFilter{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return len(list) > 0
	}

	// Control: with no rule a cookie is kept and sent back.
	get("/open/set")
	get("/open/echo")
	if got := cookie("/open/echo"); got != open+"=1" {
		t.Fatalf("control: /open/echo got Cookie %q, want %q", got, open+"=1")
	}

	// The request: the jar holds the open cookie, and /track/echo is blocked
	// from sending cookies.
	get("/track/echo")
	if got := cookie("/track/echo"); got != "" {
		t.Errorf("request: /track/echo got Cookie %q, want none", got)
	}

	// The response: /track/set sets a cookie that is not to be kept.
	get("/track/set")
	if stored(track) {
		t.Errorf("response: the Set-Cookie of /track/set was kept in the cookie store")
	}
	get("/open/echo")
	if got := cookie("/open/echo"); got != open+"=1" {
		t.Errorf("response: /open/echo then got Cookie %q, want only %q", got, open+"=1")
	}
}

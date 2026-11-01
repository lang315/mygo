//go:build !js

package e2e

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo"
)

// The bridge of each document says a secret in a hello, which the window takes
// after the main frame commits. On every backend the commit has to reach Go
// before that hello (the Windows order is the one that needs proof), or a
// document's calls would never be taken. So after each way a main frame can
// load, the page's own call to Go must still run.
func TestBridgeCallsAfterEveryKindOfLoad(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	touches := func(step string) {
		t.Helper()
		before := probe.n.Load()
		if _, err := p.Eval("mygo.call('Probe.Touch')"); err != nil || probe.n.Load() != before+1 {
			t.Fatalf("%s: the page's call: %v, count %d, want %d", step, err, probe.n.Load(), before+1)
		}
	}

	p.LoadHTML(`<p>first</p>`, "app://localhost/")
	waitFor(t, w, "window.mygo")
	touches("LoadHTML")

	p.LoadHTML(`<p>second</p>`, "app://localhost/")
	waitFor(t, w, "document.body.textContent === 'second'")
	touches("another LoadHTML")

	if err := p.LoadURL("app://localhost/"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "window.mygo")
	touches("LoadURL")

	// A reload is a new document, with a secret of its own.
	if _, err := p.Eval("location.reload()"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "window.mygo && document.readyState === 'complete'")
	touches("reload")

	// A navigation inside the document commits nothing: the secret stays.
	if _, err := p.Eval("history.pushState({}, '', '#pushed'); location.hash = 'hash'"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "location.hash === '#hash'")
	touches("pushState and a hash")
}

// An iframe can post to the handler on every backend, as a frame of another
// origin would. Its hello comes after the page's, which the window took, so it
// is ignored; a call that starts with the secret it made does not run. The page's
// own call does. (Whether WebView2 hands Go the messages of an iframe at all is
// what the log of this test says.)
func TestIframeCannotSayHello(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	secret := "0123456789abcdef0123456789abcdef"
	forged := secret + `{"t":"call","id":1,"k":"x","m":"Probe.Touch","a":[]}`
	frame := `<script>
const post = (m) => {
  const h = window.webkit && webkit.messageHandlers && webkit.messageHandlers.mygo;
  if (h) { h.postMessage(m); return "webkit"; }
  if (window.chrome && chrome.webview) { chrome.webview.postMessage(m); return "webview2"; }
  return "";
};
const via = post(JSON.stringify({ t: "hello", k: "` + secret + `" }));
post(` + "`" + forged + "`" + `);
parent.postMessage(via, "*");
</script>`
	src, _ := json.Marshal(frame)
	w.Page().LoadHTML(`<p>main</p><script>
addEventListener("message", (e) => { window.frameResult = e.data });
const f = document.createElement("iframe");
f.src = "data:text/html," + encodeURIComponent(`+string(src)+`);
document.body.append(f);
</script>`, "app://localhost/")
	waitFor(t, w, "window.frameResult !== undefined")
	via, _ := mygo.EvalAs[string](w.Page(), "window.frameResult")
	if via == "" {
		t.Skip("the iframe had no handler to post to")
	}
	t.Logf("the iframe posted through %s", via)
	// A call after the iframe's, so the forged ones were handled before it.
	before := probe.n.Load()
	if _, err := w.Page().Eval("mygo.call('Probe.Touch')"); err != nil || probe.n.Load() != before+1 {
		t.Fatalf("the page's own call: %v, count %d, want %d", err, probe.n.Load(), before+1)
	}
}

// A page may replace what the bridge posts with after it started, and an
// iframe may post to the handler as it likes: neither gets the secret, so
// neither can run a call. The page's own call, which goes through what the
// bridge captured, still runs.
func TestPageCannotTakeTheSecret(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	p.LoadHTML(`<p>main</p>`, "app://localhost/")
	waitFor(t, w, "window.mygo")
	if _, err := p.Eval(`
window.stolen = [];
const spy = (m) => { window.stolen.push(String(m)); };
const h = window.webkit && webkit.messageHandlers && webkit.messageHandlers.mygo;
if (h) h.postMessage = spy;
if (window.chrome && chrome.webview) {
  chrome.webview.postMessage = spy;
  chrome.webview.postMessageWithAdditionalObjects = spy;
}
const stringify = JSON.stringify;
JSON.stringify = (v) => { window.stolen.push("stringify"); return stringify(v); };
`); err != nil {
		t.Fatal(err)
	}
	before := probe.n.Load()
	if _, err := p.Eval("mygo.call('Probe.Touch')"); err != nil || probe.n.Load() != before+1 {
		t.Fatalf("the page's call after it replaced postMessage: %v, count %d, want %d", err, probe.n.Load(), before+1)
	}
	stolen, err := mygo.EvalAs[[]string](p, "window.stolen")
	if err != nil {
		t.Fatal(err)
	}
	// The page's own JSON.stringify is called by the Eval that reads this; what
	// the bridge posts is not, and a message to Go starts with 32 hex digits.
	for _, s := range stolen {
		if s != "stringify" {
			t.Errorf("the page saw a message posted to Go: %q", s)
		}
	}
}

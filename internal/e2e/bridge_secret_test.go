//go:build !js

package e2e

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
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
		// A bridge Go does not take messages from leaves the call pending: the
		// context ends the wait.
		if _, err := p.EvalContext(wait.Context(t), "mygo.call('Probe.Touch')"); err != nil || probe.n.Load() != before+1 {
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

	// Back and Forward: WebKit restores a document from its cache with its
	// bridge, which does not start again, while the main frame commits and Go
	// waits for a hello. The bridge says its secret once more on pageshow, and
	// the call of the restored page must run. Back to the page, then Forward.
	if err := p.LoadURL("app://localhost/capture"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "location.pathname === '/capture' && window.mygo")
	touches("a second page")
	finished := make(chan struct{}, 1)
	off := p.OnDidFinishLoad(func() {
		select {
		case finished <- struct{}{}:
		default:
		}
	})
	defer off()
	p.GoBack()
	waitFor(t, w, "location.pathname === '/' && window.mygo")
	select {
	case <-finished:
	case <-wait.Context(t).Done():
		t.Fatal("Back: the page did not finish loading")
	}
	touches("Back")
	p.GoForward()
	waitFor(t, w, "location.pathname === '/capture' && window.mygo")
	touches("Forward")
}

// The message handler is reachable from every frame, and the secret is all that
// stands between a frame and Go on a backend that cannot tell which frame posted.
// macOS can (it rejects subframes natively, so its frames may still see the
// handler); WebKitGTK and WebView2 cannot, so there the frames that are not the
// page's own must not have a handler to post to at all (bridge.HideHandlerScript):
// a frame of another origin, a srcdoc frame, an about:blank frame its script
// made, and the frames inside those.
func TestFramesGetNoHandler(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	const reach = `function reach(w) {
  try {
    const h = w.webkit && w.webkit.messageHandlers && w.webkit.messageHandlers.mygo;
    if (h && typeof h.postMessage === "function") return "webkit";
    if (w.chrome && w.chrome.webview && typeof w.chrome.webview.postMessage === "function") return "webview2";
  } catch (e) { return "error: " + e; }
  return "";
}
`
	js := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	// A srcdoc frame inside a frame of another origin reports to it, which
	// relays to the page.
	nested := `<script>` + reach + `parent.postMessage({ dataSrcdoc: reach(window) }, "*");</script>`
	other := `<script>` + reach + `
const out = { data: reach(window) };
addEventListener("message", (e) => parent.postMessage(e.data, "*"));
const b = document.createElement("iframe");
document.body.append(b);
out.dataBlank = reach(b.contentWindow);
const s = document.createElement("iframe");
s.srcdoc = ` + js(nested) + `;
document.body.append(s);
parent.postMessage(out, "*");
</script>`
	w.Page().LoadHTML(`<p>main</p><script>`+reach+`
window.results = {};
addEventListener("message", (e) => Object.assign(window.results, e.data));
const d = document.createElement("iframe");
d.src = "data:text/html," + encodeURIComponent(`+js(other)+`);
document.body.append(d);
const s = document.createElement("iframe");
s.srcdoc = "<p>srcdoc</p>";
s.onload = () => { window.results.srcdoc = reach(s.contentWindow); };
document.body.append(s);
const b = document.createElement("iframe");
document.body.append(b);
window.results.blank = reach(b.contentWindow);
</script>`, "app://localhost/")
	waitFor(t, w, "['data', 'dataBlank', 'dataSrcdoc', 'srcdoc', 'blank'].every((k) => k in window.results)")
	res, err := mygo.EvalAs[map[string]string](w.Page(), "window.results")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("the frames reach the handler through: %v", res)
	if runtime.GOOS == "darwin" {
		t.Log("macOS rejects the messages of subframes natively (frameInfo.isMainFrame); its frames may see the handler")
	} else {
		for frame, via := range res {
			if via != "" {
				t.Errorf("the %s frame has a message handler (%s): it could post to Go", frame, via)
			}
		}
	}
	// A call of the page's own still runs.
	before := probe.n.Load()
	if _, err := w.Page().EvalContext(wait.Context(t), "mygo.call('Probe.Touch')"); err != nil || probe.n.Load() != before+1 {
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

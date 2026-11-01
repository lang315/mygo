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
// made, a frame whose source has not committed, and the frames inside those.
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
const q = document.createElement("iframe");
q.src = "app://localhost/204";
document.body.append(q);
window.results.noCommit = reach(q.contentWindow);
</script>`, "app://localhost/")
	// The frames that report to the page by message may not arrive (the relay of a
	// frame of another origin): the ones that matter are read at once.
	waitFor(t, w, "['data', 'dataBlank', 'blank', 'noCommit'].every((k) => k in window.results)")
	res, err := mygo.EvalAs[map[string]string](w.Page(), "window.results")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("the frames reach the handler through: %v", res)
	if runtime.GOOS == "darwin" {
		t.Log("macOS rejects the messages of subframes natively (frameInfo.isMainFrame); its frames may see the handler")
	} else {
		for frame, via := range res {
			switch {
			case via == "":
			case frame == "noCommit" && runtime.GOOS == "linux":
				// A frame whose source has not committed is an initial empty
				// document, which WebKit gives no user script. WebKitGTK is held by
				// not restoring pages from the cache, not by this.
				t.Logf("the frame with a source that has not committed has a handler (%s): known on WebKitGTK", via)
			default:
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

// On Linux Back and Forward load the page again rather than restore it from the
// back/forward cache (enable-page-cache is off): a restored document keeps its
// bridge, and WebKitGTK runs the pageshow of a restored page's subframes before
// the main frame's, so a frame that can post could say its hello first. A new
// load runs the bridge at document start, before the page has a frame.
func TestBackIsANewLoadOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("macOS keeps the cache (its frames are filtered natively); WebView2 keeps its own")
	}
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	if err := p.LoadURL("app://localhost/"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "window.mygo")
	if _, err := p.Eval("window.restoredMarker = 1; window.restoredPersisted = null; addEventListener('pageshow', (e) => { window.restoredPersisted = e.persisted })"); err != nil {
		t.Fatal(err)
	}
	if err := p.LoadURL("app://localhost/capture"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "location.pathname === '/capture' && window.mygo")
	p.GoBack()
	waitFor(t, w, "location.pathname === '/' && window.mygo")
	if got, err := mygo.EvalAs[bool](p, "window.restoredMarker === undefined"); err != nil || !got {
		t.Errorf("Back restored the page from a cache: marker kept (%v, %v)", got, err)
	}
}

// A page that is not trusted can post from its pagehide, as it is left for a page
// of another site (the engine may swap processes, so its messages may reach Go
// after the commit of the next page): neither a hello nor a call of it may run
// with the trust of the page that comes next. Posted with the handler directly, as
// a page can, with a secret it made up.
func TestAPageCannotPostForTheNextOneFromPagehide(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	secret := "0123456789abcdef0123456789abcdef"
	hello := `{"t":"hello","k":"` + secret + `"}`
	call := secret + `{"t":"call","id":1,"k":"x","m":"Probe.Touch","a":[]}`
	hj, _ := json.Marshal(hello)
	cj, _ := json.Marshal(call)
	// A page of an origin the app does not trust, with MyGo's bridge (it is in
	// every page), leaving for the app's own page.
	p.LoadHTML(`<p>elsewhere</p><script>
addEventListener("pagehide", () => {
  const post = (m) => {
    const h = window.webkit && webkit.messageHandlers && webkit.messageHandlers.mygo;
    if (h) { h.postMessage(m); return; }
    if (window.chrome && chrome.webview) chrome.webview.postMessage(m);
  };
  post(`+string(hj)+`);
  post(`+string(cj)+`);
});
</script>`, "https://untrusted.example/")
	waitFor(t, w, "window.mygo")
	before := probe.n.Load()
	if err := p.LoadURL("app://localhost/"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "location.origin === 'app://localhost' && window.mygo")
	// The next page's own call goes through the same channel after whatever the
	// old page posted that has arrived.
	if _, err := p.EvalContext(wait.Context(t), "mygo.call('Probe.Touch')"); err != nil {
		t.Fatal(err)
	}
	if n := probe.n.Load(); n != before+1 {
		t.Fatalf("Probe.Touch ran %d times, want 1 (the new page's own): a call posted from pagehide ran", n-before)
	}
}

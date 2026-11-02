//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	win "github.com/egoist/mygo/internal/windows"
)

func messageStats() (heard, other int, others []string, commits int) {
	mygo.RunOnMain(func() { heard, other, others, commits = win.TestMessageStats() })
	return
}

// A document that is left may post messages from the events of its leaving,
// and the engine may deliver them after the next document committed. They
// take no effect there: after each round the next document's own call runs
// once, and what the left documents posted never runs a method of Go.
// None reaches Go from a document other than the one shown (measured: 0 of
// several hundred); the test fails if the engine starts to deliver them.
func TestMessagesFromAPreviousDocumentTakeNoEffect(t *testing.T) {
	const rounds = 25
	resetProbe(t)
	mygo.RunOnMain(win.TestWatchMessages)
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	secret := "0123456789abcdef0123456789abcdef"
	hello := fmt.Sprintf("%q", `{"t":"hello","k":"`+secret+`"}`)
	call := fmt.Sprintf("%q", secret+`{"t":"call","id":1,"k":"x","m":"Probe.Touch","a":[]}`)
	// A page of an origin the app does not trust, which posts with the handler
	// directly, with a secret it made up, from each event of its leaving and
	// from a burst that its leaving starts.
	page := `<p>elsewhere</p><script>
const post = (m) => {
  const h = window.webkit && webkit.messageHandlers && webkit.messageHandlers.mygo;
  if (h) { h.postMessage(m); return; }
  if (window.chrome && chrome.webview) chrome.webview.postMessage(m);
};
const send = () => { post(` + hello + `); post(` + call + `); };
for (const name of ["pagehide", "visibilitychange", "unload"]) addEventListener(name, send);
document.addEventListener("freeze", send);
addEventListener("pagehide", () => {
  const c = new MessageChannel();
  c.port1.onmessage = send;
  for (let i = 0; i < 5; i++) c.port2.postMessage(0);
});
</script>`
	for i := 1; i <= rounds; i++ {
		before := probe.n.Load()
		p.LoadHTML(page, "https://untrusted.example/")
		waitFor(t, w, "window.mygo")
		mark := fmt.Sprintf("window.left_%d", i)
		if _, err := p.Eval(mark + " = true"); err != nil {
			t.Fatal(err)
		}
		if err := p.LoadURL("app://localhost/"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, w, mark+" === undefined && window.mygo && document.readyState === 'complete'")
		// The next document's own call goes through the same channel after
		// whatever the left one posted that has arrived.
		if _, err := p.EvalContext(wait.Context(t), "mygo.call('Probe.Touch')"); err != nil {
			t.Fatal(err)
		}
		if n := probe.n.Load(); n != before+1 {
			t.Fatalf("round %d: Probe.Touch ran %d times, want 1 (the next document's own): a message of the left document ran", i, n-before)
		}
	}
	heard, other, others, commits := messageStats()
	t.Logf("%d rounds: %d messages heard, %d of them from a document other than the one shown, %d commits; first of those: %q", rounds, heard, other, commits, others)
	if other != 0 {
		t.Errorf("%d messages came from a document other than the one shown, first: %q", other, others)
	}
}

// Back to a page after another has been shown: whether the engine restores
// it from its cache (pageshow persisted), how many commits the Back makes, and
// that the page's bridge still takes its calls. The first two are logged.
func TestBackToAPageKeepsItsBridge(t *testing.T) {
	resetProbe(t)
	mygo.RunOnMain(win.TestWatchMessages)
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	first := `<p>first</p><script>
window.shows = [];
addEventListener("pageshow", (e) => window.shows.push(e.persisted));
</script>`
	p.LoadHTML(first, "app://localhost/first")
	waitFor(t, w, "window.mygo && window.shows.length > 0")
	if _, err := p.Eval("window.firstDocument = true"); err != nil {
		t.Fatal(err)
	}
	p.LoadHTML(`<p>elsewhere</p>`, "https://untrusted.example/")
	waitFor(t, w, "window.mygo && window.firstDocument === undefined && location.hostname === 'untrusted.example'")
	_, _, _, commitsBefore := messageStats()
	p.GoBack()
	waitFor(t, w, "location.pathname === '/first' && window.mygo && document.readyState === 'complete'")
	v, err := p.EvalContext(wait.Context(t), "JSON.stringify([window.firstDocument === true, window.shows])")
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, commitsAfter := messageStats()
	t.Logf("Back: the same document restored (kept its state): %v; %d commits for the Back", v, commitsAfter-commitsBefore)
	before := probe.n.Load()
	if _, err := p.EvalContext(wait.Context(t), "mygo.call('Probe.Touch')"); err != nil || probe.n.Load() != before+1 {
		t.Fatalf("the page's call after Back: %v, count %d, want %d", err, probe.n.Load(), before+1)
	}
}

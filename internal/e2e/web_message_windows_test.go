//go:build windows

package e2e

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/internal/windows"
)

// A call that a page made and a navigation outlived is answered to nobody: the
// reply of the first document is not posted to the web message listeners of
// the one that replaced it, which is of another origin and records every web
// message (/spy). The tick that follows shows the listener works.
func TestWebMessageReplyDoesNotReachANewDocument(t *testing.T) {
	w := mygo.NewWindow(mygo.WindowOptions{Title: "Bridge", Width: 400, Height: 300})
	t.Cleanup(w.Destroy)
	p := w.Page()
	if err := p.LoadURL("app://localhost/"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "window.run && window.mygo")
	benchHold = make(chan struct{})
	if _, err := p.Eval("window.held = mygo.call('Bench.Hold').catch(() => 'rejected'); 0"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-benchHeld:
	case <-time.After(wait.Guard):
		t.Fatal("the call did not reach Go")
	}
	if err := p.LoadURL("assets://localhost/spy"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "window.__spy !== undefined && window.mygo && document.readyState === 'complete'")
	close(benchHold) // the first document's call is answered now
	// Events wait for the new document's dom-ready and are posted after what
	// came before them: when the tick is in the spy, the reply was posted, or dropped.
	if err := benchTick.Emit(w, Tick{N: 4242}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "window.__spy.some((m) => m.includes('4242'))")
	log, err := mygo.EvalAs[string](p, "window.__spy.join('\\n')")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("what the spy was posted:\n%.600s", log)
	if strings.Contains(log, "FIRST-DOCUMENT-SECRET") {
		t.Fatalf("the new document was posted the reply of the first:\n%s", log)
	}
}

// TestWebMessageReplyLatency times a page's sequential calls to Go with the
// reply posted as a web message and as a script, to compare them on this
// machine. It logs, and asserts nothing about time.
func TestWebMessageReplyLatency(t *testing.T) {
	const n = 2000
	run := func(asScript bool) time.Duration {
		windows.PostMessagesAsScripts.Store(asScript)
		defer windows.PostMessagesAsScripts.Store(false)
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Bench", Width: 400, Height: 300})
		defer w.Destroy()
		if err := w.Page().LoadURL("app://localhost/"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, w, "window.run && window.mygo")
		for range 3 { // warm up
			if _, err := mygo.EvalAs[int](w.Page(), "(async () => { for (let i = 0; i < 200; i++) await mygo.call('Bench.Noop'); return 0 })()"); err != nil {
				t.Fatal(err)
			}
		}
		start := time.Now()
		got, err := mygo.EvalAs[int](w.Page(), "(async (N) => { for (let i = 0; i < N; i++) await mygo.call('Bench.Noop'); return N })("+strconv.Itoa(n)+")")
		took := time.Since(start)
		if err != nil || got != n {
			t.Fatalf("the page made %d of %d calls: %v", got, n, err)
		}
		return took
	}
	for _, c := range []struct {
		name     string
		asScript bool
	}{{"script", true}, {"web message", false}, {"script", true}, {"web message", false}} {
		took := run(c.asScript)
		t.Logf("%s: %.1f µs a call (%d calls in %v)", c.name, float64(took.Microseconds())/n, n, took.Round(time.Millisecond))
	}
}

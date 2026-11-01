//go:build !js

package mygo

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// The bridge makes a secret for each document and says it in a hello, which a
// window takes once after a navigation commits. These tests hold a window to
// that through the fake backend: the secret of one document is of no use to the
// next, and to no frame that did not make it.

const (
	secretA = "00112233445566778899aabbccddeeff"
	secretB = "ffeeddccbbaa99887766554433221100"
	secretC = "0f1e2d3c4b5a69788796a5b4c3d2e1f0"
)

// hello posts a hello with s, as the bridge does at document start.
func hello(fw *fake.Window, s string) { onMain(func() { fw.H.Message(helloMessage(s)) }) }

// commit tells w that its main frame committed url.
func commit(fw *fake.Window, url string) { onMain(func() { fw.H.NavigationCommitted(url) }) }

// takes reports whether w takes a message prefixed with s: it posts "dom-ready",
// which a window that took it has seen. The window is left as a commit left it,
// waiting for it.
func takes(w *Window, fw *fake.Window, s string) bool {
	onMain(func() { fw.H.Message(s + `{"t":"dom-ready"}`) })
	w.outMu.Lock()
	defer w.outMu.Unlock()
	was := w.domReady
	w.domReady = false
	return was
}

// newPageWindow returns a window that has not navigated, so it is not waiting for
// a hello.
func newPageWindow(t *testing.T, opts WindowOptions) (*Window, *fake.Window) {
	t.Helper()
	w := NewWindow(opts)
	t.Cleanup(w.Destroy)
	wins := fb.Windows()
	return w, wins[len(wins)-1]
}

func TestHelloIsTakenOnce(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	if takes(w, fw, secretA) {
		t.Fatal("a message before the hello was taken")
	}
	hello(fw, secretA)
	if !takes(w, fw, secretA) {
		t.Fatal("the message of the secret a hello said was not taken")
	}
	hello(fw, secretB) // a second one, as an iframe or the page itself may post
	if takes(w, fw, secretB) {
		t.Error("the secret of a second hello was taken")
	}
	if !takes(w, fw, secretA) {
		t.Error("a second hello replaced the secret")
	}
}

func TestHelloBeforeTheCommitIsDropped(t *testing.T) {
	w, fw := newPageWindow(t, WindowOptions{})
	hello(fw, secretA)
	if takes(w, fw, secretA) {
		t.Fatal("a hello before any commit was taken")
	}
	commit(fw, "about:blank")
	if takes(w, fw, secretA) {
		t.Error("the commit armed a hello that came before it")
	}
	hello(fw, secretA)
	if !takes(w, fw, secretA) {
		t.Error("the hello after the commit was not taken")
	}
}

func TestACommitRotatesTheSecret(t *testing.T) {
	// A navigation to another page, and a reload of the same one.
	for _, url := range []string{"https://other.example/", "about:blank"} {
		w, fw := testWindow(t, WindowOptions{})
		hello(fw, secretA)
		if !takes(w, fw, secretA) {
			t.Fatalf("%s: the first secret was not taken", url)
		}
		commit(fw, url)
		if takes(w, fw, secretA) {
			t.Errorf("%s: the secret of the document before the commit was taken", url)
		}
		hello(fw, secretB)
		if !takes(w, fw, secretB) || takes(w, fw, secretA) {
			t.Errorf("%s: the new document's hello did not replace the secret", url)
		}
	}
}

func TestAnyMainFrameLoadArmsAHello(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	prev := ""
	for i, url := range []string{
		"about:blank",
		"data:text/html,<p>x</p>",
		"about:srcdoc",
		"file:///app/index.html",
		"https://example.com/page#section",
	} {
		s := []string{secretA, secretB, secretC, secretA, secretB}[i]
		commit(fw, url)
		if prev != "" && takes(w, fw, prev) {
			t.Errorf("%s: the secret of the document before was taken", url)
		}
		hello(fw, s)
		if !takes(w, fw, s) {
			t.Errorf("%s: the hello of the document was not taken", url)
		}
		prev = s
	}
}

// Back and Forward restore a document from the engine's cache: the main frame
// commits, the document's bridge does not start again, and it says the secret it
// has once more (transport.ts, on pageshow). Its messages are taken again, and
// the other document's are not.
func TestARestoredDocumentSaysItsSecretAgain(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	hello(fw, secretA)
	if !takes(w, fw, secretA) {
		t.Fatal("setup: the first document's hello was not taken")
	}
	commit(fw, "about:blank#other")
	hello(fw, secretB) // the other document
	commit(fw, "about:blank")
	onMain(func() { fw.H.LoadFinished() })
	hello(fw, secretA) // the restored one, on pageshow
	if !takes(w, fw, secretA) || takes(w, fw, secretB) {
		t.Error("the restored document's secret was not the window's after its hello")
	}
}

func TestADocumentThatStaysKeepsItsSecret(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	hello(fw, secretA)
	// Same-document navigations (pushState, a hash) commit nothing; the page
	// loading, its title and its focus change meanwhile.
	onMain(func() {
		fw.H.TitleChanged("title")
		fw.H.LoadFinished()
	})
	hello(fw, secretB)
	if !takes(w, fw, secretA) || takes(w, fw, secretB) {
		t.Error("the secret changed with no commit")
	}
}

func TestHelloMustBeASecret(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	for _, bad := range []string{
		``,
		`{"t":"hello"}`,
		`{"t":"hello","k":""}`,
		`{"t":"hello","k":"abc"}`,
		`{"t":"hello","k":"` + strings.ToUpper(secretA) + `"}`,
		`{"t":"hello","k":"` + secretA + `00"}`,
		`{"t":"hello","k":"zz112233445566778899aabbccddeeff"}`,
		`{"t":"hello","k":7}`,
		`{"t":"hello","k":"` + secretA + `","k":"` + secretB + `"}`, // a name twice
		`{"t":"hello","k":"` + secretA + `"}{}`,                     // not one document
	} {
		onMain(func() { fw.H.Message(bad) })
		if takes(w, fw, secretA) || takes(w, fw, secretB) || takes(w, fw, "") {
			t.Fatalf("%q was taken as a hello", bad)
		}
	}
	// None of them used up the wait for the real one.
	hello(fw, secretC)
	if !takes(w, fw, secretC) {
		t.Error("a bad hello made the window refuse the good one")
	}
}

func TestPopupsHaveTheirOwnSecret(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	w.Page().SetWindowOpenHandler(func(WindowOpenRequest) *WindowOptions { return &WindowOptions{} })
	hello(fw, secretA)
	before := len(Windows())
	onMain(func() { fw.H.NewWindow(platform.NewWindowRequest{URL: "https://example.com/", UserInitiated: true}) })
	wins := Windows()
	if len(wins) != before+1 {
		t.Fatalf("windows = %d, want %d", len(wins), before+1)
	}
	child := wins[len(wins)-1]
	fchild := fb.Windows()[len(fb.Windows())-1]
	t.Cleanup(child.Destroy)
	commit(fchild, "https://example.com/")
	if takes(child, fchild, secretA) {
		t.Error("the opener's secret was taken by the popup")
	}
	hello(fchild, secretB)
	if !takes(child, fchild, secretB) {
		t.Error("the popup did not take its own secret")
	}
	if takes(w, fw, secretB) || !takes(w, fw, secretA) {
		t.Error("the popup's hello reached its opener")
	}
}

func TestPagesWithoutBindingsKeepTheirLimits(t *testing.T) {
	for name, opts := range map[string]WindowOptions{
		"untrusted": {Page: PageOptions{Untrusted: true}},
	} {
		w, fw := testWindow(t, opts)
		hello(fw, secretA)
		if !takes(w, fw, secretA) {
			t.Errorf("%s: the hello was not taken", name)
		}
		// A message over what the bridge posts there is dropped before anything.
		long := secretA + `{"t":"dom-ready","pad":"` + strings.Repeat("x", maxUntrustedMessage) + `"}`
		onMain(func() { fw.H.Message(long) })
		w.outMu.Lock()
		got := w.domReady
		w.outMu.Unlock()
		if got {
			t.Errorf("%s: a message over %d bytes was taken", name, maxUntrustedMessage)
		}
	}
}

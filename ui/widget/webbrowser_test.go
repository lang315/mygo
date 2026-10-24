package widget

import (
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

// shownBrowser shows b in a form on the fake backend and returns the form
// and its surface.
func shownBrowser(t *testing.T, b *WebBrowserWidget) (*Form, *fake.Surface) {
	t.Helper()
	f := NewForm("Web", 400, 300).SetContent(b)
	f.Show()
	t.Cleanup(f.Close)
	s := lastFake(t).FakeSurface()
	return f, s
}

// webViews draws a frame and returns the web views of the surface.
func webViews(s *fake.Surface) []*fake.WebView {
	var vs []*fake.WebView
	mygo.RunOnMain(func() { s.Frame(); vs = s.WebViews() })
	return vs
}

// theView waits for the one web view to be made.
func theView(t *testing.T, s *fake.Surface) *fake.WebView {
	t.Helper()
	var v *fake.WebView
	wait.Until(t, "the web view", func() bool {
		if vs := webViews(s); len(vs) == 1 {
			v = vs[0]
		}
		return v != nil
	})
	return v
}

func TestWebBrowserMakesItsViewOnlyOnceTheFormIsShown(t *testing.T) {
	b := WebBrowser().Navigate("https://example.com/a")
	tt := NewTester(t, b, 400, 300) // no window: an empty box
	tt.Frame()
	if b.Url() != "https://example.com/a" {
		t.Fatalf("Url %q before any view", b.Url())
	}
	f, s := shownBrowser(t, WebBrowser())
	_ = f
	v := theView(t, s)
	for i := 0; i < 3; i++ {
		webViews(s)
	}
	if n := len(webViews(s)); n != 1 {
		t.Fatalf("%d web views after several frames", n)
	}
	if v.URL() != "" && !strings.HasPrefix(v.URL(), "about:") {
		t.Errorf("a browser nobody told to navigate loaded %q", v.URL())
	}
}

func TestWebBrowserCreatesItsViewOffTheLock(t *testing.T) {
	var held atomic.Int32
	old := webViewCreateHook
	webViewCreateHook = func() {
		// Another goroutine (a frame) may hold mu for a moment; this one
		// holding it would never let go.
		for i := 0; i < 100000; i++ {
			if mu.TryLock() {
				mu.Unlock()
				return
			}
			runtime.Gosched()
		}
		held.Add(1)
	}
	t.Cleanup(func() { webViewCreateHook = old })
	_, s := shownBrowser(t, WebBrowser())
	theView(t, s)
	if held.Load() != 0 {
		t.Fatal("the web view was made with the package lock held")
	}
}

func TestWebBrowserAppliesWhatWasSetBeforeTheViewExisted(t *testing.T) {
	b := WebBrowser().Navigate("https://example.com/start")
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	wait.Until(t, "the page to load", func() bool { return v.URL() == "https://example.com/start" })

	h := WebBrowser().SetDocumentText("<p>hi</p>")
	_, s2 := shownBrowser(t, h)
	v2 := theView(t, s2)
	wait.Until(t, "the document", func() bool { return len(v2.HTML) == 1 })
	if v2.HTML[0] != "<p>hi</p>" || h.DocumentText() != "<p>hi</p>" {
		t.Fatalf("HTML %q, DocumentText %q", v2.HTML, h.DocumentText())
	}
}

func TestWebBrowserNavigateAfterTheViewExistsLoadsAtOnce(t *testing.T) {
	b := WebBrowser()
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	b.Navigate("https://example.com/later")
	wait.Until(t, "the page", func() bool { return v.URL() == "https://example.com/later" })
	b.GoBack().GoForward().Reload().Stop()
	wait.Until(t, "the commands", func() bool { return strings.Join(v.Navigations(), ",") == "back,forward,reload,stop" })
}

func TestWebBrowserCanGoBackAsksThePage(t *testing.T) {
	b := WebBrowser()
	if b.CanGoBack() || b.CanGoForward() {
		t.Fatal("a browser without a view can go back")
	}
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	v.CanBack = true
	if !b.CanGoBack() || b.CanGoForward() {
		t.Fatalf("CanGoBack %v CanGoForward %v", b.CanGoBack(), b.CanGoForward())
	}
}

func TestWebBrowserEventsArePostedAndNavigatingIsSynchronousOffTheLock(t *testing.T) {
	var navigated, finished atomic.Int32
	url := make(chan string, 4)
	title := make(chan string, 4)
	var b *WebBrowserWidget
	cancelled := false
	b = WebBrowser().
		OnNavigating(func(u string) bool {
			// A getter locks the package: the handler must run off mu.
			_ = b.Url()
			return cancelled
		}).
		OnNavigated(func(u string) {
			navigated.Add(1)
			if !onEvents() {
				t.Error("OnNavigated ran off the event goroutine")
			}
			url <- u
		}).
		OnDocumentCompleted(func() { finished.Add(1) }).
		OnTitleChanged(func(s string) { title <- s })
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	mygo.RunOnMain(func() {
		if !v.H.WillNavigate(navigationTo("https://example.com/x")) {
			t.Error("a navigation was cancelled that the handler allowed")
		}
		v.H.NavigationCommitted("https://example.com/x")
		v.H.LoadFinished()
		v.H.TitleChanged("X page")
	})
	if got := received(t, url); got != "https://example.com/x" {
		t.Fatalf("OnNavigated saw %q", got)
	}
	if got := received(t, title); got != "X page" {
		t.Fatalf("title %q", got)
	}
	wait.Until(t, "OnDocumentCompleted", func() bool { return finished.Load() == 1 })
	cancelled = true
	mygo.RunOnMain(func() {
		if v.H.WillNavigate(navigationTo("https://example.com/y")) {
			t.Error("OnNavigating returned true (cancel) and the navigation went on")
		}
	})
}

func TestWebBrowserKeepsItsPageInAHiddenTabAndGoesWithTheWindow(t *testing.T) {
	b := WebBrowser()
	tabs := TabControl().AddPage("Web", b).AddPage("Other", Label("other"))
	f := NewForm("Tabs", 400, 300).SetContent(tabs)
	f.Show()
	s := lastFake(t).FakeSurface()
	v := theView(t, s)
	tabs.SetPage(1)
	webViews(s)
	tabs.SetPage(0)
	if n := len(webViews(s)); n != 1 || webViews(s)[0] != v {
		t.Fatalf("%d views after switching tabs; the page must stay", n)
	}
	f.Close()
	wait.Until(t, "the view to close", func() bool { return v.IsClosed() })
}

func TestWebBrowserMovedToAnotherFormGetsAFreshViewWithTheCurrentUrl(t *testing.T) {
	b := WebBrowser().Navigate("https://example.com/one")
	f1, s1 := shownBrowser(t, b)
	v1 := theView(t, s1)
	wait.Until(t, "load", func() bool { return v1.URL() == "https://example.com/one" })
	f2 := NewForm("Second", 400, 300).SetContent(b)
	f2.Show()
	t.Cleanup(f2.Close)
	s2 := lastFake(t).FakeSurface()
	v2 := theView(t, s2)
	wait.Until(t, "replay", func() bool { return v2.URL() == "https://example.com/one" })
	_ = f1
	wait.Until(t, "the old view to go", func() bool { return v1.IsClosed() })
}

func TestAWebBrowserPageCannotCallGoItWasNotGiven(t *testing.T) {
	b := WebBrowser().Navigate("https://example.com/")
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	// A page that is not the app's own is untrusted: it has no bridge.
	if len(v.UserScripts()) > 0 {
		for _, u := range v.UserScripts() {
			if strings.Contains(u.Source, "Bind") {
				t.Fatalf("an external page got a binding script: %q", u.Source)
			}
		}
	}
}

func navigationTo(url string) platform.Navigation {
	return platform.Navigation{URL: url, IsMainFrame: true, UserInitiated: true}
}

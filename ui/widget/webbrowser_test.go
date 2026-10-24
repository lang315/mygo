package widget

import (
	"fmt"
	"regexp"
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
	wait.Until(t, "the document", func() bool { return len(v2.Documents()) == 1 })
	if v2.Documents()[0] != "<p>hi</p>" || h.DocumentText() != "<p>hi</p>" {
		t.Fatalf("HTML %q, DocumentText %q", v2.Documents(), h.DocumentText())
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

type browserService struct{ calls atomic.Int32 }

func (b *browserService) Hello(name string) string { b.calls.Add(1); return "hello " + name }

// A page cannot call Go whatever its origin: not an https page, not about:
// or file: pages, and not HTML the app gave with SetDocumentText.
func TestAWebBrowserPageNeverCallsGo(t *testing.T) {
	svc := &browserService{}
	mygo.BindAs("BrowserHello", svc)
	b := WebBrowser().SetDocumentText("<p>app's HTML</p>")
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	secret := regexp.MustCompile(`"secret":"([^"]+)"`).FindStringSubmatch(v.UserScripts()[0].Source)
	if secret == nil {
		t.Fatal("no bridge secret")
	}
	for i, url := range []string{"about:blank", "file:///tmp/x.html", "https://example.com/", "http://localhost:5173/"} {
		mygo.RunOnMain(func() { v.H.NavigationCommitted(url) })
		id := 300 + i
		msg := fmt.Sprintf(`{"t":"call","id":%d,"k":"tok","m":"BrowserHello.Hello","a":["x"]}`, id)
		mygo.RunOnMain(func() { v.H.Message(secret[1] + msg) })
		wait.Until(t, "the reply", func() bool {
			for _, sc := range v.Scripts() {
				if strings.Contains(sc, fmt.Sprintf(`"id":%d`, id)) && strings.Contains(sc, `"ok":false`) {
					return true
				}
			}
			return false
		})
	}
	if n := svc.calls.Load(); n != 0 {
		t.Fatalf("a page called a Go method %d times", n)
	}
}

func TestWebBrowserRefusesJavascriptURLs(t *testing.T) {
	b := WebBrowser()
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	b.Navigate("JavaScript:alert(1)")
	b.Navigate(" javascript:void(0)")
	<-posted()
	if u := v.URL(); strings.Contains(strings.ToLower(u), "javascript:") {
		t.Fatalf("the page was sent to %q", u)
	}
}

// A browser built in a Tester, or in a plain View, after a Form drew must
// not make its view in that Form's window.
func TestWebBrowserOutsideAFormMakesNoViewInAnotherFormsWindow(t *testing.T) {
	f := NewForm("Plain", 300, 200).SetContent(Label("a form that drew"))
	f.Show()
	t.Cleanup(f.Close)
	s := lastFake(t).FakeSurface()
	webViews(s)
	b := WebBrowser().Navigate("https://example.com/")
	tt := NewTester(t, b, 300, 200)
	tt.Frame()
	tt.Settle()
	if n := len(webViews(s)); n != 0 {
		t.Fatalf("a browser built by a Tester made %d web view(s) in a form's window", n)
	}
}

func TestAPanickingOnNavigatingIsReportedAndTheNavigationGoesOn(t *testing.T) {
	var reported atomic.Int32
	OnPanic(func(v any, stack []byte) { reported.Add(1) })
	t.Cleanup(func() { OnPanic(nil) })
	b := WebBrowser().OnNavigating(func(string) bool { panic("boom") })
	_, s := shownBrowser(t, b)
	v := theView(t, s)
	allowed := false
	mygo.RunOnMain(func() { allowed = v.H.WillNavigate(navigationTo("https://example.com/")) })
	if !allowed || reported.Load() != 1 {
		t.Fatalf("navigation allowed %v, panics reported %d", allowed, reported.Load())
	}
}

// A Navigate that comes while a new view takes the page to load wins.
func TestWebBrowserNavigateDuringTheFirstLoadWins(t *testing.T) {
	paused := make(chan struct{}, 1)
	release := make(chan struct{})
	old := webViewReplayHook
	webViewReplayHook = func() {
		select {
		case paused <- struct{}{}:
			<-release
		default:
		}
	}
	t.Cleanup(func() { webViewReplayHook = old })
	b := WebBrowser().Navigate("https://example.com/old")
	_, s := shownBrowser(t, b)
	wait.Until(t, "the replay to pause", func() bool {
		webViews(s)
		select {
		case <-paused:
			return true
		default:
			return false
		}
	})
	done := make(chan struct{})
	go func() { b.Navigate("https://example.com/new"); close(done) }()
	wait.Until(t, "Navigate to record the new URL", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return b.url == "https://example.com/new"
	})
	close(release)
	wait.For(t, "Navigate", done)
	v := theView(t, s)
	wait.Until(t, "the newest page", func() bool { return v.URL() == "https://example.com/new" })
}

func TestWebBrowserViewIsMadeAgainIfTheFormIsShownAgainAfterAClosedCreation(t *testing.T) {
	// The form closes between the build and the creation.
	gate := make(chan struct{})
	old := webViewCreateHook
	webViewCreateHook = func() { <-gate }
	b := WebBrowser()
	f := NewForm("Again", 300, 200).SetContent(b)
	f.Show()
	s1 := lastFake(t).FakeSurface()
	mygo.RunOnMain(func() { s1.Frame() })
	f.Close()
	f.Show() // shown again while the first creation is still waiting
	t.Cleanup(f.Close)
	s2 := lastFake(t).FakeSurface()
	mygo.RunOnMain(func() { s2.Frame() }) // its build finds a creation under way
	close(gate)
	webViewCreateHook = old
	theView(t, s2)
}

func navigationTo(url string) platform.Navigation {
	return platform.Navigation{URL: url, IsMainFrame: true, UserInitiated: true}
}

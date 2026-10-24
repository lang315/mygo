package widget

import (
	"log"
	"runtime/debug"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// buildingForm is the form whose view is building. Only views touch it,
// under mu.
var buildingForm *Form

// webViewCreateHook runs just before a web view is made, for tests to check
// that no lock is held.
var webViewCreateHook atomic.Pointer[func()]

// runOnMain is mygo.RunOnMain; tests count its calls.
var runOnMain = mygo.RunOnMain

// webViewReplayHook runs after a new view took the page to load, before it
// loads it.
var webViewReplayHook atomic.Pointer[func()]

// WebBrowserWidget shows a web page in a window of native UI: the system's
// web view (WKWebView, WebView2, WebKitGTK) over ui.WebView. See
// docs/ui/widget/webbrowser.md.
//
// The web view is made, outside the package's lock, on the event goroutine (a
// slow handler delays it; it then runs on the main thread), the first frame after
// the control is built in a Form that is shown; until then it draws an empty
// box, and what was set is remembered and applied when the view exists. A
// control has one web view: moved to another form it loses it, and the new
// form's gets the current URL or document, without the history.
type WebBrowserWidget struct {
	common[WebBrowserWidget]
	url, html  string
	hasHTML    bool
	background string

	form     *Form         // the form the view belongs to
	view     *mygo.WebView // nil until made
	creating bool
	seq      uint64 // counts Navigate and SetDocumentText: the latest load wins

	onNavigating func(url string) bool
	onNavigated  func(url string)
	onCompleted  func()
	onTitle      func(title string)
	onDownload   func(url, suggestedName string) (path string, ok bool)
	onNewWindow  func(url string)
}

// WebBrowser makes a browser with no page.
func WebBrowser() *WebBrowserWidget {
	b := &WebBrowserWidget{}
	b.init(b)
	return b
}

// page runs fn on the page if the view exists. Page's methods are safe from
// any goroutine.
func (b *WebBrowserWidget) page(fn func(p *mygo.Page)) {
	mu.Lock()
	v := b.view
	mu.Unlock()
	if v != nil && !v.IsDestroyed() {
		fn(v.Page())
	}
}

// Navigate loads url. A javascript: URL is refused (logged and ignored, and
// nothing is stored). A URL with a scheme (http, https, or one registered
// with Protocol.Handle) is a page of the web; one without, such as "/docs",
// is a page of the app's frontend. A URL that cannot be loaded is logged.
func (b *WebBrowserWidget) Navigate(url string) *WebBrowserWidget {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(url)), "javascript:") {
		log.Printf("widget: WebBrowser: Navigate refuses a javascript: URL")
		return b
	}
	var seq uint64
	b.change(func() { b.seq++; seq = b.seq; b.url, b.html, b.hasHTML = url, "", false })
	b.load(seq, func(p *mygo.Page) { loadURL(p, url) })
	return b
}

func loadURL(p *mygo.Page, url string) {
	if err := p.LoadURL(url); err != nil {
		log.Printf("widget: WebBrowser: %v", err)
	}
}

// load runs fn on the page on the main thread, if the view exists and no
// Navigate or SetDocumentText came since seq: the latest wins, whatever
// goroutine got there first, and nothing waits for a lock across the main
// thread.
func (b *WebBrowserWidget) load(seq uint64, fn func(p *mygo.Page)) {
	mu.Lock()
	has := b.view != nil
	mu.Unlock()
	if !has {
		return // nothing to load into: it is applied when the view is made, and no main loop is needed
	}
	runOnMain(func() {
		mu.Lock()
		v, cur := b.view, b.seq
		mu.Unlock()
		if v != nil && !v.IsDestroyed() && cur == seq {
			fn(v.Page())
		}
	})
}

// SetDocumentText loads html as the page; relative URLs in it have no base.
func (b *WebBrowserWidget) SetDocumentText(html string) *WebBrowserWidget {
	var seq uint64
	b.change(func() { b.seq++; seq = b.seq; b.html, b.hasHTML, b.url = html, true, "" })
	b.load(seq, func(p *mygo.Page) { p.LoadHTML(html, "") })
	return b
}

// DocumentText returns the HTML last given to SetDocumentText, "" after a
// Navigate: the page's own document is not read back.
func (b *WebBrowserWidget) DocumentText() string { mu.Lock(); defer mu.Unlock(); return b.html }

// Url returns the URL the page is at once it has one, else the URL last
// given to Navigate.
func (b *WebBrowserWidget) Url() string {
	mu.Lock()
	v, url := b.view, b.url
	mu.Unlock()
	if v != nil && !v.IsDestroyed() {
		if u := v.Page().URL(); u != "" {
			return u
		}
	}
	return url
}

// GoBack, GoForward, Reload and Stop act on the page; without a view they do
// nothing.
func (b *WebBrowserWidget) GoBack() *WebBrowserWidget {
	b.page(func(p *mygo.Page) { p.GoBack() })
	return b
}
func (b *WebBrowserWidget) GoForward() *WebBrowserWidget {
	b.page(func(p *mygo.Page) { p.GoForward() })
	return b
}
func (b *WebBrowserWidget) Reload() *WebBrowserWidget {
	b.page(func(p *mygo.Page) { p.Reload() })
	return b
}
func (b *WebBrowserWidget) Stop() *WebBrowserWidget {
	b.page(func(p *mygo.Page) { p.Stop() })
	return b
}

// CanGoBack and CanGoForward ask the page; false without a view.
func (b *WebBrowserWidget) CanGoBack() (ok bool) {
	b.page(func(p *mygo.Page) { ok = p.CanGoBack() })
	return ok
}
func (b *WebBrowserWidget) CanGoForward() (ok bool) {
	b.page(func(p *mygo.Page) { ok = p.CanGoForward() })
	return ok
}

// SetBackgroundColor fills the view with a CSS color until the page paints
// ("light-dark(#fff, #111)" follows the appearance).
func (b *WebBrowserWidget) SetBackgroundColor(css string) *WebBrowserWidget {
	b.change(func() { b.background = css })
	mu.Lock()
	v := b.view
	mu.Unlock()
	if v != nil && !v.IsDestroyed() {
		v.SetBackgroundColor(css)
	}
	return b
}

// OnNavigating runs fn before the page navigates to another URL, as when
// the user follows a link: true cancels the navigation. It is not called for
// Navigate, SetDocumentText, reloads or history. fn runs synchronously on
// the main thread, off the package's lock: it must be quick and must not
// call Invoke or anything that waits for the event goroutine; getters work.
func (b *WebBrowserWidget) OnNavigating(fn func(url string) (cancel bool)) *WebBrowserWidget {
	b.change(func() { b.onNavigating = fn })
	return b
}

// OnDownload allows downloads, which are refused by default: fn runs
// synchronously on the main thread, off the package's lock, like
// OnNavigating's (quick, and nothing that waits for the event goroutine), and
// returns where to save the file, or false to refuse it.
func (b *WebBrowserWidget) OnDownload(fn func(url, suggestedName string) (path string, ok bool)) *WebBrowserWidget {
	b.change(func() { b.onDownload = fn })
	return b
}

// OnNewWindow runs fn, on the event goroutine, with the URL a page asked to
// open in a new window (window.open, a link with target=_blank). No window
// opens by itself: the app decides, say with Navigate.
func (b *WebBrowserWidget) OnNewWindow(fn func(url string)) *WebBrowserWidget {
	b.change(func() { b.onNewWindow = fn })
	return b
}

// OnNavigated runs fn, on the event goroutine, after a navigation committed
// to url.
func (b *WebBrowserWidget) OnNavigated(fn func(url string)) *WebBrowserWidget {
	b.change(func() { b.onNavigated = fn })
	return b
}

// OnDocumentCompleted runs fn, on the event goroutine, after the page
// finished loading.
func (b *WebBrowserWidget) OnDocumentCompleted(fn func()) *WebBrowserWidget {
	b.change(func() { b.onCompleted = fn })
	return b
}

// OnTitleChanged runs fn, on the event goroutine, when the page's title
// changes.
func (b *WebBrowserWidget) OnTitleChanged(fn func(title string)) *WebBrowserWidget {
	b.change(func() { b.onTitle = fn })
	return b
}

func (b *WebBrowserWidget) build(c *ui.Context) ui.Element {
	f := buildingForm
	if b.view != nil && (b.view.IsDestroyed() || b.form != f) {
		// Its window closed, or the control moved to another form: one view
		// per control.
		old := b.view
		b.view, b.form = nil, nil
		if !old.IsDestroyed() {
			Post(old.Destroy)
		}
	}
	if b.view == nil && !b.creating && f != nil && f.win != nil {
		b.creating = true
		win := f.win
		Post(func() { b.create(f, win) })
	}
	if v := b.view; v != nil {
		return ui.WebView(c, v).Grow(1)
	}
	return ui.Box(c).Grow(1)
}

// create makes the view, on the event goroutine with no lock held.
func (b *WebBrowserWidget) create(f *Form, win *mygo.Window) {
	if h := webViewCreateHook.Load(); h != nil {
		(*h)()
	}
	mu.Lock()
	bg := b.background
	mu.Unlock()
	v, err := win.NewWebView(mygo.WebViewOptions{BackgroundColor: bg, NoBindings: true})
	if err != nil {
		mu.Lock()
		b.creating = false
		cur := f.win
		mu.Unlock()
		if cur == win {
			log.Printf("widget: WebBrowser: %v", err)
		} else if cur != nil {
			cur.Invalidate() // the form was shown again meanwhile: build anew
		}
		return
	}
	p := v.Page()
	p.OnWillNavigate(func(e *mygo.NavigateEvent) {
		mu.Lock()
		h := b.onNavigating
		mu.Unlock()
		if h != nil && cancels(h, e.URL) {
			e.PreventDefault()
		}
	})
	p.OnDidNavigate(func(url string) {
		mu.Lock()
		h := b.onNavigated
		mu.Unlock()
		if h != nil {
			Post(func() { h(url) })
		}
	})
	p.OnDidFinishLoad(func() {
		mu.Lock()
		h := b.onCompleted
		mu.Unlock()
		if h != nil {
			Post(h)
		}
	})
	p.OnWillDownload(func(e *mygo.DownloadEvent) {
		mu.Lock()
		h := b.onDownload
		mu.Unlock()
		if h == nil {
			return // a view without bindings saves nothing by default
		}
		if path, ok := downloads(h, e.URL, e.SuggestedName); ok {
			e.Path = path
		} else {
			e.PreventDefault()
		}
	})
	p.SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
		mu.Lock()
		h := b.onNewWindow
		mu.Unlock()
		if h != nil {
			url := req.URL
			Post(func() { h(url) })
		}
		return nil // a WebBrowser opens no window of its own
	})
	p.OnPageTitleUpdated(func(e *mygo.TitleEvent) {
		mu.Lock()
		h := b.onTitle
		mu.Unlock()
		if h != nil {
			title := e.Title
			Post(func() { h(title) })
		}
	})
	mu.Lock()
	b.view, b.form, b.creating = v, f, false
	url, html, hasHTML, seq := b.url, b.html, b.hasHTML, b.seq
	mu.Unlock()
	if h := webViewReplayHook.Load(); h != nil {
		(*h)()
	}
	switch {
	case hasHTML:
		b.load(seq, func(p *mygo.Page) { p.LoadHTML(html, "") })
	case url != "":
		b.load(seq, func(p *mygo.Page) { loadURL(p, url) })
	}
	win.Invalidate()
}

// cancels runs the handler of OnNavigating, which runs on the main thread
// where a panic would take the app down: it is reported as the other
// handlers' are, and the navigation goes on.
func cancels(h func(string) bool, url string) (cancel bool) {
	defer func() {
		if v := recover(); v != nil {
			report(v, debug.Stack())
			cancel = false
		}
	}()
	return h(url)
}

// downloads runs the handler of OnDownload where a panic would take the app
// down: it is reported and the download refused.
func downloads(h func(url, name string) (string, bool), url, name string) (path string, ok bool) {
	defer func() {
		if v := recover(); v != nil {
			report(v, debug.Stack())
			path, ok = "", false
		}
	}()
	return h(url, name)
}

package widget

import (
	"log"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// buildingForm is the form whose view is building. Only views touch it,
// under mu.
var buildingForm *Form

// webViewCreateHook runs just before a web view is made, for tests to check
// that no lock is held.
var webViewCreateHook func()

// webViewReplayHook runs after a new view took the page to load, before it
// loads it.
var webViewReplayHook func()

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
	loadMu   sync.Mutex // orders the first load of a new view and later Navigates

	onNavigating func(url string) bool
	onNavigated  func(url string)
	onCompleted  func()
	onTitle      func(title string)
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

// Navigate loads url. A javascript: URL is refused (logged and ignored). A
// URL with a scheme (http, https, or one registered
// with Protocol.Handle) is a page of the web; one without, such as "/docs",
// is a page of the app's frontend. A URL that cannot be loaded is logged.
func (b *WebBrowserWidget) Navigate(url string) *WebBrowserWidget {
	b.change(func() { b.url, b.html, b.hasHTML = url, "", false })
	b.loadMu.Lock()
	defer b.loadMu.Unlock()
	b.page(func(p *mygo.Page) { loadURL(p, url) })
	return b
}

func loadURL(p *mygo.Page, url string) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(url)), "javascript:") {
		log.Printf("widget: WebBrowser: Navigate refuses a javascript: URL")
		return
	}
	if err := p.LoadURL(url); err != nil {
		log.Printf("widget: WebBrowser: %v", err)
	}
}

// SetDocumentText loads html as the page; relative URLs in it have no base.
func (b *WebBrowserWidget) SetDocumentText(html string) *WebBrowserWidget {
	b.change(func() { b.html, b.hasHTML, b.url = html, true, "" })
	b.loadMu.Lock()
	defer b.loadMu.Unlock()
	b.page(func(p *mygo.Page) { p.LoadHTML(html, "") })
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
	if h := webViewCreateHook; h != nil {
		h()
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
	p.OnPageTitleUpdated(func(e *mygo.TitleEvent) {
		mu.Lock()
		h := b.onTitle
		mu.Unlock()
		if h != nil {
			title := e.Title
			Post(func() { h(title) })
		}
	})
	b.loadMu.Lock()
	defer b.loadMu.Unlock()
	mu.Lock()
	b.view, b.form, b.creating = v, f, false
	url, html, hasHTML := b.url, b.html, b.hasHTML
	mu.Unlock()
	if h := webViewReplayHook; h != nil {
		h()
	}
	switch {
	case hasHTML:
		p.LoadHTML(html, "")
	case url != "":
		loadURL(p, url)
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

# WebBrowser

`WebBrowser()` shows a web page in a window: the system's web view (WKWebView,
WebView2, WebKitGTK) over [`ui.WebView`](../web-view.md).

```go
var browser *widget.WebBrowserWidget
browser = widget.WebBrowser().
	Navigate("https://example.com").
	OnNavigated(func(url string) { address.SetText(url) }).
	OnNavigating(func(url string) bool { return blocked(url) })
back := widget.Button("Back").OnClick(func() { browser.GoBack() })
```

- `Navigate(url)`: a `javascript:` URL is refused (logged and ignored). A URL with a scheme (`https://…`, or one registered with
  `Protocol.Handle`) is a page of the web; one without, as `"/docs"`, is a page
  of the app's frontend. A URL that cannot be loaded is logged.
- `SetDocumentText(html)` loads HTML, with no base URL. `DocumentText()` returns
  what was set (the page's own document is not read back), `Url()` the URL the
  page is at.
- `GoBack`, `GoForward`, `Reload`, `Stop`, `CanGoBack`, `CanGoForward`,
  `SetBackgroundColor(css)` (the color the view fills with until the page
  paints).
- `OnDownload(url, name) (path, ok)` allows downloads and `OnNewWindow(url)` hears
  of the windows a page asked to open (nothing opens by itself); both are below.
- Events: `OnNavigated(url)`, `OnDocumentCompleted()` and `OnTitleChanged(title)`
  run on the event goroutine. **`OnNavigating(url) (cancel bool)` runs
  synchronously on the main thread, off the package's lock**: it must be quick
  and must not call `Invoke` or anything that waits for the event goroutine
  (which may be blocked); widget getters work. It is not called for
  `Navigate`, `SetDocumentText`, reloads or history.

## The view

The web view is made the first frame after the control is built in a form that
is **shown**, outside the package's lock: a Posted function on the event
goroutine makes it (on the main thread, as the platform needs), so a slow
handler delays it. Until then the
control draws an empty box, and `Navigate`, `SetDocumentText` and
`SetBackgroundColor` are remembered and applied when the view exists; so
without a window (a `widget.Tester`) nothing loads, `Url()` returns what
`Navigate` was given, and `GoBack`, `CanGoBack` and the like do nothing.

A control has **one** web view. It keeps its page while it is not built (in a
tab not shown, or hidden), and goes with its window. Moved to another form (or
added to two) it loses it: the new form gets a new view, with the current URL
or document and without the history.

**No page in a WebBrowser calls Go**, whatever its origin (the view is made with
`WebViewOptions.NoBindings`): not an `https` page, not a `file:` or `about:`
page, not HTML given to `SetDocumentText`, not a trusted origin. A WebBrowser
shows content that is not the app's own, as WinForms' does. For a page of
your frontend that calls the bound methods, use `ui.WebView` and
`mygo.Window.NewWebView` directly, with content rules and scripting too.

## What a page in a WebBrowser cannot do

The view is made with `WebViewOptions.NoBindings`, which makes it untrusted for
everything a page can ask of the host, unless the app says otherwise:

- **Call Go:** never, whatever the origin.
- **Use permissions:** denied unless the app's `SetPermissionHandler` allows
  them. Camera and microphone are routed through it on macOS; camera,
  microphone, location and notifications on Linux and Windows. The other kinds
  (clipboard read, file system access, local fonts, MIDI sysex, window
  management...) never reach the handler: macOS and Linux leave them to the
  engine's own default answer, and Windows denies them to this view
  instead of asking the user.
- **Read responses of your custom schemes** from another origin: no CORS
  headers, even for `TrustedOrigins`.
- **Move, resize or maximize your window**, or hand it dropped files: only the
  app's own pages can.
- **Download:** refused, unless `OnDownload` returns a path. `OnDownload` runs
  synchronously on the main thread, off the package's lock, like
  `OnNavigating`: quick, and nothing that waits for the event goroutine.
- **Open windows** (`window.open`, `target=_blank`): none opens; `OnNewWindow`
  is told the URL, and the app may `Navigate` there.
- **Run `javascript:` URLs** from `Navigate`: refused, and nothing is stored.

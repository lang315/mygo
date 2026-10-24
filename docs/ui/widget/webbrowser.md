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

- `Navigate(url)`: a URL with a scheme (`https://…`, or one registered with
  `Protocol.Handle`) is a page of the web; one without, as `"/docs"`, is a page
  of the app's frontend. A URL that cannot be loaded is logged.
- `SetDocumentText(html)` loads HTML, with no base URL. `DocumentText()` returns
  what was set (the page's own document is not read back), `Url()` the URL the
  page is at.
- `GoBack`, `GoForward`, `Reload`, `Stop`, `CanGoBack`, `CanGoForward`,
  `SetBackgroundColor(css)` (the color the view fills with until the page
  paints).
- Events: `OnNavigated(url)`, `OnDocumentCompleted()` and `OnTitleChanged(title)`
  run on the event goroutine. **`OnNavigating(url) (cancel bool)` runs
  synchronously on the main thread, off the package's lock**: it must be quick
  and must not call `Invoke` or anything that waits for the event goroutine
  (which may be blocked); widget getters work. It is not called for
  `Navigate`, `SetDocumentText`, reloads or history.

## The view

The web view is made the first frame after the control is built in a form that
is **shown**, outside the package's lock and on the main thread. Until then the
control draws an empty box, and `Navigate`, `SetDocumentText` and
`SetBackgroundColor` are remembered and applied when the view exists; so
without a window (a `widget.Tester`) nothing loads, `Url()` returns what
`Navigate` was given, and `GoBack`, `CanGoBack` and the like do nothing.

A control has **one** web view. It keeps its page while it is not built (in a
tab not shown, or hidden), and goes with its window. Moved to another form (or
added to two) it loses it: the new form gets a new view, with the current URL
or document and without the history.

A page that is not the app's own, as one at `https://example.com`, cannot call
Go: see [who may call](../../bindings.md#who-may-call). Content rules,
scripting and bindings need `ui.WebView` and `mygo.Window.NewWebView` directly.

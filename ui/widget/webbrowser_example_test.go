package widget_test

import "github.com/egoist/mygo/ui/widget"

func blocked(string) bool { return false }

// The snippet of docs/ui/widget/webbrowser.md, which must compile.
func ExampleWebBrowser() {
	address := widget.Label("")
	var browser *widget.WebBrowserWidget
	browser = widget.WebBrowser().
		Navigate("https://example.com").
		OnNavigated(func(url string) { address.SetText(url) }).
		OnNavigating(func(url string) bool { return blocked(url) })
	back := widget.Button("Back").OnClick(func() { browser.GoBack() })
	_ = back
}

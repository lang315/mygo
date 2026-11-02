//go:build mygo_nowebview

package mygo

import (
	"strings"
	"testing"
)

// In a build with nativeOnly, no platform makes a web view: a window that
// would show a page is refused (newWindow panics with the error, as the
// Windows backend does), and NewWebView (so widget.WebBrowser) is an error.
// (The other tests of the package make web views: run this one alone, as
// `go test -tags mygo_nowebview -run TestNativeOnly`.)
func TestNativeOnlyRefusesWebViews(t *testing.T) {
	err := webViewAllowed(false)
	if err == nil || !strings.Contains(err.Error(), "nativeOnly") || !strings.Contains(err.Error(), "no web view") {
		t.Errorf("a window showing a page: %v, want an error that says nativeOnly and no web view", err)
	}
	if err := webViewAllowed(true); err != nil {
		t.Errorf("a window of native UI: %v", err)
	}
	if _, err := (&Window{}).NewWebView(WebViewOptions{}); err != errNativeOnly {
		t.Errorf("NewWebView = %v, want errNativeOnly", err)
	}
}

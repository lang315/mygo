//go:build darwin && mygo_nowebview

package darwin

import (
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"
)

// In a build with nativeOnly there is no web view to make: the window delegate
// has none of the protocols and methods of WebKit, and the error that stands
// in says what to change. (It registers no class: another test of the package
// registers a stand-in for the web view's.)
func TestNoWebViewInANativeOnlyBuild(t *testing.T) {
	if len(webViewProtocols) != 0 {
		t.Errorf("the window delegate answers the protocols %v of WebKit", webViewProtocols)
	}
	if m := webViewDelegateMethods(); len(m) != 0 {
		t.Errorf("the window delegate has %d methods of WebKit", len(m))
	}
	// The window delegate keeps what a window of native UI needs, the
	// observing of its appearance (a window with traffic lights) among it:
	// without the method, AppKit raises an exception when the appearance
	// changes.
	load()
	have := map[objc.SEL]bool{}
	for _, m := range windowDelegateMethods() {
		have[m.Cmd] = true
	}
	for _, name := range []string{"windowShouldClose:", "windowDidResize:", "observeValueForKeyPath:ofObject:change:context:"} {
		if !have[sel(name)] {
			t.Errorf("the window delegate lacks %s", name)
		}
	}
	if have[sel("webView:didCommitNavigation:")] {
		t.Error("the window delegate has WebKit's webView:didCommitNavigation:")
	}
	for _, want := range []string{"nativeOnly", "no web view"} {
		if !strings.Contains(errNativeOnly.Error(), want) {
			t.Errorf("the error %q does not say %q", errNativeOnly, want)
		}
	}
	defer func() {
		if err, _ := recover().(error); err != errNativeOnly {
			t.Errorf("createWebView panicked with %v, want errNativeOnly", err)
		}
	}()
	(&window{}).createWebView(NSRect{})
}

//go:build darwin && !mygo_nowebview

package darwin

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// The window delegate answers what WebKit calls and the classes of web views
// are registered, with the others, when the build has a web view.
func TestWebViewClassesAreRegistered(t *testing.T) {
	load()
	registerClasses()
	for _, name := range []string{"MyGoWebView", "MyGoSchemeHandler"} {
		if objc.GetClass(name) == 0 {
			t.Errorf("the class %s is not registered", name)
		}
	}
	delegate := id(objc.GetClass("MyGoWindowDelegate"))
	for _, name := range []string{"windowShouldClose:", "webView:didCommitNavigation:", "userContentController:didReceiveScriptMessage:", "download:didFailWithError:resumeData:"} {
		if !sendBool(delegate, "instancesRespondToSelector:", uintptr(sel(name))) {
			t.Errorf("the window delegate does not answer %s", name)
		}
	}
}

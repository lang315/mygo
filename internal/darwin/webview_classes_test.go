//go:build darwin && !mygo_nowebview

package darwin

import (
	"slices"
	"testing"

	"github.com/ebitengine/purego/objc"
)

// The window delegate has the protocols and methods WebKit calls when the build
// has a web view. (It registers no class: another test of the package
// registers a stand-in for the web view's.)
func TestWebViewDelegateHasWebKitsMethods(t *testing.T) {
	load()
	for _, p := range []string{"WKNavigationDelegate", "WKUIDelegate", "WKScriptMessageHandler", "WKDownloadDelegate"} {
		if !slices.Contains(webViewProtocols, p) {
			t.Errorf("the window delegate does not answer %s", p)
		}
	}
	have := map[objc.SEL]bool{}
	for _, m := range webViewDelegateMethods() {
		have[m.Cmd] = true
	}
	for _, name := range []string{"webView:didCommitNavigation:", "userContentController:didReceiveScriptMessage:", "download:didFailWithError:resumeData:", "webView:decidePolicyForNavigationAction:decisionHandler:"} {
		if !have[sel(name)] {
			t.Errorf("the window delegate lacks %s", name)
		}
	}
	if len(have) < 20 {
		t.Errorf("%d methods of WebKit, want about 23", len(have))
	}
}

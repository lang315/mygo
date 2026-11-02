//go:build windows && (amd64 || arm64) && mygo_nowebview

package windows

import (
	"strings"
	"testing"
)

// In a build with nativeOnly there is no web view to make: the environment is
// refused, and a window that asks for its web view has the reason, which says
// what to change.
func TestNoWebViewInANativeOnlyBuild(t *testing.T) {
	if err := (&Backend{}).startEnvironment(nil); err != errNativeOnly {
		t.Errorf("startEnvironment = %v, want errNativeOnly", err)
	}
	w := &window{}
	w.createWebView()
	if w.webViewErr != errNativeOnly {
		t.Errorf("a window's web view failed with %v, want errNativeOnly", w.webViewErr)
	}
	for _, want := range []string{"nativeOnly", "no web view"} {
		if got := errNativeOnly.Error(); !strings.Contains(got, want) {
			t.Errorf("the error %q does not say %q", got, want)
		}
	}
}

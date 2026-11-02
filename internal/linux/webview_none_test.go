//go:build linux && (amd64 || arm64) && mygo_nowebview

package linux

import (
	"strings"
	"testing"
)

// In a build with nativeOnly there is no web view to make: WebKit is never
// loaded, the callbacks of web views are not created (the others are), and
// the error that stands in says what to change.
func TestNoWebViewInANativeOnlyBuild(t *testing.T) {
	if err := webKit(); err != errNativeOnly {
		t.Errorf("webKit = %v, want errNativeOnly", err)
	}
	initWindowCallbacks()
	if cbDeleteEvent == 0 {
		t.Error("the callbacks of windows are not created")
	}
	if cbScriptMessage != 0 || cbLoadChanged != 0 || cbDecidePolicy != 0 {
		t.Error("the callbacks of web views are created in a build without a web view")
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
	(&window{}).createWebView()
}

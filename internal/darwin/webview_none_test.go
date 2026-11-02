//go:build darwin && mygo_nowebview

package darwin

import (
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"
)

// In a build with nativeOnly there is no web view to make: the classes of web
// views are not registered, the window delegate answers what a window needs and
// none of what WebKit calls, and the error that stands in says what to change.
func TestNoWebViewInANativeOnlyBuild(t *testing.T) {
	load()
	registerClasses()
	for _, name := range []string{"MyGoWebView", "MyGoSchemeHandler"} {
		if objc.GetClass(name) != 0 {
			t.Errorf("the class %s is registered in a build without a web view", name)
		}
	}
	delegate := id(objc.GetClass("MyGoWindowDelegate"))
	if delegate == 0 {
		t.Fatal("no window delegate class")
	}
	if !sendBool(delegate, "instancesRespondToSelector:", uintptr(sel("windowShouldClose:"))) {
		t.Error("the window delegate does not answer windowShouldClose:")
	}
	if sendBool(delegate, "instancesRespondToSelector:", uintptr(sel("webView:didCommitNavigation:"))) {
		t.Error("the window delegate answers WebKit's webView:didCommitNavigation:")
	}
	defer func() {
		err, _ := recover().(error)
		if err != errNativeOnly {
			t.Errorf("createWebView panicked with %v, want errNativeOnly", err)
		}
		for _, want := range []string{"nativeOnly", "no web view"} {
			if !strings.Contains(errNativeOnly.Error(), want) {
				t.Errorf("the error %q does not say %q", errNativeOnly, want)
			}
		}
	}()
	(&window{}).createWebView(NSRect{})
}

package mygo

import "errors"

// errNativeOnly is why an app built with the tag mygo_nowebview (`mygo build`
// of a project with nativeOnly) makes no web view: a window that shows a page
// and Window.NewWebView (so widget.WebBrowser) are errors on every platform,
// and the backends of Windows, macOS and Linux leave the code that makes one out of the program.
var errNativeOnly = errors.New("mygo: this app was built with nativeOnly: it has no web view")

// webViewAllowed is nil unless the build has the tag and the window would show
// a page, which newWindow panics on as the Windows backend does.
func webViewAllowed(showsContent bool) error {
	if noWebView && !showsContent {
		return errNativeOnly
	}
	return nil
}

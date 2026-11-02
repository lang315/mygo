//go:build darwin && mygo_nowebview

package darwin

import (
	"errors"

	"github.com/ebitengine/purego/objc"
)

// This file stands in for webview_start.go in a build with the tag
// mygo_nowebview (`mygo build` of a project with nativeOnly): the app has no
// web view, and none of the code that makes one is linked. A window of native
// UI is not affected. Package mygo refuses a window that would show a page and
// NewWebView before they reach the backend; this is the last word if one gets
// here.

var errNativeOnly = errors.New("mygo: this app was built with nativeOnly: it has no web view")

var webViewProtocols []string

func registerWebViewClasses() {}

func webViewDelegateMethods() []objc.MethodDef { return nil }

func (w *window) createWebView(NSRect) { panic(errNativeOnly) }

//go:build linux && (amd64 || arm64) && mygo_nowebview

package linux

import "errors"

// This file stands in for webview_start.go in a build with the tag
// mygo_nowebview (`mygo build` of a project with nativeOnly): the app has no
// web view, and none of the code that makes one is linked. A window of native
// UI is not affected. Package mygo refuses a window that would show a page and
// NewWebView before they reach the backend; this is the last word if one gets
// here.

var errNativeOnly = errors.New("mygo: this app was built with nativeOnly: it has no web view")

func loadWebKit() error { return errNativeOnly }

func initWebCallbacks() {}

func (w *window) createWebView() { panic(errNativeOnly) }

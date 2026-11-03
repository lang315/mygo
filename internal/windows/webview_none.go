//go:build windows && (amd64 || arm64) && mygo_nowebview

package windows

import (
	"errors"

	"github.com/egoist/mygo/internal/platform"
)

// This file stands in for webview_start.go in a build with the tag
// mygo_nowebview (`mygo build` of a project with nativeOnly): the app has no
// web view, and none of the code that makes one is linked. A window of native UI
// is not affected; a window that would show a page, or a web view in a surface,
// is an error that says why.

var errNativeOnly = errors.New("mygo: this app was built with nativeOnly: it has no web view")

func (b *Backend) startEnvironment(*platform.WindowOptions) error { return errNativeOnly }

func (w *window) createWebView() { w.webViewFailed(errNativeOnly) }

func (b *Backend) explainMissingRuntime(error) {}

// A build without a web view places none.
func (s *surface) PlaceWebViews([]platform.WebViewPlacement) {}

func (w *window) placeEmbedded() {}

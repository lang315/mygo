//go:build js

package mygo

import "github.com/egoist/mygo/internal/platform"

// Browsers show no web page, so there is no custom scheme to serve (package
// protocol, which does, is not built for them).

func (h *windowHandler) SchemeRequest(req *platform.SchemeRequest) {}

func (h *windowHandler) SchemeDownload(rawURL string) {}

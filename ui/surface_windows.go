//go:build windows && (amd64 || arm64)

package ui

import (
	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/windows"
)

// Content windows need the Windows backend's surface, which an app of web views
// does not link: it is here, in the package every app of native UI imports.
func init() {
	windows.UseSurface()
	windows.PrewarmFunc = d3d11.Prewarm // the first window finds its device made
}

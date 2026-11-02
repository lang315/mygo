//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"testing"
)

// A window that asks for a surface while no package registered the code that makes
// one (package ui does, when it is imported) is an error that says so, not a
// window that stays blank.
func TestSurfaceWithoutUseSurfaceIsAnError(t *testing.T) {
	saved := makeSurfaceFunc
	defer func() { makeSurfaceFunc = saved }()
	makeSurfaceFunc = nil
	w := &window{}
	if err := w.makeSurface(); !errors.Is(err, errNoSurface) {
		t.Fatalf("makeSurface without UseSurface: %v, want errNoSurface", err)
	}
	if w.sf != nil {
		t.Error("the window got a surface")
	}
	UseSurface()
	if makeSurfaceFunc == nil {
		t.Error("UseSurface did not register the surface")
	}
}

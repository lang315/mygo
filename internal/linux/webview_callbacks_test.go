//go:build linux && (amd64 || arm64) && !mygo_nowebview

package linux

import "testing"

// The callbacks of web views are created with the others when the build has a
// web view.
func TestWebViewCallbacksAreCreated(t *testing.T) {
	initWindowCallbacks()
	if cbDeleteEvent == 0 || cbScriptMessage == 0 || cbLoadChanged == 0 || cbDecidePolicy == 0 || cbPermission == 0 {
		t.Errorf("callbacks: window %v, script message %v, load %v, policy %v, permission %v", cbDeleteEvent, cbScriptMessage, cbLoadChanged, cbDecidePolicy, cbPermission)
	}
}

package widget

import "testing"

// UseCalm turns skin.Calm on, or with false every skin off. A widget_test
// file sets it (usecalm_external_test.go): package skin imports tw, which
// imports widget, so this package's own tests cannot import skin.
var UseCalm func(on bool)

// useCalm turns Calm on or off, failing the test without UseCalm.
func useCalm(t *testing.T, on bool) {
	t.Helper()
	if UseCalm == nil {
		t.Fatal("UseCalm is not set: usecalm_external_test.go is not built")
	}
	UseCalm(on)
}

// calmOff turns every skin off, for a Cleanup.
func calmOff() {
	if UseCalm != nil {
		UseCalm(false)
	}
}

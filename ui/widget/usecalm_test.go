package widget

import "testing"

// UseCalm turns skin.Calm on, or with false every skin off. A widget_test
// file sets it (usecalm_external_test.go): package skin imports tw, which
// imports widget, so this package's own tests cannot import skin.
var UseCalm func(on bool)

// UseSkinNamed turns on the built-in skin of that name ("Calm", "Classic",
// "HighContrast"), compact when compact; UseLooks turns on Calm with the
// looks given, by Looks field name. usecalm_external_test.go sets both.
var (
	UseSkinNamed func(name string, compact bool)
	UseLooks     func(looks map[string]string)
)

// useSkinNamed turns the skin on until the test ends.
func useSkinNamed(t *testing.T, name string, compact bool) {
	t.Helper()
	UseSkinNamed(name, compact)
	t.Cleanup(calmOff)
}

// useLooks turns Calm with the looks on until the test ends.
func useLooks(t *testing.T, looks map[string]string) {
	t.Helper()
	UseLooks(looks)
	t.Cleanup(calmOff)
}

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

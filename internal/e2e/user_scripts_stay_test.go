//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
)

// loadAfterEach loads a document right after each of 200 changes of what a
// window's scripts are, and checks that every one has its bridge.
func loadAfterEach(t *testing.T, change func(i int)) {
	t.Helper()
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	waitFor(t, w, "window.mygo")
	for i := 1; i <= 200; i++ {
		change(i)
		if err := p.LoadURL(fmt.Sprintf("app://localhost/?n=%d", i)); err != nil {
			t.Fatal(err)
		}
		waitFor(t, w, fmt.Sprintf("location.search === '?n=%d' && !!window.mygo", i))
	}
}

// The scripts that do not change stay registered while the others are
// replaced, as the theme changes the preferences script: a document created
// right after each change has its bridge.
func TestDocumentsHaveTheirBridgeWhileThemeChanges(t *testing.T) {
	t.Cleanup(func() { mygo.RunOnMain(func() { mygo.Theme.SetSource(mygo.ThemeSystem) }) })
	loadAfterEach(t, func(i int) {
		source := mygo.ThemeLight
		if i%2 == 0 {
			source = mygo.ThemeDark
		}
		mygo.RunOnMain(func() { mygo.Theme.SetSource(source) })
	})
}

// A change of the language replaces the bridge itself, which carries it, so
// this only observes: it passes with and without the scripts that did not
// change staying registered.
func TestDocumentsHaveTheirBridgeWhileLanguageChanges(t *testing.T) {
	localeLanguages(t, "en", "ja")
	mygo.RunOnMain(func() { mygo.App.SetLocale("en") })
	loadAfterEach(t, func(i int) {
		locale := "en"
		if i%2 == 0 {
			locale = "ja"
		}
		mygo.RunOnMain(func() { mygo.App.SetLocale(locale) })
	})
}

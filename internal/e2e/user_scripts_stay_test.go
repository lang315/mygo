package e2e

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
)

// The bridge stays registered while the scripts of a window are replaced
// (as the app's language changes them): a document created right after
// each change has its bridge.
func TestDocumentsHaveTheirBridgeWhileScriptsChange(t *testing.T) {
	localeLanguages(t, "en", "ja")
	mygo.RunOnMain(func() { mygo.App.SetLocale("en") })
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	waitFor(t, w, "window.mygo")
	for i := 1; i <= 200; i++ {
		locale := "en"
		if i%2 == 0 {
			locale = "ja"
		}
		mygo.RunOnMain(func() { mygo.App.SetLocale(locale) })
		if err := p.LoadURL(fmt.Sprintf("app://localhost/?n=%d", i)); err != nil {
			t.Fatal(err)
		}
		waitFor(t, w, fmt.Sprintf("location.search === '?n=%d' && !!window.mygo", i))
	}
}

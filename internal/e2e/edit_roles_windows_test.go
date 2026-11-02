//go:build windows && (amd64 || arm64)

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
)

// Edit › Paste and Edit › Select All in a page go through the DevTools protocol
// (editRole), with parameters the backend writes itself: the text pasted is the
// clipboard's, quote, backslash, angle brackets, ampersand and non-ASCII
// included, and the page must receive it as it is.
func TestEditRolesInAPage(t *testing.T) {
	prev := mygo.App.Menu()
	defer mygo.App.SetMenu(prev)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleEditMenu}}))
	const text = `a"b\c<d>&e é😀 ü`
	mygo.RunOnMain(func() { mygo.Clipboard.WriteText(text) })
	w := newWindow(t, mygo.WindowOptions{Width: 400, Height: 300})
	w.Page().LoadHTML(`<div id="f" contenteditable="true" style="width:300px;height:100px">x</div>`, "")
	waitFor(t, w, `document.getElementById('f') && document.readyState === 'complete'`)
	if _, err := w.Page().Eval(`(() => { const f = document.getElementById('f'); f.textContent = ''; f.focus(); })()`); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, `document.activeElement && document.activeElement.id === 'f'`)
	if err := activateMenu(w, "Edit", "Paste"); err != nil {
		t.Fatal(err)
	}
	got := func() string {
		s, _ := mygo.EvalAs[string](w.Page(), `document.getElementById('f').textContent`)
		return s
	}
	eventually(t, "the paste", func() bool { return got() != "" })
	if g := got(); g != text {
		t.Errorf("the page has %q after Paste, want %q", g, text)
	}
	if err := activateMenu(w, "Edit", "Select All"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the selection", func() bool {
		s, _ := mygo.EvalAs[string](w.Page(), `getSelection().toString()`)
		return s == text
	})
}

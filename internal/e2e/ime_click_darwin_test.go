package e2e

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// TestRichTextClickEndsCompositionMac clicks in a rich text editor while
// Vietnamese Telex, a real input method, composes a word in it: the click
// ends the input method's composition too, so the next key starts a new
// one instead of composing the old word on.
func TestRichTextClickEndsCompositionMac(t *testing.T) {
	var restore func()
	var ok bool
	mygo.RunOnMain(func() {
		for _, id := range []string{"com.apple.inputmethod.VietnameseIM.VietnameseTelex", "com.apple.inputmethod.VietnameseTelex"} {
			if restore, ok = darwin.TestSelectInputSource(id); ok {
				return
			}
		}
	})
	if !ok {
		t.Skip("Vietnamese Telex is not an enabled input source")
	}
	defer mygo.RunOnMain(restore)
	doc := richtext.NewDocument(richtext.NewText("Hello "))
	var frames atomic.Int32
	view := func(c *ui.Context) {
		richtext.Editor(c, doc, richtext.EditorOptions{Label: "Note"}).Fill()
		frames.Add(1)
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Composition", Width: 420, Height: 220, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if !click(w, 300, 160) {
		t.Skip("native click automation unavailable")
	}
	marked := func() (s string) {
		mygo.RunOnMain(func() { s, _ = darwin.TestMarkedText(w.NativeHandle()) })
		return s
	}
	text := func() (s string) {
		mygo.RunOnMain(func() { s = doc.Text().String() })
		return s
	}
	for _, k := range []struct {
		code  uint16
		chars string
	}{{9, "v"}, {34, "i"}, {14, "e"}, {14, "e"}} {
		pressKey(w, k.code, k.chars)
	}
	var word string
	eventually(t, "a composition", func() bool { word = marked(); return word != "" })
	click(w, 300, 160)
	eventually(t, "the composition ended by the click", func() bool { return marked() == "" && text() == "Hello "+word })
	pressKey(w, 2, "d")
	eventually(t, "a new composition", func() bool { return marked() != "" })
	if m := marked(); strings.Contains(m, word) {
		t.Fatalf("the input method composed on after the click: marked %q, document %q", m, text())
	}
}

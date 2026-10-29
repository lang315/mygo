package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// An app that shows no text input and no selectable text does not link the
// text editor: the function that prepares the editor of selectable text is
// set by Selectable, and nothing else names it. The test starts the test
// binary again, running nothing but this test, which builds text that cannot
// be selected, and asks whether the function was set at the start and after
// the frames.
func TestNoEditorIsLinkedBeforeSelectableText(t *testing.T) {
	if os.Getenv("MYGO_EDITOR_CHILD") == "1" {
		start := prepareEditorTextFn != nil
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Children(func() {
				coreText(c, "plain text")
				coreButton(c, "a button")
			})
		}, 300, 200)
		tt.Frame()
		tt.Frame()
		fmt.Printf("editor linked at start: %v, after plain frames: %v\n", start, prepareEditorTextFn != nil)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNoEditorIsLinkedBeforeSelectableText$", "-test.count=1")
	cmd.Env = append(os.Environ(), "MYGO_EDITOR_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "editor linked at start: false, after plain frames: false\n") {
		t.Errorf("the editor of selectable text is linked into an app that shows none:\n%s", out)
	}
}

// Selectable links the editor of selectable text, on a text and on a box.
func TestSelectableLinksTheEditorOfSelectableText(t *testing.T) {
	for _, onBox := range []bool{false, true} {
		prepareEditorTextFn = nil
		tt := coreNewTester(func(c *context) {
			if onBox {
				coreColumn(c).Selectable().Children(func() { coreText(c, "selectable") })
			} else {
				coreText(c, "selectable").Selectable()
			}
		}, 300, 200)
		tt.Frame()
		if prepareEditorTextFn == nil {
			t.Errorf("onBox=%v: Selectable did not link the editor", onBox)
		}
		if !tt.HasText("selectable") {
			t.Errorf("onBox=%v: the text is not shown", onBox)
		}
	}
}

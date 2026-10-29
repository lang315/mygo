package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func editorLinked() bool {
	return layoutInputFn != nil || inputHeightFn != nil || paintInputFn != nil || pressFn != nil || wantsFn != nil || textKeyFn != nil ||
		caretRectFn != nil || textSelectionKeyFn != nil || textUnitFn != nil || placeMenuCaretFn != nil
}

// editorFnsSet reports whether newEditor set all the functions.
func editorFnsSet() bool {
	return layoutInputFn != nil && inputHeightFn != nil && paintInputFn != nil && pressFn != nil && wantsFn != nil && textKeyFn != nil &&
		caretRectFn != nil && textSelectionKeyFn != nil && textUnitFn != nil && placeMenuCaretFn != nil
}

func clearEditorFns() {
	layoutInputFn, inputHeightFn, paintInputFn, pressFn, wantsFn, textKeyFn = nil, nil, nil, nil, nil, nil
	caretRectFn, textSelectionKeyFn, textUnitFn, placeMenuCaretFn = nil, nil, nil, nil
	editorOnce = sync.Once{} // linked again by the test
}

// An app with no text input and no selectable text does not link the
// editor's layout, painting and keys: the engine reaches them through
// functions that newEditor sets. The test starts the test binary again,
// running nothing but this test, which lays out and clicks a view with no
// editor, and asks whether they were set at the start and after the frames.
func TestNoEditorFunctionsAreLinkedBeforeAnEditor(t *testing.T) {
	if os.Getenv("MYGO_EDITORFNS_CHILD") == "1" {
		start := editorLinked()
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Children(func() {
				coreText(c, "no input here")
				coreButton(c, "a button")
			})
		}, 300, 200)
		tt.Frame()
		tt.Click("a button")
		tt.Key(0, KeyTab)
		fmt.Printf("editor linked at start: %v, after frames without an editor: %v\n", start, editorLinked())
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNoEditorFunctionsAreLinkedBeforeAnEditor$", "-test.count=1")
	cmd.Env = append(os.Environ(), "MYGO_EDITORFNS_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "editor linked at start: false, after frames without an editor: false\n") {
		t.Errorf("the editor is linked into an app that makes none:\n%s", out)
	}
}

// Making an editor, by building a text input, a text area or a buffer input,
// or by making text selectable, links all the functions, and the input lays
// out, paints, takes a click and keys as without the hooks.
func TestMakingAnEditorLinksItsFunctions(t *testing.T) {
	for name, view := range map[string]func(c *context, value *string){
		"input": func(c *context, value *string) { coreTextInput(c, value) },
		"area":  func(c *context, value *string) { coreTextArea(c, value) },
	} {
		t.Run(name, func(t *testing.T) {
			clearEditorFns()
			value := "abc"
			tt := coreNewTester(func(c *context) { view(c, &value) }, 300, 200)
			tt.Frame()
			if !editorFnsSet() {
				t.Fatal("making an editor did not link its functions")
			}
			tt.ClickAt(20, 12)
			tt.Key(0, KeyEnd)
			tt.Type("d")
			if value != "abcd" {
				t.Errorf("value %q after the click, End and a typed d, want abcd", value)
			}
			if _, ok := tt.TextCaret(); !ok {
				t.Error("the focused input has no caret")
			}
		})
	}
}

// Text made selectable makes its editor too, and takes a click and keys.
func TestSelectableTextLinksTheEditorFunctions(t *testing.T) {
	clearEditorFns()
	tt := coreNewTester(func(c *context) {
		coreText(c, "select me").Selectable()
	}, 300, 200)
	tt.Frame()
	if !editorFnsSet() {
		t.Fatal("selectable text did not link the editor")
	}
	tt.ClickAt(10, 10)
	tt.Key(0, KeyEnd)
}

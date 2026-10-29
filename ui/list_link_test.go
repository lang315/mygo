package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// An app that builds no List does not link the layout of lists: layout and
// scrolling reach it through functions that building a list sets. The test
// starts the test binary again, running nothing but this test, which lays out
// a view with no list, and asks whether they were set at the start and after
// the frames.
func TestNoListLayoutIsLinkedBeforeAList(t *testing.T) {
	if os.Getenv("MYGO_LIST_CHILD") == "1" {
		start := layoutListFn != nil || relayoutListFn != nil
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Children(func() {
				coreText(c, "no list here")
				coreButton(c, "a button")
			})
		}, 300, 200)
		tt.Frame()
		tt.Frame()
		fmt.Printf("list layout linked at start: %v, after frames without a list: %v\n", start, layoutListFn != nil || relayoutListFn != nil)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNoListLayoutIsLinkedBeforeAList$", "-test.count=1")
	cmd.Env = append(os.Environ(), "MYGO_LIST_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "list layout linked at start: false, after frames without a list: false\n") {
		t.Errorf("the layout of lists is linked into an app that builds none:\n%s", out)
	}
}

// Building a list links its layout, and the list is laid out and scrolled as
// without the hook.
func TestBuildingAListLinksItsLayout(t *testing.T) {
	layoutListFn, relayoutListFn = nil, nil
	var s ListState
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 100, func(i int) { coreText(c, fmt.Sprint("row ", i)) }).Grow(1)
	}, 300, 200)
	tt.Frame()
	if layoutListFn == nil || relayoutListFn == nil {
		t.Fatal("building a list did not link its layout")
	}
	if !tt.HasText("row 0") {
		t.Errorf("the first row is not shown: %q", tt.Texts())
	}
	tt.Scroll(150, 100, 0, 400)
	if tt.HasText("row 0") || !tt.HasText("row 30") {
		t.Errorf("after scrolling: %q", tt.Texts())
	}
}

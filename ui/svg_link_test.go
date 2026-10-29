package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// An app that shows no SVG does not link the SVG parser and renderer
// (internal/svg), so no package variable of ui may parse one when the program
// starts: a call in an initializer is kept by the linker, and with it the
// package. The test starts the test binary again, running nothing but this
// test, and asks how many SVGs the initializers parsed.
func TestNoSVGIsParsedWhenTheProgramStarts(t *testing.T) {
	if os.Getenv("MYGO_SVG_CHILD") == "1" {
		fmt.Printf("svgs parsed at start: %d\n", lastSVG.Load())
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNoSVGIsParsedWhenTheProgramStarts$", "-test.count=1")
	cmd.Env = append(os.Environ(), "MYGO_SVG_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "svgs parsed at start: 0\n") {
		t.Errorf("the initializers of ui parse SVGs, which links internal/svg into every app:\n%s", out)
	}
}

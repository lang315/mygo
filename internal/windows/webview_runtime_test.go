//go:build windows && (amd64 || arm64) && !mygo_nowebview

package windows

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/webview2"
)

// stubRuntimeBox replaces the message box and the opening of the download
// page, and returns what they were shown and asked.
func stubRuntimeBox(t *testing.T, response int) (boxes *[]platform.MessageBoxOptions, opened *int) {
	t.Helper()
	box, open := runtimeBox, openDownload
	t.Cleanup(func() { runtimeBox, openDownload = box, open })
	boxes, opened = new([]platform.MessageBoxOptions), new(int)
	runtimeBox = func(_ *Backend, o *platform.MessageBoxOptions) platform.MessageBoxResult {
		*boxes = append(*boxes, *o)
		return platform.MessageBoxResult{Response: response}
	}
	openDownload = func() error { *opened++; return nil }
	return boxes, opened
}

// A window that shows a page, where the machine has no WebView2 Runtime that
// works, tells the user once (the app has no console to say it on), and
// offers the download page. A WEBVIEW2_BROWSER_EXECUTABLE_FOLDER with no
// runtime in it stands for the machine without one: the runner has one.
func TestMissingRuntimeIsExplained(t *testing.T) {
	t.Setenv("WEBVIEW2_BROWSER_EXECUTABLE_FOLDER", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	if loaderProc() != nil {
		t.Skip("a loader next to the executable finds a runtime itself")
	}
	for _, tc := range []struct {
		name     string
		response int
		opened   int
	}{{"the download page is asked for", 0, 1}, {"closed", 1, 0}} {
		boxes, opened := stubRuntimeBox(t, tc.response)
		b := &Backend{name: "My App"}
		err := b.startEnvironment(&platform.WindowOptions{})
		if err == nil {
			t.Fatalf("%s: the environment started without a runtime", tc.name)
		}
		b.explainMissingRuntime(err)
		b.explainMissingRuntime(err) // once
		if len(*boxes) != 1 {
			t.Fatalf("%s: %d message boxes, want 1", tc.name, len(*boxes))
		}
		o := (*boxes)[0]
		if o.Title != "My App" || o.Type != "error" || !strings.Contains(o.Message, "My App") || !strings.Contains(o.Message, "WebView2 Runtime") {
			t.Errorf("%s: box %+v", tc.name, o)
		}
		if !strings.Contains(o.Detail, "holds no WebView2 Runtime") || !strings.Contains(o.Detail, webview2.Download) {
			t.Errorf("%s: the detail %q lacks the reason or the download link", tc.name, o.Detail)
		}
		if *opened != tc.opened {
			t.Errorf("%s: the download page opened %d times, want %d", tc.name, *opened, tc.opened)
		}
	}
}

// A failure with a runtime in place is no business of this message.
func TestRuntimeThereSaysNothing(t *testing.T) {
	if _, _, err := webview2.Find(); err != nil {
		t.Skip("no WebView2 Runtime here:", err)
	}
	boxes, opened := stubRuntimeBox(t, 0)
	(&Backend{name: "My App"}).explainMissingRuntime(errors.New("another failure"))
	if len(*boxes) != 0 || *opened != 0 {
		t.Errorf("%d boxes, %d openings for a failure with a runtime in place", len(*boxes), *opened)
	}
}

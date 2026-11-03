package main

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNativeOnlyLinksNoWebViewPlacement builds counter-native, an app of
// native UI without a web view, as mygo build builds one with nativeOnly,
// and asks the linker (-dumpdep) what it kept: neither the engine's noting
// and placing of web views, nor the window's, nor the backend's
// Surface.PlaceWebViews with the corners it cuts. The same code is in
// embed-webview, an app that puts a web view in its content, built without
// the tag: the test would not tell the two apart otherwise.
func TestNativeOnlyLinksNoWebViewPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip("building three apps")
	}
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
	default:
		t.Skip("no backend with web views")
	}
	placement := []string{
		"ui.(*engine).noteWebView",
		"ui.(*engine).placeWebViews",
		"mygo.(*Window).placeWebViews",
		"internal/" + runtime.GOOS + ".(*surface).PlaceWebViews",
		// What the backend's PlaceWebViews does, which its stand-in for a
		// build with nativeOnly (webview_none.go) does not.
		"internal/" + runtime.GOOS + map[string]string{"darwin": ".(*window).roundPane", "linux": ".shapeRegion", "windows": ".(*window).roundCorners"}[runtime.GOOS],
	}
	dumped := func(app, tags string) string {
		t.Helper()
		out := filepath.Join(t.TempDir(), exeName(runtime.GOOS, "app"))
		var log bytes.Buffer
		cmd := goCommand(filepath.Join("..", ".."), nil, "build", "-trimpath", "-tags", tags, "-ldflags", "-s -w -dumpdep", "-o", out, "./examples/"+app)
		cmd.Stdout, cmd.Stderr = &log, &log
		if err := cmd.Run(); err != nil {
			t.Fatalf("go build %s: %v\n%s", app, err, log.String())
		}
		return log.String()
	}
	with := dumped("embed-webview", "mygo_noinspector")
	for _, sym := range placement {
		if !strings.Contains(with, sym) {
			t.Errorf("an app with a web view in its content does not link %s: this test would not tell", sym)
		}
	}
	// An app that does not say nativeOnly (no mygo_nowebview) and builds no
	// web view in its content links none of the engine's and the window's
	// either: the registration, not the tag, keeps them out.
	plain := dumped("counter-native", "mygo_noinspector")
	for _, sym := range placement[:3] {
		if strings.Contains(plain, sym) {
			t.Errorf("an app without a web view in its content links %s without the tag", sym)
		}
	}
	without := dumped("counter-native", "mygo_noinspector,mygo_nowebview")
	for _, sym := range placement {
		if strings.Contains(without, sym) {
			t.Errorf("a nativeOnly app without a web view links %s", sym)
		}
	}
}

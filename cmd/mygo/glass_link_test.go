package main

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestAppWithoutGlassLinksNoNativeGlassPlacement builds counter-native, an
// app that puts no glass material in its content, and asks the linker
// (-dumpdep) what it kept: neither the engine's noting and placing of the
// system's glass, nor the window's. The same code is in liquid-glass, an app
// that uses a NativeGlassMaterial: the test would not tell the two apart
// otherwise.
func TestAppWithoutGlassLinksNoNativeGlassPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip("building two apps")
	}
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
	default:
		t.Skip("no backend")
	}
	placement := []string{
		"ui.(*engine).noteNativeGlass",
		"ui.(*engine).placeNativeGlass",
		"ui.(*windowHost).placeGlass",
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
	with := dumped("liquid-glass", "mygo_noinspector")
	for _, sym := range placement {
		if !strings.Contains(with, sym) {
			t.Errorf("an app with a glass material does not link %s: this test would not tell", sym)
		}
	}
	without := dumped("counter-native", "mygo_noinspector,mygo_nowebview")
	for _, sym := range placement {
		if strings.Contains(without, sym) {
			t.Errorf("an app without a glass material links %s", sym)
		}
	}
}

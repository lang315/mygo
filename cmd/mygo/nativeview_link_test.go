package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

// TestAppWithoutNativeViewLinksNoViewHost builds counter-native, an app that
// shows no view of a native module (ui.NativeView), and asks the linker
// (-dumpdep) what it kept: neither the surface's hosting of such a view nor
// the hook that does it. The engine holds the views it shows as any, so
// that its type does not keep platformView's methods; examples/native-module,
// which shows one, links both, or this test would not tell the two apart.
func TestAppWithoutNativeViewLinksNoViewHost(t *testing.T) {
	if testing.Short() {
		t.Skip("building six apps")
	}
	symbols := []string{
		"ui.(*platformView).SurfaceWebView",
		"internal/viewhost.Embed",
	}
	for _, goos := range []string{"darwin", "linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dumped := func(app, tags string) string {
				t.Helper()
				out := filepath.Join(t.TempDir(), exeName(goos, "app"))
				var log bytes.Buffer
				cmd := goCommand(filepath.Join("..", ".."), []string{"GOOS=" + goos, "GOARCH=amd64"}, "build", "-trimpath", "-tags", tags, "-ldflags", "-s -w -dumpdep", "-o", out, "./examples/"+app)
				cmd.Stdout, cmd.Stderr = &log, &log
				if err := cmd.Run(); err != nil {
					t.Fatalf("go build %s: %v\n%s", app, err, log.String())
				}
				return log.String()
			}
			with := dumped("native-module", "mygo_noinspector")
			for _, sym := range symbols {
				if !linksSymbol(with, sym) {
					t.Errorf("an app that shows a native view does not link %s: this test would not tell", sym)
				}
			}
			without := dumped("counter-native", "mygo_noinspector,mygo_nowebview")
			for _, sym := range symbols {
				if linksSymbol(without, sym) {
					t.Errorf("an app that shows no native view links %s", sym)
				}
			}
		})
	}
}

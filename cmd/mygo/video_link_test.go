package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

// TestAppWithoutVideoLinksNoVideoPath builds counter-native, an app that shows
// no video, for each system with a backend, and asks the linker (-dumpdep) what
// it kept: neither the engine's holding of the videos it shows, nor the
// renderers' and the CPU rasterizer's drawing of the system decoder's frames
// (NV12 planes, IOSurfaces, shared textures). examples/video is the same
// build for an app that plays one, which links all of it: the test would not
// tell the two apart otherwise.
func TestAppWithoutVideoLinksNoVideoPath(t *testing.T) {
	if testing.Short() {
		t.Skip("building six apps")
	}
	common := []string{
		"ui.(*engine).sweepVideos",
		"ui.(*engine).releaseVideos",
		"internal/raster.sampleNV12",
	}
	for _, target := range []struct {
		goos    string
		symbols []string
	}{
		{"darwin", []string{
			"internal/gpu/metal.(*Renderer).surfaceTexture",
			"internal/gpu/metal.(*Renderer).wrapSurface",
			"internal/gpu/metal.(*Renderer).releaseHeld",
			"internal/gpu/metal.loadCoreVideo",
		}},
		{"windows", []string{
			"internal/gpu/d3d11.(*Renderer).surfaceView",
			"internal/gpu/d3d11.(*Renderer).sharedView",
			"internal/gpu/d3d11.(*Renderer).sweepShared",
			"internal/gpu/d3d11.(*Renderer).releaseShared",
			"internal/gpu/d3d11.(*Renderer).planeTexture",
		}},
		{"linux", []string{
			"internal/gpu/gl.(*Renderer).planesTexture",
			"internal/gpu/gl.(*Renderer).elementTexture",
		}},
	} {
		t.Run(target.goos, func(t *testing.T) {
			dumped := func(app, tags string) string {
				t.Helper()
				out := filepath.Join(t.TempDir(), exeName(target.goos, "app"))
				var log bytes.Buffer
				cmd := goCommand(filepath.Join("..", ".."), []string{"GOOS=" + target.goos, "GOARCH=amd64"}, "build", "-trimpath", "-tags", tags, "-ldflags", "-s -w -dumpdep", "-o", out, "./examples/"+app)
				cmd.Stdout, cmd.Stderr = &log, &log
				if err := cmd.Run(); err != nil {
					t.Fatalf("go build %s: %v\n%s", app, err, log.String())
				}
				return log.String()
			}
			symbols := append(append([]string(nil), common...), target.symbols...)
			with := dumped("video", "mygo_noinspector")
			for _, sym := range symbols {
				if !linksSymbol(with, sym) {
					t.Errorf("an app that plays a video does not link %s: this test would not tell", sym)
				}
			}
			without := dumped("counter-native", "mygo_noinspector,mygo_nowebview")
			for _, sym := range symbols {
				if linksSymbol(without, sym) {
					t.Errorf("an app that shows no video links %s", sym)
				}
			}
		})
	}
}

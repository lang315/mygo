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
		if !linksSymbol(with, sym) {
			t.Errorf("an app with a glass material does not link %s: this test would not tell", sym)
		}
	}
	without := dumped("counter-native", "mygo_noinspector,mygo_nowebview")
	for _, sym := range placement {
		if linksSymbol(without, sym) {
			t.Errorf("an app without a glass material links %s", sym)
		}
	}
}

// linksSymbol reports whether the -dumpdep log has an edge into the function
// sym of the module (ui.(*engine).noteNativeGlass): a line that ends with
// "-> github.com/egoist/mygo/<sym>", or is that symbol. Its pieces
// (.argliveinfo, .stkobj, .func1) and the name of a method in the type data
// (type:.namedata.noteNativeGlass-) do not count: the linker shares
// identical argliveinfo blobs between functions, so another function's edge
// can end in a name that has the symbol's.
func linksSymbol(dump, sym string) bool {
	full := "github.com/egoist/mygo/" + sym
	for _, line := range strings.Split(dump, "\n") {
		if line == full || strings.HasSuffix(line, " -> "+full) {
			return true
		}
	}
	return false
}

func TestLinksSymbolIgnoresSharedPieces(t *testing.T) {
	const sym = "ui.(*engine).noteNativeGlass"
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"github.com/egoist/mygo/ui.(*engine).noteGlassOf -> github.com/egoist/mygo/ui.(*engine).noteNativeGlass", true},
		{"github.com/egoist/mygo/ui.(*engine).noteNativeGlass", true},
		// Windows amd64: raster's draw shares the argliveinfo of noteNativeGlass.
		{"github.com/egoist/mygo/internal/raster.(*drawer).draw -> github.com/egoist/mygo/ui.(*engine).noteNativeGlass.argliveinfo", false},
		{"github.com/egoist/mygo/ui.(*engine).noteNativeGlass -> github.com/egoist/mygo/ui.(*engine).noteNativeGlass.arginfo1", false},
		{"type:*github.com/egoist/mygo/ui.engine -> type:.namedata.noteNativeGlass-", false},
	} {
		if got := linksSymbol(tc.line, sym); got != tc.want {
			t.Errorf("linksSymbol(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

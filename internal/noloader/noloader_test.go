package noloader

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// loaders are the functions that load a native library, by the path of
// the package that has them.
var loaders = map[string][]string{
	"github.com/ebitengine/purego": {"Dlopen"},
	"syscall":                      {"LoadLibrary", "LoadDLL", "MustLoadDLL", "NewLazyDLL"},
	"golang.org/x/sys/windows":     {"LoadLibrary", "LoadLibraryEx", "LoadDLL", "MustLoadDLL", "NewLazyDLL", "NewLazySystemDLL"},
}

// loadAllowed are the directories, relative to the repository's root, whose
// files may load a native library, each with why. A directory is allowed
// alone, not its subdirectories. The system's libraries are loaded by name
// where they are used; a library of an app or a plugin is loaded and bound
// with package ffi, never with a loader of its own (AGENTS.md).
var loadAllowed = map[string]string{
	".":                             "kernel32, on Windows",
	"audio":                         "the system's sound libraries (AudioToolbox, PulseAudio, ALSA, ole32), and the C library in a test",
	"benchmarks/run":                "the system's libraries, to read a process's usage on macOS",
	"cmd/mygo":                      "kernel32 and user32 on Windows, and a test that loads a native library the CLI built",
	"ffi":                           "package ffi itself: the one loader of the native libraries of apps and plugins",
	"internal/darwin":               "macOS's frameworks, through purego",
	"internal/download":             "kernel32, to lock a file on Windows",
	"internal/e2e":                  "the system's libraries, in GUI tests",
	"internal/gpu/d3d11":            "Direct3D, DXGI and DirectComposition",
	"internal/gpu/d3d11/device":     "Direct3D",
	"internal/gpu/gl":               "OpenGL and EGL",
	"internal/gpu/metal":            "Metal and its frameworks",
	"internal/idlemem":              "the system's process APIs, to read memory",
	"internal/inputlatency":         "macOS's frameworks, to measure input latency",
	"internal/linux":                "GTK, WebKitGTK and the other system libraries, through purego",
	"internal/media/mediatest":      "CoreFoundation's run loop, in tests",
	"internal/raster":               "kernel32, on Windows",
	"internal/text":                 "the system's text engines (Core Text, Pango, DirectWrite)",
	"internal/texture":              "IOSurface and Metal, on macOS",
	"internal/webview2":             "advapi32, for the registry on Windows",
	"internal/windows":              "Win32 and WebView2's loader, through syscall",
	"native":                        "the C library's malloc and free, and CoreFoundation; the modules' libraries load through package ffi",
	"netproxy":                      "the system's proxy settings (CFNetwork, GIO, WinHTTP)",
	"plugins/game/gamepad":          "the system's controller APIs (GameController, XInput)",
	"plugins/permissionflow":        "the system's privacy frameworks, and shell32 on Windows",
	"plugins/render":                "the system's encoders (AVFoundation, GStreamer, Media Foundation)",
	"plugins/sfsymbols":             "macOS's frameworks, for SF Symbols",
	"plugins/sqlite":                "the system's SQLite, in a test that compares with it on Linux",
	"plugins/terminal/internal/pty": "the pseudo console of kernel32, on Windows",
	"plugins/terminal/internal/vt":  "libghostty-vt, loaded through purego: older than package ffi, its one exception, kept until it moves to ffi",
	"ui/video":                      "the system's video players (AVFoundation, GStreamer, Media Foundation)",
}

func TestNoNativeLoaders(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files := 0
	found := map[string]bool{} // the allowed directories some file loads in
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // removed while walked, as by another package's test
		}
		if err != nil {
			return err
		}
		if d.IsDir() {
			// .git, worktrees and build outputs (.mygo), and Bun's packages.
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		fset := token.NewFileSet()
		calls, err := scan(fset, path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		for _, c := range calls {
			if _, ok := loadAllowed[dir]; ok {
				found[dir] = true
				continue
			}
			t.Errorf("%s:%d: %s loads a native library outside the directories loadAllowed lists in internal/noloader/noloader_test.go (not their subdirectories), because a library of an app or a plugin is named in its package's mygo-plugin.json, which mygo dev, mygo build and mygo native build build and ship, and is loaded and bound with package ffi alone (AGENTS.md); use ffi.Library (Path, Symbol, Bind). A library of the system goes in loadAllowed with why",
				rel, fset.Position(c.pos).Line, c.name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 1000 {
		t.Fatalf("read %d Go files under %s, fewer than the repository has: is the root right?", files, root)
	}
	for _, dir := range slices.Sorted(maps.Keys(loadAllowed)) {
		if !found[dir] {
			t.Errorf("loadAllowed has %q, where no file loads a native library: delete it from internal/noloader/noloader_test.go", dir)
		}
	}
}

type call struct {
	pos  token.Pos
	name string
}

// scan returns a file's calls of the functions loaders lists, and of
// LoadLibrary found with NewProc, whatever the file's build constraints.
func scan(fset *token.FileSet, path string) ([]call, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	names := map[string]string{} // local name of each package in loaders
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if _, ok := loaders[p]; !ok {
			continue
		}
		name := p[strings.LastIndex(p, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = p
	}
	var out []call
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := s.X.(*ast.Ident); ok && names[x.Name] != "" {
			for _, fn := range loaders[names[x.Name]] {
				if s.Sel.Name == fn {
					out = append(out, call{c.Pos(), x.Name + "." + fn})
				}
			}
		}
		if s.Sel.Name == "NewProc" && len(c.Args) == 1 {
			if l, ok := c.Args[0].(*ast.BasicLit); ok && strings.HasPrefix(l.Value, `"LoadLibrary`) {
				out = append(out, call{c.Pos(), "NewProc(" + l.Value + ")"})
			}
		}
		return true
	})
	return out, nil
}

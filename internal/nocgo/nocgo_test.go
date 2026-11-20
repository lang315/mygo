package nocgo

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// cgoAllowed are the directories, relative to the repository's root, whose
// files may import "C", each with why.
var cgoAllowed = map[string]string{
	"internal/mobile": "iOS and Android link the Go code with Xcode's and the NDK's clang, as Go needs for them",
}

func TestNoCgo(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	files := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p != "C" {
				continue
			}
			if _, ok := cgoAllowed[filepath.ToSlash(filepath.Dir(rel))]; ok {
				continue
			}
			t.Errorf(`%s:%d: import "C" is forbidden outside internal/mobile, because desktops and browsers build with CGO_ENABLED=0; call native code through purego (internal/darwin, internal/linux), syscall (internal/windows) or package ffi for a native library`,
				rel, fset.Position(imp.Pos()).Line)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 1000 {
		t.Fatalf("read %d Go files under %s, fewer than the repository has: is the root right?", files, root)
	}
}

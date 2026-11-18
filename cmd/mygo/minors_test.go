package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// An icon that is not square is an error naming the file and its size, in
// every renderer, not a picture squashed into each size.
func TestNonSquareIconFails(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 200, 100))); err != nil {
		t.Fatal(err)
	}
	src := buf.Bytes()
	if _, err := pngToICNS(src); err == nil || !strings.Contains(err.Error(), "200x100") {
		t.Errorf("pngToICNS: %v, want an error with the size 200x100", err)
	}
	if _, err := pngToICO(src); err == nil || !strings.Contains(err.Error(), "200x100") {
		t.Errorf("pngToICO: %v, want an error with the size 200x100", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wide.png"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Config{root: dir, Icon: "wide.png"}
	if _, err := appIcon(c); err == nil || !strings.Contains(err.Error(), "wide.png") || !strings.Contains(err.Error(), "200x100") {
		t.Errorf("appIcon: %v, want an error naming wide.png and 200x100", err)
	}
}

// Two native modules whose libraries have the same file name cannot both go
// into Frameworks: the error names both modules.
func TestNativeLibraryNameCollision(t *testing.T) {
	root := t.TempDir()
	c := &Config{root: root}
	builds := []nativeBuild{
		{dir: filepath.Join(root, "a", "language"), dylib: "/x/a/libLanguage.dylib"},
		{dir: filepath.Join(root, "b", "language"), dylib: "/x/b/libLanguage.dylib"},
	}
	err := checkNativeNames(c, builds)
	if err == nil {
		t.Fatal("two modules with libLanguage.dylib: no error")
	}
	for _, want := range []string{"libLanguage.dylib", "a/language", "b/language"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	if err := checkNativeNames(c, builds[:1]); err != nil {
		t.Errorf("one module: %v", err)
	}
}

// The scan of an app that binds services from several packages gives them in
// the same order every time, so that its builds have the same bytes.
func TestBoundTypesAreSorted(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go list")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, `package main

import (
	"github.com/egoist/mygo"

	_ "bindtest/p1"
	_ "bindtest/p2"
	_ "bindtest/p3"
	_ "bindtest/p4"
	_ "bindtest/p5"
	_ "bindtest/p6"
	_ "bindtest/p7"
	_ "bindtest/p8"
)

func main() { _ = mygo.App.Run() }
`)
	for i := 1; i <= 8; i++ {
		p := filepath.Join(dir, fmt.Sprintf("p%d", i))
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatal(err)
		}
		src := fmt.Sprintf("package p%d\n\nimport \"github.com/egoist/mygo\"\n\ntype A struct{}\n\nfunc (A) M() {}\n\ntype B struct{}\n\nfunc (B) M() {}\n\n// bound in the reverse of the sorted order\nfunc init() { mygo.Bind(B{}, &A{}, A{}) }\n", i)
		if err := os.WriteFile(filepath.Join(p, "p.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	mainPkg, err := packageImportPath(c)
	if err != nil {
		t.Fatal(err)
	}
	got, reason, err := targetBoundTypes(c, runtime.GOOS, runtime.GOARCH, "", mainPkg, nil)
	if err != nil || reason != "" {
		t.Fatalf("targetBoundTypes: %v %s", err, reason)
	}
	// Each package gives A, *A, B: by package, then type, then the value before the pointer.
	if len(got) != 24 {
		t.Fatalf("found %d types, want 24: %v", len(got), got)
	}
	for i, b := range got {
		want := boundType{Pkg: fmt.Sprintf("bindtest/p%d", i/3+1), Type: []string{"A", "A", "B"}[i%3], Pointer: i%3 == 1}
		if b.Pkg != want.Pkg || b.Type != want.Type || b.Pointer != want.Pointer {
			t.Errorf("type %d is %s.%s (pointer %v), want %s.%s (pointer %v): the order must not depend on the scan", i, b.Pkg, b.Type, b.Pointer, want.Pkg, want.Type, want.Pointer)
		}
	}
}

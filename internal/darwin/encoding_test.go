//go:build darwin

package darwin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// purego derives the type encoding of a method from its Go signature
// (objc.RegisterClass), and the runtime keeps it for NSInvocation, forwarding
// and the tools that read it: a uintptr parameter becomes a bare "^", which is
// no type, and an int32 one an "i" where the C parameter is 64 bits. These
// tests keep the signatures of the methods of this package to the types whose
// encoding is right, and check that encoding in the runtime, without a window.

// methodTypes are the Go types a method may take or return, and what the
// runtime encodes them as (@encode of the C type): objects "@", selectors
// ":", a void * "^v" (unsafe.Pointer; uintptr is no pointer to purego), a
// block "@" (objc.Block), NSInteger "q", NSUInteger "Q", BOOL "B" (arm64's),
// a pointer to a struct "^{name}", and the structs by name.
var methodTypes = map[string]string{
	"id": "@", "objc.ID": "@", "objc.SEL": ":", "objc.Class": "#", "objc.Block": "@",
	"unsafe.Pointer": "^v", "bool": "B", "int": "q", "uint": "Q",
	"*nsRange": "^{nsRange}", "cgRect": "{cgRect}", "NSPoint": "{NSPoint}", "NSRect": "{NSRect}", "NSSize": "{NSSize}", "nsRange": "{nsRange}",
}

// TestMethodSignatures reads the methods registered through method() in the
// package's sources: every parameter and result has a type of methodTypes,
// and a selector has the parameters its colons say (self and _cmd besides).
func TestMethodSignatures(t *testing.T) {
	fset := token.NewFileSet()
	files, _ := filepath.Glob("*.go")
	n := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(af, func(node ast.Node) bool {
			c, ok := node.(*ast.CallExpr)
			if !ok || len(c.Args) != 2 {
				return true
			}
			if id, ok := c.Fun.(*ast.Ident); !ok || id.Name != "method" {
				return true
			}
			sel, ok := c.Args[0].(*ast.BasicLit)
			fn, isLit := c.Args[1].(*ast.FuncLit)
			if !ok || !isLit {
				return true // a function made elsewhere: its own literal is read there
			}
			n++
			where := fset.Position(c.Pos()).String()
			name := strings.Trim(sel.Value, `"`)
			params := 0
			for _, p := range fn.Type.Params.List {
				k := max(len(p.Names), 1)
				params += k
				if typ := typeString(fset, p.Type); methodTypes[typ] == "" {
					t.Errorf("%s: %s takes a %s, which the runtime does not encode right (use unsafe.Pointer for a void *, objc.Block for a block): extend methodTypes if it does", where, name, typ)
				}
			}
			if fn.Type.Results != nil {
				for _, r := range fn.Type.Results.List {
					if typ := typeString(fset, r.Type); methodTypes[typ] == "" {
						t.Errorf("%s: %s returns a %s, which the runtime does not encode right", where, name, typ)
					}
				}
			}
			if want := strings.Count(name, ":") + 2; params != want {
				t.Errorf("%s: %s has %d parameters, its selector says %d (self and _cmd included)", where, name, params, want)
			}
			return true
		})
	}
	if n < 100 {
		t.Fatalf("read only %d methods: did the registration change?", n)
	}
}

func typeString(fset *token.FileSet, e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return typeString(fset, e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(fset, e.X)
	}
	return "?"
}

// TestMethodEncodings registers a class whose methods take the types of
// methodTypes and reads back the encoding the runtime keeps.
func TestMethodEncodings(t *testing.T) {
	lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		t.Fatal(err)
	}
	var getMethod func(cls objc.Class, sel objc.SEL) uintptr
	var typeEncoding func(m uintptr) *byte
	purego.RegisterLibFunc(&getMethod, lib, "class_getInstanceMethod")
	purego.RegisterLibFunc(&typeEncoding, lib, "method_getTypeEncoding")

	cls, err := objc.RegisterClass("MyGoEncodingProbe", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("probeVoidPointer:"), Fn: func(self id, _ objc.SEL, p unsafe.Pointer) {}},
		{Cmd: sel("probeBlock:"), Fn: func(self id, _ objc.SEL, b objc.Block) {}},
		{Cmd: sel("probeInts:unsigned:flag:"), Fn: func(self id, _ objc.SEL, i int, u uint, b bool) bool { return b }},
		{Cmd: sel("probeUintptr:"), Fn: func(self id, _ objc.SEL, p uintptr) {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	enc := func(selector string) string {
		m := getMethod(cls, sel(selector))
		if m == 0 {
			t.Fatalf("no method %s", selector)
		}
		p := typeEncoding(m)
		var b []byte
		for q := unsafe.Pointer(p); *(*byte)(q) != 0; q = unsafe.Add(q, 1) {
			b = append(b, *(*byte)(q))
		}
		return string(b)
	}
	for selector, want := range map[string]string{
		"probeVoidPointer:":        "v@:^v",
		"probeBlock:":              "v@:@",
		"probeInts:unsigned:flag:": "B@:qQB",
	} {
		if got := enc(selector); got != want {
			t.Errorf("%s is encoded %q, want %q", selector, got, want)
		}
	}
	// What the signatures were: a pointer to nothing, no type.
	if got := enc("probeUintptr:"); got != "v@:^" {
		t.Errorf("a uintptr parameter is encoded %q, not the bare \"^\" TestMethodSignatures keeps out", got)
	}
}

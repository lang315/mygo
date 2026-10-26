// Package bindcheck holds the test that no function variable a package
// calls is left unbound. The native backends declare the functions of the
// libraries they load as package variables (var gtkFoo func(...)) and bind
// them with purego (mustBind(lib, &gtkFoo, "gtk_foo")). A bind lost in a
// merge, or never added, leaves a nil function: calling it crashes the
// process on a signal the Go runtime cannot handle inside a callback
// ("non-Go code set up signal handler without SA_ONSTACK flag"), and no
// test fails until a native call reaches it. The test reads the sources of
// every OS, so it runs on any host.
package bindcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// pkg is what the test learns of the files of one directory.
type pkg struct {
	declared map[string]token.Position // function variables, no initial value
	called   map[string]bool           // called as name(...)
	bound    map[string]bool           // &name, or assigned to
	optional map[string]bool           // compared with nil: a hook, not a bind
}

func newPkg() *pkg {
	return &pkg{declared: map[string]token.Position{}, called: map[string]bool{}, bound: map[string]bool{}, optional: map[string]bool{}}
}

func isNil(e ast.Expr) bool { id, ok := e.(*ast.Ident); return ok && id.Name == "nil" }

func (p *pkg) read(fset *token.FileSet, f *ast.File) {
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.VAR {
			continue
		}
		for _, s := range g.Specs {
			vs := s.(*ast.ValueSpec)
			if _, ok := vs.Type.(*ast.FuncType); !ok || len(vs.Values) != 0 {
				continue
			}
			for _, n := range vs.Names {
				p.declared[n.Name] = fset.Position(n.Pos())
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok {
				p.called[id.Name] = true
			}
		case *ast.UnaryExpr:
			if id, ok := n.X.(*ast.Ident); ok && n.Op == token.AND {
				p.bound[id.Name] = true
			}
		case *ast.AssignStmt:
			for _, l := range n.Lhs {
				if id, ok := l.(*ast.Ident); ok {
					p.bound[id.Name] = true
				}
			}
		case *ast.BinaryExpr:
			if n.Op != token.EQL && n.Op != token.NEQ {
				break
			}
			if id, ok := n.X.(*ast.Ident); ok && isNil(n.Y) {
				p.optional[id.Name] = true
			}
			if id, ok := n.Y.(*ast.Ident); ok && isNil(n.X) {
				p.optional[id.Name] = true
			}
		}
		return true
	})
}

// TestEveryFunctionVariableCalledIsBound fails for a function variable a
// package calls that nothing in the package binds or assigns, other than
// one the package compares with nil, which is an optional hook.
func TestEveryFunctionVariableCalledIsBound(t *testing.T) {
	root := filepath.Join("..", "..")
	dirs := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "testdata", "website", "packages":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			dirs[filepath.Dir(path)] = append(dirs[filepath.Dir(path)], path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for dir, files := range dirs {
		fset := token.NewFileSet()
		p := newPkg()
		for _, path := range files {
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			p.read(fset, f)
		}
		var names []string
		for name := range p.declared {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !p.called[name] {
				continue
			}
			checked++
			if !p.bound[name] && !p.optional[name] {
				t.Errorf("%s: %s is called but nothing in %s binds it", p.declared[name], name, dir)
			}
		}
	}
	if checked < 500 {
		t.Errorf("checked only %d function variables: the test no longer finds the backends' bindings", checked)
	}
}

// TestACalledFunctionVariableWithNoBindIsFound is the check of the test
// above: a variable that is called, and never bound, is reported.
func TestACalledFunctionVariableWithNoBindIsFound(t *testing.T) {
	src := `package x
var bound, lost, hook func()
func load() { mustBind(&bound) }
func use() { bound(); lost(); if hook != nil { hook() } }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := newPkg()
	p.read(fset, f)
	var unbound []string
	for name := range p.declared {
		if p.called[name] && !p.bound[name] && !p.optional[name] {
			unbound = append(unbound, name)
		}
	}
	if len(unbound) != 1 || unbound[0] != "lost" {
		t.Errorf("unbound = %v, want [lost]", unbound)
	}
}

package waitcheck

import (
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

// instead says, for each call that waits on the clock, what a test uses
// instead.
var instead = map[string]string{
	"time.Sleep":           "wait.Until or wait.For (internal/wait), or ui.Tester.Advance",
	"time.After":           "wait.For (internal/wait) on the channel",
	"time.Tick":            "wait.Until or wait.For (internal/wait)",
	"time.NewTimer":        "wait.For (internal/wait) on the channel",
	"time.NewTicker":       "wait.Until (internal/wait)",
	"time.AfterFunc":       "wait.For (internal/wait), or ui.Tester.Advance",
	"time.Now().Add":       "wait.Until or wait.Poll (internal/wait), whose guard is the only deadline",
	"time.Since":           "wait.Until (internal/wait) for a wait, counts (steps, allocations) for work",
	"time.Until":           "wait.Until (internal/wait)",
	"context.WithTimeout":  "wait.Context (internal/wait)",
	"context.WithDeadline": "wait.Context (internal/wait)",
}

type finding struct {
	pos  token.Position
	call string
}

func TestNoWallClockWaits(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string][]finding{}
	files := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// .git, worktrees and build outputs (.mygo), Bun's packages, and
			// the fixtures of tests, which go test does not run.
			n := d.Name()
			if path != root && (strings.HasPrefix(n, ".") || n == "node_modules" || n == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files++
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		calls, err := scan(path)
		if err != nil {
			return err
		}
		if len(calls) > 0 {
			found[rel] = calls
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 500 {
		t.Fatalf("read %d test files under %s, fewer than the repository has: is the root right?", files, root)
	}
	for _, rel := range slices.Sorted(maps.Keys(found)) {
		calls := found[rel]
		if len(calls) <= baseline[rel] {
			continue
		}
		for _, f := range calls {
			t.Errorf("%s:%d: %s waits on the wall clock, which fails right code on a slow runner (\"Waiting in tests\", docs/architecture.md); use %s",
				rel, f.pos.Line, f.call, instead[f.call])
		}
		if n := baseline[rel]; n > 0 {
			t.Errorf("%s: %d waits on the clock, the baseline allows %d: remove the new ones above", rel, len(calls), n)
		}
	}
	for _, rel := range slices.Sorted(maps.Keys(baseline)) {
		n := baseline[rel]
		if got := len(found[rel]); got < n {
			t.Errorf("%s: %d waits on the clock, fewer than the baseline's %d: set its line in internal/waitcheck/baseline_test.go to %q: %d, (delete the line at 0)",
				rel, got, n, rel, got)
		}
	}
}

// scan returns the calls of a test file that wait on the clock, outside
// benchmarks, which measure time.
func scan(path string) ([]finding, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	names := map[string]string{} // local name of time and context
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if p != "time" && p != "context" {
			continue
		}
		name := p
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = p
	}
	// pkgCall returns "time.Sleep" for a call of time.Sleep, whatever time's
	// local name.
	pkgCall := func(e ast.Expr) string {
		c, ok := e.(*ast.CallExpr)
		if !ok {
			return ""
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return ""
		}
		x, ok := s.X.(*ast.Ident)
		if !ok || names[x.Name] == "" {
			return ""
		}
		return names[x.Name] + "." + s.Sel.Name
	}
	var out []finding
	add := func(n ast.Node, call string) {
		out = append(out, finding{fset.Position(n.Pos()), call})
	}
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && strings.HasPrefix(fd.Name.Name, "Benchmark") {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BinaryExpr:
				// A duration compared: a deadline, or work measured by time.
				switch n.Op {
				case token.LSS, token.GTR, token.LEQ, token.GEQ:
					for _, e := range []ast.Expr{n.X, n.Y} {
						if c := pkgCall(e); c == "time.Since" || c == "time.Until" {
							add(e, c)
						}
					}
				}
			case *ast.CallExpr:
				c := pkgCall(n)
				switch c {
				case "time.Sleep", "time.After", "time.Tick", "time.NewTimer", "time.NewTicker", "time.AfterFunc",
					"context.WithTimeout", "context.WithDeadline":
					add(n, c)
				}
				// time.Now().Add(d): a deadline.
				if s, ok := n.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "Add" && pkgCall(s.X) == "time.Now" {
					add(n, "time.Now().Add")
				}
			}
			return true
		})
	}
	return out, nil
}

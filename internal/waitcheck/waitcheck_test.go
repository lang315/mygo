package waitcheck

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

// instead says, for each call that waits on the clock, what a test uses
// instead.
var instead = map[string]string{
	"time.Sleep":           "wait.Until or wait.For (internal/wait), or ui.Tester.Advance",
	"time.After":           "wait.For (internal/wait) on the channel",
	"time.Tick":            "wait.Until or wait.For (internal/wait)",
	"time.NewTimer":        "wait.For (internal/wait) on the channel",
	"time.NewTicker":       "wait.Until (internal/wait)",
	"time.AfterFunc":       "wait.For (internal/wait), or ui.Tester.Advance",
	"time.Now().After":     "wait.Until or wait.Poll (internal/wait), whose guard is the only deadline",
	"time.Now().Before":    "wait.Until or wait.Poll (internal/wait), whose guard is the only deadline",
	"time.Since":           "wait.Until (internal/wait) for a wait, counts (steps, allocations) for work",
	"time.Until":           "wait.Until (internal/wait)",
	"context.WithTimeout":  "wait.Context (internal/wait)",
	"context.WithDeadline": "wait.Context (internal/wait)",
}

// allowed are the waits on the clock that are right, as "file function
// call" (a short time.After in a check that nothing happens, which a slow
// machine can only make miss a bug), each with why.
var allowed = map[string]string{
	"plugins/updater/updater_test.go TestOnChange time.After":            "no change is told when none was made: a late call can only miss a bug",
	"plugins/updater/updater_test.go TestCheckAtLaunch time.Sleep":       "no second check starts at once: a slow machine can only miss a bug",
	"plugins/updater/native/window_test.go TestFit time.After":           "the window asks for no size once it has its own: a slow machine can only miss a bug",
	"protocol/protocol_test.go TestDownload time.After":                  "waits a while for a report of a download the window refused, which must never come",
	"updates/updates_test.go TestUpdaterWaitsForTheAppToExit time.Sleep": "checks the bundle stays in place while the app runs, which only a late look can miss",
}

type finding struct {
	rel  string // the file, relative to the repository's root
	line int
	fn   string // the function the call is in
	call string
}

// key is the finding's key in baseline and allowed.
func (f finding) key() string { return f.rel + " " + f.fn + " " + f.call }

func TestNoWallClockWaits(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string][]finding{} // by key, the allowed ones left out
	matched := map[string]bool{}    // the keys of allowed some call has
	files := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // removed while walked, as by another package's test
		}
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
		calls, err := scan(path, filepath.ToSlash(rel))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, f := range calls {
			if _, ok := allowed[f.key()]; ok {
				matched[f.key()] = true
				continue
			}
			found[f.key()] = append(found[f.key()], f)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 500 {
		t.Fatalf("read %d test files under %s, fewer than the repository has: is the root right?", files, root)
	}
	for _, k := range slices.Sorted(maps.Keys(allowed)) {
		if !matched[k] {
			t.Errorf("allowed has %q, which no call in the source matches: delete it from internal/waitcheck/waitcheck_test.go", k)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(found)) {
		calls, n := found[k], baseline[k]
		if len(calls) <= n {
			continue
		}
		for _, f := range calls {
			t.Errorf("%s:%d: %s waits on the wall clock, which fails right code on a slow runner (\"Waiting in tests\", docs/architecture.md); use %s. Only a short wait in a check that nothing happens is right: it goes in allowed in internal/waitcheck/waitcheck_test.go as %q, with why",
				f.rel, f.line, f.call, instead[f.call], k)
		}
		if n > 0 {
			t.Errorf("%q: %d waits on the clock, the baseline allows %d: remove the new ones above", k, len(calls), n)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(baseline)) {
		if got, n := len(found[k]), baseline[k]; got < n {
			t.Errorf("%q: %d waits on the clock, fewer than the baseline's %d: set its line in internal/waitcheck/baseline_test.go to %q: %d, (delete the line at 0)",
				k, got, n, k, got)
		}
	}
}

// scan returns the calls of a test file that wait on the clock, outside
// benchmarks, which measure time.
func scan(path, rel string) ([]finding, error) {
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
	for _, decl := range f.Decls {
		fn := "(package)" // a variable's initializer
		if fd, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fd.Name.Name, "Benchmark") {
				continue
			}
			fn = funcName(fd)
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			call := pkgCall(c)
			switch call {
			case "time.Sleep", "time.After", "time.Tick", "time.NewTimer", "time.NewTicker", "time.AfterFunc",
				"time.Since", "time.Until", "context.WithTimeout", "context.WithDeadline":
			default:
				// time.Now().After(deadline): a deadline checked.
				s, ok := c.Fun.(*ast.SelectorExpr)
				if !ok || (s.Sel.Name != "After" && s.Sel.Name != "Before") || pkgCall(s.X) != "time.Now" {
					return true
				}
				call = "time.Now()." + s.Sel.Name
			}
			out = append(out, finding{rel, fset.Position(c.Pos()).Line, fn, call})
			return true
		})
	}
	return out, nil
}

// funcName returns F for a function and T.M for a method.
func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	typ := fd.Recv.List[0].Type
	if s, ok := typ.(*ast.StarExpr); ok {
		typ = s.X
	}
	switch x := typ.(type) {
	case *ast.IndexExpr:
		typ = x.X
	case *ast.IndexListExpr:
		typ = x.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name + "." + fd.Name.Name
	}
	return fd.Name.Name
}

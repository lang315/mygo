package ui

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// touchedFields are the fields of state that state.pending reads (the
// "pending" group of statePlaced). A kept row
// is built again only after one of them changes (retain.go: rowKeep.dirty),
// so every write to one of them must say so: the function writing
// `x.field` has `x.touch()` among the statements of its block or of a block
// around it, in the same function or function literal.
// TestStatePendingPlacesEveryField places each field of state;
// TestEveryWriteToPendingStateTouches reads the sources.
var touchedFields = statePlaced["pending"]

// A write that only zeroes a field (false, 0, nil, [:0], delete, clear) needs
// no touch: state.pending is true for a field set, so a row clean has none to
// zero, and one that is not clean is scanned.
func zeroing(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name == "false" || x.Name == "nil"
	case *ast.BasicLit:
		return x.Value == "0"
	case *ast.SliceExpr:
		return x.Low == nil && x.High != nil && zeroing(x.High)
	}
	return false
}

// An address taken of a field (&s.releaseLater) counts as a write: what the
// alias writes is the function's, which touches. touchFree lists the functions that write those fields without touching, and
// why. A write is not touched when no kept row can hold the state, or when it
// is the build that reads the field again.
var touchFree = map[string]string{
	"engine.dropState": "the state is recycled: it holds no row",
	"engine.lookState": "a state made new holds no row; the rest only zeroes what the build sets again",
	"engine.unlook":    "the state the lookup made goes at once",
	"state.pointer":    "reads releaseLater, through its address",
}

type touchSite struct {
	pos, fn, field, base, kind string
	// touched: a statement `base.touch()` is in the block of the write or
	// in a block around it, within the function or function literal.
	touched bool
}

// touchSites returns the writes to the fields of state in the package's
// sources.
func touchSites(t *testing.T) (sites []touchSite) {
	t.Helper()
	if build.Default.GOROOT == "" { // built with -trimpath: the source importer needs it
		out, err := exec.Command("go", "env", "GOROOT").Output()
		if err != nil {
			t.Fatal(err)
		}
		build.Default.GOROOT = strings.TrimSpace(string(out))
	}
	pkg, err := build.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range pkg.GoFiles {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	info := &types.Info{Selections: map[*ast.SelectorExpr]*types.Selection{}, Types: map[ast.Expr]types.TypeAndValue{}}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check("github.com/egoist/mygo/ui", fset, files, info); err != nil {
		t.Fatal(err)
	}
	fields := map[string]bool{}
	for _, f := range touchedFields {
		fields[f] = true
	}
	isState := func(e ast.Expr) bool {
		se, ok := e.(*ast.SelectorExpr)
		if !ok || !fields[se.Sel.Name] {
			return false
		}
		sel := info.Selections[se]
		if sel == nil {
			return false
		}
		rt := sel.Recv()
		if p, ok := rt.(*types.Pointer); ok {
			rt = p.Elem()
		}
		n, ok := rt.(*types.Named)
		return ok && n.Obj().Name() == "state" && n.Obj().Pkg().Name() == "ui"
	}
	src := func(e ast.Expr) string { return types.ExprString(e) }
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			name := fd.Name.Name
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				rt := fd.Recv.List[0].Type
				if s, ok := rt.(*ast.StarExpr); ok {
					rt = s.X
				}
				if id, ok := rt.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			var stack []ast.Node
			add := func(e ast.Expr, kind string) {
				for {
					switch x := e.(type) {
					case *ast.IndexExpr:
						e = x.X
						continue
					case *ast.ParenExpr:
						e = x.X
						continue
					case *ast.StarExpr:
						e = x.X
						continue
					}
					break
				}
				if isState(e) {
					se := e.(*ast.SelectorExpr)
					sites = append(sites, touchSite{fset.Position(se.Pos()).String(), name, se.Sel.Name, src(se.X), kind, touchedIn(stack, src(se.X))})
				}
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				if n == nil {
					stack = stack[:len(stack)-1]
					return true
				}
				stack = append(stack, n)
				switch x := n.(type) {
				case *ast.AssignStmt:
					allZero := true
					for _, r := range x.Rhs {
						allZero = allZero && zeroing(r)
					}
					if allZero && x.Tok == token.ASSIGN && len(x.Lhs) == len(x.Rhs) {
						return true
					}
					for _, l := range x.Lhs {
						add(l, "assign")
						// *s = state{...} writes every field.
						if star, ok := l.(*ast.StarExpr); ok && isStatePtr(info.Types[star.X].Type) {
							sites = append(sites, touchSite{fset.Position(l.Pos()).String(), name, "(the whole state)", src(star.X), "assign", touchedIn(stack, src(star.X))})
						}
					}
				case *ast.IncDecStmt:
					add(x.X, "++")
				case *ast.UnaryExpr:
					if x.Op == token.AND {
						add(x.X, "address taken")
					}
				case *ast.RangeStmt:
					if x.Tok == token.ASSIGN {
						add(x.Key, "range")
						if x.Value != nil {
							add(x.Value, "range")
						}
					}
				}
				return true
			})
		}
	}
	return
}

// touchedIn reports whether a statement base.touch() is among the statements of
// a block on the stack of nodes around a write, up to the function or function
// literal it is in (a closure runs later: a touch outside it does not count).
// A touch in a branch of the function that the write is not in does not count
// either.
func touchedIn(stack []ast.Node, base string) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		var list []ast.Stmt
		switch n := stack[i].(type) {
		case *ast.BlockStmt:
			list = n.List
		case *ast.CaseClause:
			list = n.Body
		case *ast.CommClause:
			list = n.Body
		case *ast.FuncLit:
			return false
		}
		for _, st := range list {
			if es, ok := st.(*ast.ExprStmt); ok {
				if c, ok := es.X.(*ast.CallExpr); ok && len(c.Args) == 0 {
					if se, ok := c.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "touch" && types.ExprString(se.X) == base {
						return true
					}
				}
			}
		}
	}
	return false
}

func isStatePtr(t types.Type) bool {
	p, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	n, ok := p.Elem().(*types.Named)
	return ok && n.Obj().Name() == "state" && n.Obj().Pkg().Name() == "ui"
}

func TestEveryWriteToPendingStateTouches(t *testing.T) {
	sites := touchSites(t)
	if len(sites) < 50 {
		t.Fatalf("found %d writes: the scan lost its way", len(sites))
	}
	var bad []string
	for _, s := range sites {
		if _, free := touchFree[s.fn]; free {
			continue
		}
		if !s.touched {
			bad = append(bad, s.pos+": "+s.fn+" writes "+s.base+"."+s.field+" ("+s.kind+") without "+s.base+".touch() in its block or one around it")
		}
	}
	sort.Strings(bad)
	for _, b := range bad {
		t.Error(b)
	}
	// The allow-list names functions that write.
	seen := map[string]bool{}
	for _, s := range sites {
		seen[s.fn] = true
	}
	for fn := range touchFree {
		if !seen[fn] {
			t.Errorf("touchFree lists %s, which writes none of the fields", fn)
		}
	}
}

// touchedIn tells a touch in the block of the write, or a block around it,
// from one in a branch the write is not in, after it in a condition, or
// outside the closure that writes.
func TestTouchedInSeesTheBlock(t *testing.T) {
	const src = `package p
func branch(s *T, b bool) {
	if b {
		s.clicks++
	} else {
		s.touch()
	}
}
func closure(s *T) { f := func() { s.clicks++ }; s.touch(); f() }
func conditional(s *T, b bool) {
	s.clicks++
	if b {
		s.touch()
	}
}
func around(s *T, b bool) {
	if b {
		s.clicks++
	}
	s.touch()
}
func same(s *T) { s.clicks++; s.touch() }
func inClosure(s *T) { func() { s.clicks++; s.touch() }() }
func other(s, u *T) { s.clicks++; u.touch() }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"branch": false, "closure": false, "conditional": false, "around": true, "same": true, "inClosure": true, "other": false}
	for _, d := range f.Decls {
		fd := d.(*ast.FuncDecl)
		var stack []ast.Node
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, n)
			if inc, ok := n.(*ast.IncDecStmt); ok {
				if got := touchedIn(stack, types.ExprString(inc.X.(*ast.SelectorExpr).X)); got != want[fd.Name.Name] {
					t.Errorf("%s: touchedIn = %v, want %v", fd.Name.Name, got, want[fd.Name.Name])
				}
			}
			return true
		})
	}
}

package ui

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"testing"
)

// touchedFields are the fields of state that state.pending reads (the
// "pending" group of statePlaced). A kept row
// is built again only after one of them changes (retain.go: rowKeep.dirty),
// so every write to one of them must say so: the function writing
// `x.field` calls `x.touch()` too (same expression, same function).
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

// touchFree lists the functions that write those fields without touching, and
// why. A write is not touched when no kept row can hold the state, or when it
// is the build that reads the field again.
var touchFree = map[string]string{
	"engine.dropState": "the state is recycled: it holds no row",
	"engine.lookState": "a state made new holds no row; the rest only zeroes what the build sets again",
	"engine.unlook":    "the state the lookup made goes at once",
}

type touchSite struct {
	pos, fn, field, base, kind string
}

// touchSites returns the writes to the fields of state in the package's
// sources, and the touch calls of each function (by function name, then the
// expression touched).
func touchSites(t *testing.T) (sites []touchSite, touches map[string]map[string]bool) {
	t.Helper()
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
	touches = map[string]map[string]bool{}
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
					sites = append(sites, touchSite{fset.Position(se.Pos()).String(), name, se.Sel.Name, src(se.X), kind})
				}
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
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
							sites = append(sites, touchSite{fset.Position(l.Pos()).String(), name, "(the whole state)", src(star.X), "assign"})
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
				case *ast.CallExpr:
					if se, ok := x.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "touch" && len(x.Args) == 0 {
						if touches[name] == nil {
							touches[name] = map[string]bool{}
						}
						touches[name][src(se.X)] = true
					}
				}
				return true
			})
		}
	}
	return
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
	sites, touches := touchSites(t)
	if len(sites) < 50 {
		t.Fatalf("found %d writes: the scan lost its way", len(sites))
	}
	var bad []string
	for _, s := range sites {
		if _, free := touchFree[s.fn]; free {
			continue
		}
		if s.kind == "address taken" || !touches[s.fn][s.base] {
			bad = append(bad, s.pos+": "+s.fn+" writes "+s.base+"."+s.field+" ("+s.kind+") without "+s.base+".touch()")
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

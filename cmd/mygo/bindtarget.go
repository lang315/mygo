package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// The list of the bound types comes from running the app on this computer, so
// it is what this computer's GOOS, GOARCH, tags and conditions at run time gave
// it, where the build is for a target: a service chosen by runtime.GOOS, one
// bound only in a branch the run did not take, or a method that exists only
// here, would be missing from the table, or named in it and not compile for the
// target, and the app would panic at Bind. So the types come from the build
// itself: the packages that import mygo are type-checked as the target's go list
// presents them (go/types, the export data of their imports), every call of Bind
// and BindAs and every Service of a mygo.Plugin in them is read for the static
// type of its value, and the method set of each type is read there. A call whose
// value has no static type (an interface, a type parameter, a spread) makes the
// build keep reflection, with the place of the call. The run is a cross-check: a
// type it bound that no call names is a call the scan missed.

// mygoPath is the import path of this module's root package.
const mygoPath = "github.com/egoist/mygo"

// A listedPackage is what go list says of a package of the build.
type listedPackage struct {
	ImportPath string
	Dir        string
	Standard   bool
	GoFiles    []string
	CgoFiles   []string
	Imports    []string
	ImportMap  map[string]string
	Export     string
	Error      *struct{ Err string }
}

// targetBoundTypes returns the bound types with the methods the build for
// goos/goarch with tags (comma separated) gives them, or the reason a type
// cannot be named there (declared inside a function, not in the build), which
// keeps reflection for the build.
func targetBoundTypes(c *Config, goos, goarch, tags, mainPkg string, bound []boundType) (out []boundType, reason string, err error) {
	args := []string{"list", "-e", "-export", "-trimpath", "-deps", "-json=ImportPath,Dir,Standard,GoFiles,CgoFiles,Imports,ImportMap,Export,Error"}
	// The build with the table has the tag that brings in the tables of the
	// plugins' services (bindtable.go), whose packages the scan leaves out.
	if !slices.Contains(strings.Split(tags, ","), staticTag) {
		tags = strings.TrimPrefix(tags+","+staticTag, ",")
	}
	args = append(args, "-tags", tags, c.Main)
	var stdout, stderr bytes.Buffer
	cmd := goCommand(c.root, []string{"GOOS=" + goos, "GOARCH=" + goarch}, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("go list for %s/%s: %w\n%s", goos, goarch, err, stderr.String())
	}
	pkgs := map[string]*listedPackage{}
	exports := map[string]string{}
	for dec := json.NewDecoder(&stdout); ; {
		var p listedPackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, "", fmt.Errorf("go list for %s/%s: %w", goos, goarch, err)
		}
		pkgs[p.ImportPath] = &p
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
	}

	fset := token.NewFileSet()
	gc := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		f, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		return os.Open(f)
	})
	// typeCheck type-checks the package from its source: with the bodies of its
	// functions (and info) to read its calls, without them to read what it
	// declares. An error is an error: a package that cannot be checked cannot say
	// what it binds.
	typeCheck := func(lp *listedPackage, bodies bool, info *types.Info) (*types.Package, []*ast.File, error) {
		if len(lp.CgoFiles) > 0 {
			return nil, nil, fmt.Errorf("%s uses cgo", lp.ImportPath)
		}
		var files []*ast.File
		for _, name := range lp.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(lp.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				return nil, nil, err
			}
			files = append(files, f)
		}
		var first error
		conf := types.Config{
			Importer:         resolvingImporter{gc, lp.ImportMap},
			IgnoreFuncBodies: !bodies,
			FakeImportC:      true,
			Sizes:            types.SizesFor("gc", goarch),
			Error: func(err error) {
				if first == nil {
					first = err
				}
			},
		}
		tp, _ := conf.Check(lp.ImportPath, fset, files, info)
		if first != nil {
			return nil, nil, fmt.Errorf("cannot type-check %s: %w", lp.ImportPath, first)
		}
		return tp, files, nil
	}
	declared := map[string]*types.Package{}
	declaredIn := func(path string) (*types.Package, error) {
		if tp, ok := declared[path]; ok {
			return tp, nil
		}
		lp := pkgs[path]
		if lp == nil {
			return nil, nil
		}
		tp, _, err := typeCheck(lp, false, nil)
		if err != nil {
			return nil, err
		}
		declared[path] = tp
		return tp, nil
	}

	var found []foundType
	var unseen []string
	tabled := map[string]bool{} // the types that have a table of their own (BindMethods), as pkg.Name
	reaches := map[string]bool{}
	var reachesMygo func(path string) bool
	reachesMygo = func(path string) bool {
		if v, ok := reaches[path]; ok {
			return v
		}
		reaches[path] = false // an import cycle cannot happen; this also stops one
		lp := pkgs[path]
		if lp == nil {
			return false
		}
		for _, imp := range lp.Imports {
			if m, ok := lp.ImportMap[imp]; ok {
				imp = m
			}
			if imp == mygoPath || strings.HasSuffix(imp, "/"+mygoPath) || reachesMygo(imp) {
				reaches[path] = true
				break
			}
		}
		return reaches[path]
	}
	for _, lp := range pkgs {
		// A package that only imports one that imports mygo can still write the
		// Service of a Plugin of its type (lib.P{Service: ...}, an embedded Plugin).
		if lp.Standard || lp.ImportPath == mygoPath || !reachesMygo(lp.ImportPath) {
			continue
		}
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}, Instances: map[*ast.Ident]types.Instance{}}
		_, files, err := typeCheck(lp, true, info)
		if err != nil {
			return nil, "", err
		}
		f, u, tb := scanBinds(c.root, fset, files, lp, info)
		for _, k := range tb {
			tabled[k] = true
		}
		found = append(found, f...)
		unseen = append(unseen, u...)
	}
	if len(unseen) > 0 {
		more := ""
		if len(unseen) > 1 {
			more = fmt.Sprintf(" (and %d more)", len(unseen)-1)
		}
		return nil, unseen[0] + more, nil
	}

	seen := map[string]bool{}
	var scanned []foundType
	for _, f := range found {
		if tabled[f.b.Pkg+"."+f.b.Type] {
			continue // the type gives Bind its methods itself
		}
		if why := cannotName(mainPkg, f.b); why != "" {
			return nil, f.place + ": " + why, nil
		}
		k := fmt.Sprintf("%s.%s/%v", f.b.Pkg, f.b.Type, f.b.Pointer)
		if !seen[k] {
			seen[k] = true
			scanned = append(scanned, f)
		}
	}
	// The run is a cross-check. A type it bound that is in the build for this
	// target and that no call names is a call the scan did not see. A type of
	// another GOOS or of a branch the target does not have is not in the build.
	for _, b := range bound {
		path := b.Pkg
		if isMainPackage(mainPkg, path) {
			path = mainPkg
		}
		if tabled[path+"."+b.Type] {
			continue // the run, which has no tag, lists a service its type gives itself
		}
		if slices.ContainsFunc(scanned, func(f foundType) bool { return f.b.Pkg == path && f.b.Type == b.Type && f.b.Pointer == b.Pointer }) {
			continue
		}
		tp, err := declaredIn(path)
		if err != nil {
			return nil, "", err
		}
		if tp == nil {
			continue
		}
		if _, ok := tp.Scope().Lookup(b.Type).(*types.TypeName); ok {
			star := ""
			if b.Pointer {
				star = "*"
			}
			return nil, fmt.Sprintf("the app bound %s%s.%s when it ran, which no call of Bind or Service of a plugin in the build names", star, b.Pkg, b.Type), nil
		}
	}
	for _, f := range scanned {
		b := f.b
		ms := types.NewMethodSet(f.typ)
		var names []string
		for i := range ms.Len() {
			if obj := ms.At(i).Obj(); obj.Exported() {
				names = append(names, obj.Name())
			}
		}
		sort.Strings(names) // the order reflection lists them in
		b.Methods = slices.Clip(names)
		if b.Pointer {
			vms := types.NewMethodSet(f.typ.(*types.Pointer).Elem())
			for i := range vms.Len() {
				if obj := vms.At(i).Obj(); obj.Exported() {
					b.ValueMethods = append(b.ValueMethods, obj.Name())
				}
			}
		}
		out = append(out, b)
	}
	return out, "", nil
}

// A foundType is a service a call of the build binds: the bound type, and its
// Go type, a pointer when it is bound as one.
type foundType struct {
	place string // file:line of the call
	b     boundType
	typ   types.Type
}

// hasServiceField reports whether t is a struct (or a pointer to one) with the
// field Service of mygo.Plugin: Plugin itself, an alias of it, a type defined as
// it.
func hasServiceField(t types.Type, isMygo func(types.Object) bool) bool {
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := range st.NumFields() {
		if f := st.Field(i); f.Name() == "Service" && isMygo(f) {
			return true
		}
	}
	return false
}

// scanBinds reads, in a type-checked package, the services it binds and the
// places of the values whose type is not known statically (unseen). The types
// that have a table of their own (BindMethods) are returned as pkg.Name: their
// services give Bind their methods themselves.
//
// The forms of Go that give a Plugin its Service, and what the scan does with each
// (the field is found by its declaration in mygo, so an alias, a defined type of
// Plugin and an embedding all reach it):
//   - a literal with the key Service: the value is read; an unkeyed literal: unseen;
//   - x.Service = v, in any selector (a field, an element, through a pointer): the
//     value is read; with several values on the right and one call: unseen;
//   - for x.Service = range ...: unseen; &x.Service: unseen;
//   - a conversion to a type with that field (mygo.Plugin(x)): unseen;
//   - reflect or unsafe writing the field: not found, the release panics at start
//     with the cause (documented);
//   - Bind, BindAs and Use that are not called where they are named: unseen.
func scanBinds(root string, fset *token.FileSet, files []*ast.File, lp *listedPackage, info *types.Info) (found []foundType, unseen []string, tabled []string) {
	isMygo := func(o types.Object) bool {
		if o == nil || o.Pkg() == nil {
			return false
		}
		p := o.Pkg().Path()
		return p == mygoPath || strings.HasSuffix(p, "/"+mygoPath)
	}
	place := func(pos token.Pos) string {
		p := fset.Position(pos)
		if rel, err := filepath.Rel(root, p.Filename); err == nil && !strings.HasPrefix(rel, "..") {
			p.Filename = rel
		}
		return fmt.Sprintf("%s:%d", p.Filename, p.Line)
	}
	// isServiceField reports whether sel selects the field Service of a mygo.Plugin.
	isServiceField := func(sel *ast.SelectorExpr) bool {
		s := info.Selections[sel]
		if s == nil || s.Kind() != types.FieldVal {
			return false
		}
		v, ok := s.Obj().(*types.Var)
		return ok && v.IsField() && v.Name() == "Service" && isMygo(v)
	}
	called := map[*ast.Ident]bool{} // the functions of mygo the package calls directly
	var f []foundType
	var u []string
	var t []string // the types with a table
	value := func(arg ast.Expr, what string) {
		t := info.TypeOf(arg)
		if t == nil {
			u = append(u, fmt.Sprintf("%s: the type of the value %s is not known", place(arg.Pos()), what))
			return
		}
		t = types.Unalias(t)
		if _, ok := t.(*types.TypeParam); ok || types.IsInterface(t) {
			u = append(u, fmt.Sprintf("%s: %s binds a value of the type %s, whose type is not known until the app runs; bind a value of its type", place(arg.Pos()), what, t))
			return
		}
		ptr := false
		if p, ok := t.(*types.Pointer); ok {
			t, ptr = types.Unalias(p.Elem()), true
		}
		n, ok := t.(*types.Named)
		if !ok || n.Obj().Pkg() == nil {
			u = append(u, fmt.Sprintf("%s: %s binds a value of the type %s, which has no name", place(arg.Pos()), what, t))
			return
		}
		obj := n.Obj()
		if n.TypeArgs().Len() > 0 {
			u = append(u, fmt.Sprintf("%s: %s binds a value of the generic type %s", place(arg.Pos()), what, t))
			return
		}
		if obj.Parent() != obj.Pkg().Scope() {
			u = append(u, fmt.Sprintf("%s: %s binds a value of %s, declared inside a function", place(arg.Pos()), what, obj.Name()))
			return
		}
		var gt types.Type = n
		if ptr {
			gt = types.NewPointer(n)
		}
		f = append(f, foundType{b: boundType{Pkg: obj.Pkg().Path(), Type: obj.Name(), Pointer: ptr}, typ: gt, place: place(arg.Pos())})
	}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if tv, ok := info.Types[n.Fun]; ok && tv.IsType() && hasServiceField(tv.Type, isMygo) {
					u = append(u, fmt.Sprintf("%s: a mygo.Plugin made by a conversion, whose Service is not known until the app runs", place(n.Pos())))
					return true
				}
				var id *ast.Ident
				switch fun := n.Fun.(type) {
				case *ast.SelectorExpr:
					id = fun.Sel
				case *ast.Ident:
					id = fun
				}
				if id == nil || !isMygo(info.Uses[id]) {
					return true
				}
				called[id] = true
				switch id.Name {
				case "BindMethods":
					if len(n.Args) == 0 {
						return true
					}
					if k := tableType(n.Args[0], info); k != "" {
						t = append(t, k)
					} else {
						u = append(u, fmt.Sprintf("%s: BindMethods is called with a type the scan cannot name", place(n.Pos())))
					}
				case "Bind", "BindAs":
					args := n.Args
					if id.Name == "BindAs" && len(args) > 1 {
						args = args[1:]
					}
					if n.Ellipsis.IsValid() {
						u = append(u, fmt.Sprintf("%s: %s is called with a slice of services, whose types are not known until the app runs; call it with each", place(n.Pos()), id.Name))
						return true
					}
					for _, a := range args {
						value(a, id.Name)
					}
				}
			case *ast.AssignStmt:
				// p.Service = &S{}: the Service of a mygo.Plugin set after the literal,
				// through any selector (embedded, nested, a pointer, an element).
				for i, l := range n.Lhs {
					sel, ok := ast.Unparen(l).(*ast.SelectorExpr)
					if !ok || !isServiceField(sel) {
						continue
					}
					if len(n.Lhs) != len(n.Rhs) {
						u = append(u, fmt.Sprintf("%s: the Service of a mygo.Plugin is assigned from a call with several results, whose type is not known until the app runs", place(n.Pos())))
						continue
					}
					value(n.Rhs[i], "the plugin "+lp.ImportPath)
				}
			case *ast.UnaryExpr:
				// &p.Service lets anything write it.
				if sel, ok := ast.Unparen(n.X).(*ast.SelectorExpr); ok && n.Op == token.AND && isServiceField(sel) {
					u = append(u, fmt.Sprintf("%s: the address of the Service of a mygo.Plugin is taken, and what writes it is not known until the app runs", place(n.Pos())))
				}
			case *ast.RangeStmt:
				for _, e := range []ast.Expr{n.Key, n.Value} {
					if sel, ok := ast.Unparen(e).(*ast.SelectorExpr); ok && n.Tok == token.ASSIGN && isServiceField(sel) {
						u = append(u, fmt.Sprintf("%s: the Service of a mygo.Plugin is the variable of a range, whose type is not known until the app runs", place(n.Pos())))
					}
				}
			case *ast.CompositeLit:
				if !hasServiceField(info.TypeOf(n), isMygo) {
					return true
				}
				for _, e := range n.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					if !ok {
						u = append(u, fmt.Sprintf("%s: a mygo.Plugin with unkeyed fields", place(n.Pos())))
						return true
					}
					if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Service" && isMygo(info.Uses[k]) {
						value(kv.Value, "the plugin "+lp.ImportPath)
					}
				}
			}
			return true
		})
	}
	// Bind, BindAs or Use that is not called where it is named (reg := mygo.Bind)
	// binds what the scan cannot follow.
	for id, obj := range info.Uses {
		if fn, ok := obj.(*types.Func); ok && isMygo(fn) && !called[id] {
			switch fn.Name() {
			case "Bind", "BindAs", "Use":
				u = append(u, fmt.Sprintf("%s: %s is used as a value, and what it binds is not known until the app runs; call it", place(id.Pos()), fn.Name()))
			}
		}
	}
	slices.Sort(u) // the map above has no order
	return f, u, t
}

// tableType names the type a call of BindMethods gives a table to (its first
// argument is reflect.TypeFor[T]() or reflect.TypeOf(x)), as pkg.Name, or "".
func tableType(arg ast.Expr, info *types.Info) string {
	call, ok := ast.Unparen(arg).(*ast.CallExpr)
	if !ok {
		return ""
	}
	var t types.Type
	var fun *ast.Ident
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.IndexExpr:
		fun = selName(f.X)
	default:
		fun = selName(f)
	}
	if fun != nil {
		if inst, ok := info.Instances[fun]; ok && inst.TypeArgs.Len() == 1 {
			t = inst.TypeArgs.At(0)
		}
	}
	if t == nil && len(call.Args) == 1 {
		t = info.TypeOf(call.Args[0])
	}
	if t == nil {
		return ""
	}
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := types.Unalias(t).(*types.Named)
	if !ok || n.Obj().Pkg() == nil {
		return ""
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}

func selName(e ast.Expr) *ast.Ident {
	switch e := ast.Unparen(e).(type) {
	case *ast.SelectorExpr:
		return e.Sel
	case *ast.Ident:
		return e
	}
	return nil
}

// resolvingImporter imports by the path a package of the build means: the
// vendored copy, say, that go list's ImportMap names.
type resolvingImporter struct {
	types.Importer
	importMap map[string]string
}

func (r resolvingImporter) Import(path string) (*types.Package, error) {
	if m, ok := r.importMap[path]; ok {
		path = m
	}
	return r.Importer.Import(path)
}

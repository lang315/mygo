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
)

// The list of the bound types comes from running the app on this computer, so
// it has the methods this computer's GOOS, GOARCH and tags give them, where the
// build is for a target. A method that exists only here would be named in the
// table and the build for the target would not compile; a method only the target has
// would be missing from it, and the app would panic at Bind. So the names of the
// types come from the run, and their methods from the target: the packages of the
// build are type-checked as the target's go list presents them (go/types, the
// export data of their imports), and the method set of each type is read there.

// A listedPackage is what go list says of a package of the build.
type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	CgoFiles   []string
	ImportMap  map[string]string
	Export     string
	Error      *struct{ Err string }
}

// targetBoundTypes returns the bound types with the methods the build for
// goos/goarch with tags (comma separated) gives them, or the reason a type
// cannot be named there (declared inside a function, not in the build), which
// keeps reflection for the build.
func targetBoundTypes(c *Config, goos, goarch, tags, mainPkg string, bound []boundType) (out []boundType, reason string, err error) {
	args := []string{"list", "-e", "-export", "-deps", "-json=ImportPath,Dir,GoFiles,CgoFiles,ImportMap,Export,Error"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, c.Main)
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
	checked := map[string]*types.Package{}
	check := func(path string) (*types.Package, error) {
		if p, ok := checked[path]; ok {
			return p, nil
		}
		lp := pkgs[path]
		if lp == nil {
			return nil, fmt.Errorf("%s is not in the build for %s/%s", path, goos, goarch)
		}
		if len(lp.CgoFiles) > 0 {
			return nil, fmt.Errorf("%s uses cgo", path)
		}
		var files []*ast.File
		for _, name := range lp.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(lp.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		conf := types.Config{
			Importer:         resolvingImporter{gc, lp.ImportMap},
			IgnoreFuncBodies: true,
			FakeImportC:      true,
			Sizes:            types.SizesFor("gc", goarch),
			Error:            func(error) {}, // only the declarations are read
		}
		tp, _ := conf.Check(path, fset, files, nil)
		if tp == nil {
			return nil, fmt.Errorf("cannot type-check %s", path)
		}
		checked[path] = tp
		return tp, nil
	}

	for _, b := range bound {
		path := b.Pkg
		if isMainPackage(mainPkg, path) {
			path = mainPkg
		}
		tp, err := check(path)
		if err != nil {
			return nil, "", err
		}
		tn, ok := tp.Scope().Lookup(b.Type).(*types.TypeName)
		if !ok {
			return nil, fmt.Sprintf("%s is not declared at the top of a package for %s/%s (inside a function?) and cannot be named", b.Type, goos, goarch), nil
		}
		t := tn.Type()
		if b.Pointer {
			t = types.NewPointer(t)
		}
		ms := types.NewMethodSet(t)
		var names []string
		for i := range ms.Len() {
			if obj := ms.At(i).Obj(); obj.Exported() {
				names = append(names, obj.Name())
			}
		}
		sort.Strings(names) // the order reflection lists them in
		b.Methods = slices.Clip(names)
		b.ValueMethods = nil
		if b.Pointer {
			vms := types.NewMethodSet(tn.Type())
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

package main

import (
	"bytes"
	"context"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A boundType is a service the app bound, as generate mode lists it
// (MYGO_GENERATE_BINDINGS): where its type is, whether it was bound as a
// pointer, and its exported methods in the order reflection lists them.
type boundType struct {
	Pkg     string   `json:"pkg"`
	Type    string   `json:"type"`
	Pointer bool     `json:"pointer"`
	Methods []string `json:"methods"`
}

// bindGenFile is the name of the file in the overlay that gives Bind the
// methods of the services without reflection (see bind_methods.go in package mygo).
const bindGenFile = "0mygo_bind_gen.go"

// staticTag is the build tag under which Bind takes the methods of a service
// from the file bindingsGo writes, and not from reflection (package mygo).
const staticTag = "mygo_static"

// readBoundTypes reads the list generate mode wrote; a missing file is an
// app that binds nothing of its own.
func readBoundTypes(path string) ([]boundType, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []boundType
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// bindFiles returns the overlay entry that gives Bind the methods of the
// services the app bound, writing the generated source into work, or the reason
// it cannot (see bindingsGo): with a file the build gets the tag mygo_static,
// without one it keeps reflection. mainPkg is the import path of the main
// package in pkg. It returns neither when generate mode did not run (an app
// without a frontend) or the app binds nothing of its own.
func bindFiles(c *Config, pkg, work, mainPkg string) (files map[string]string, reason string, err error) {
	files = map[string]string{}
	if !c.boundRead {
		return files, "", nil
	}
	src, reason := bindingsGo(mainPkg, c.bound)
	if src == nil {
		return files, reason, nil
	}
	if _, err := os.Lstat(filepath.Join(pkg, bindGenFile)); err == nil {
		return nil, "", fmt.Errorf("%s is in the way of giving Bind the methods of the services; rename it", filepath.Join(pkg, bindGenFile))
	}
	gen := filepath.Join(work, bindGenFile)
	if err := os.WriteFile(gen, src, 0o644); err != nil {
		return nil, "", err
	}
	files[filepath.Join(pkg, bindGenFile)] = gen
	return files, "", nil
}

// generateTimeout is how long a release build lets an app run in generate mode
// to list what it binds. Generate mode returns from App.Run before any window,
// but the code of the app before it runs.
const generateTimeout = 2 * time.Minute

// listBoundTypes lists the services an app with no TypeScript client to write
// (no frontend: an app like hello, which calls its Go from a page of its own)
// binds, as writeClient does for one that has a client. It builds the app for
// this computer, and runs it in generate mode, once, only if the build calls
// Bind (bindsServices): the code of an app that binds nothing, most apps of native
// UI, does not run at build time, and its list is empty. An error leaves no
// list: the caller says so and the build goes on with reflection, which
// does not need one.
func listBoundTypes(ctx context.Context, c *Config) error {
	if c.Bindings != "" || c.boundRead {
		return nil // writeClient listed them, or there is nothing to run for
	}
	t := con.start("Listing the services the app binds")
	bin := tempBinary(c.executableName())
	defer os.Remove(bin)
	err := goBuild(ctx, c, t, bin, nil, generatorTags(c.root)...)
	binds := false
	if err == nil {
		binds, err = bindsServices(bin)
	}
	if err == nil && !binds {
		c.boundRead = true // nothing bound: nothing to run
		t.stop()
		return nil
	}
	if err == nil {
		t.set("running the app in generate mode")
		var dir string
		if dir, err = os.MkdirTemp("", "mygo-bound-"); err == nil {
			defer os.RemoveAll(dir)
			run, cancel := context.WithTimeout(ctx, generateTimeout)
			defer cancel()
			err = runInGenerateMode(run, c, bin, filepath.Join(dir, "mygo.ts"))
			if err != nil && run.Err() == context.DeadlineExceeded {
				err = fmt.Errorf("it did not return from generate mode in %v", generateTimeout)
			}
		}
	}
	t.stop()
	return err
}

// bindSymbol is the function that Bind, BindAs and Use register a service
// with. A program that never calls any of them does not have it: the linker
// leaves out what nothing reaches.
const bindSymbol = "github.com/egoist/mygo.bind"

// bindsServices reports whether the Go program at path, an executable of this
// computer that is not stripped, calls Bind (or BindAs, or Use), from its
// symbols, without running it.
func bindsServices(path string) (bool, error) {
	names, err := symbolNames(path)
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == bindSymbol || strings.HasSuffix(n, "/"+bindSymbol) { // a vendored MyGo
			return true, nil
		}
	}
	return false, nil
}

// symbolNames returns the names in the symbol table of an ELF, Mach-O or PE
// executable.
func symbolNames(path string) ([]string, error) {
	var names []string
	if f, err := elf.Open(path); err == nil {
		defer f.Close()
		syms, err := f.Symbols()
		if err != nil {
			return nil, fmt.Errorf("reading the symbols of %s: %w", filepath.Base(path), err)
		}
		for _, s := range syms {
			names = append(names, s.Name)
		}
		return names, nil
	}
	if f, err := macho.Open(path); err == nil {
		defer f.Close()
		if f.Symtab == nil {
			return nil, fmt.Errorf("%s has no symbol table", filepath.Base(path))
		}
		for _, s := range f.Symtab.Syms {
			names = append(names, strings.TrimPrefix(s.Name, "_"))
		}
		return names, nil
	}
	if f, err := pe.Open(path); err == nil {
		defer f.Close()
		if len(f.Symbols) == 0 {
			return nil, fmt.Errorf("%s has no symbol table", filepath.Base(path))
		}
		for _, s := range f.Symbols {
			names = append(names, s.Name)
		}
		return names, nil
	}
	return nil, fmt.Errorf("%s is not an ELF, Mach-O or PE executable", filepath.Base(path))
}

// bindTable gives a release build the table of the methods of the services
// the app bound, and the tag that makes Bind use it, when every service can be
// named; otherwise it says why Bind keeps reflection, once.
func bindTable(c *Config, opts *buildOptions) error {
	if !c.boundRead || len(c.bound) == 0 {
		return nil
	}
	mainPkg, err := packageImportPath(c)
	if err != nil {
		return err
	}
	files, reason, err := bindFiles(c, opts.pkg, opts.work, mainPkg)
	if err != nil {
		return err
	}
	if reason != "" {
		logf("Bind keeps reflection (%s), about 1.4 MB more", reason)
	}
	if len(files) > 0 {
		opts.overlay = mergeFiles(opts.overlay, files)
		opts.static = true
	}
	return nil
}

// packageImportPath returns the import path of the app's main package.
func packageImportPath(c *Config) (string, error) {
	var out bytes.Buffer
	cmd := goCommand(c.root, nil, "list", "-f", "{{.ImportPath}}", c.Main)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go list %s: %w", c.Main, err)
	}
	return strings.TrimSpace(out.String()), nil
}

// withTag returns the go build flags with tag among the -tags.
func withTag(flags []string, tag string) []string {
	flags = slices.Clone(flags)
	for i, f := range flags {
		if f == "-tags" && i+1 < len(flags) {
			flags[i+1] += "," + tag
			return flags
		}
	}
	return append(flags, "-tags", tag)
}

// bindingsGo returns the Go file, for the main package of the app, that gives
// Bind the methods of every bound service as method expressions, which the
// linker follows without keeping every method of the program. mainPkg is
// the import path of the main package.
//
// It returns no file and a reason when some service cannot be named from the main
// package (an unexported type of another package, a generic type, an internal
// package the main package cannot import): a build is all or nothing, so the
// tag mygo_static that goes with the file is left off and Bind uses reflection.
// With nothing bound it returns neither. The file never fails to compile for a
// type that was not checked.
func bindingsGo(mainPkg string, bound []boundType) (src []byte, reason string) {
	if len(bound) == 0 {
		return nil, ""
	}
	for _, b := range bound {
		if why := cannotName(mainPkg, b); why != "" {
			return nil, why
		}
	}
	aliases := map[string]string{} // import path → alias
	var paths []string
	used := map[string]bool{"mygo_reflect": true, "mygo_mygo": true}
	for _, b := range bound {
		if isMainPackage(mainPkg, b.Pkg) {
			continue
		}
		if _, ok := aliases[b.Pkg]; ok {
			continue
		}
		alias := "mygo_" + identifierOf(b.Pkg[strings.LastIndex(b.Pkg, "/")+1:])
		for n := 2; used[alias]; n++ {
			alias = "mygo_" + identifierOf(b.Pkg[strings.LastIndex(b.Pkg, "/")+1:]) + strconv.Itoa(n)
		}
		used[alias] = true
		aliases[b.Pkg] = alias
		paths = append(paths, b.Pkg)
	}
	sort.Strings(paths)

	var out bytes.Buffer
	out.WriteString("// Code generated by mygo build. DO NOT EDIT.\n\n//go:build mygo_static\n\npackage main\n\nimport (\n")
	out.WriteString("\tmygo_reflect \"reflect\"\n\n\tmygo_mygo \"github.com/egoist/mygo\"\n")
	for _, p := range paths {
		fmt.Fprintf(&out, "\t%s %q\n", aliases[p], p)
	}
	out.WriteString(")\n\nfunc init() {\n")
	seen := map[string]bool{}
	for _, b := range bound {
		typ := b.Type
		if !isMainPackage(mainPkg, b.Pkg) {
			typ = aliases[b.Pkg] + "." + b.Type
		}
		key := fmt.Sprintf("%v %s", b.Pointer, typ)
		if seen[key] {
			continue // the same type bound under two names
		}
		seen[key] = true
		recv, expr := typ, typ
		if b.Pointer {
			recv, expr = "*"+typ, "(*"+typ+")"
		}
		fmt.Fprintf(&out, "\tmygo_mygo.BindMethods(mygo_reflect.TypeFor[%s]()", recv)
		for _, m := range b.Methods {
			fmt.Fprintf(&out, ",\n\t\tmygo_mygo.BoundMethod{Name: %q, Func: %s.%s}", m, expr, m)
		}
		out.WriteString(")\n")
	}
	out.WriteString("}\n")
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		return nil, fmt.Sprintf("the generated file does not parse: %v", err)
	}
	return formatted, ""
}

// isMainPackage reports whether pkg is the main package: reflect names it
// "main" in a program, whatever its import path.
func isMainPackage(mainPkg, pkg string) bool { return pkg == "main" || pkg == mainPkg }

// cannotName returns why the generated file, in the main package, cannot name
// the methods of b, or "".
func cannotName(mainPkg string, b boundType) string {
	what := b.Pkg + "." + b.Type
	if isMainPackage(mainPkg, b.Pkg) {
		what = b.Type
	}
	if b.Type == "" || b.Pkg == "" {
		return "a bound service has an unnamed type"
	}
	if strings.Contains(b.Type, "[") {
		return what + " is a generic type"
	}
	if !token.IsIdentifier(b.Type) {
		return what + " is not a Go identifier"
	}
	for _, m := range b.Methods {
		if m == "" || !ast.IsExported(m) || !token.IsIdentifier(m) {
			return what + " has a method named " + strconv.Quote(m) + ", which cannot be named"
		}
	}
	if isMainPackage(mainPkg, b.Pkg) {
		return ""
	}
	if !ast.IsExported(b.Type) {
		return what + " is unexported and cannot be named from package main"
	}
	if !validImportPath(b.Pkg) {
		return b.Pkg + " is not an import path the file can use"
	}
	// An internal package can be imported from the tree of its parent only.
	if i := slices.Index(strings.Split(b.Pkg, "/"), "internal"); i >= 0 {
		parent := strings.Join(strings.Split(b.Pkg, "/")[:i], "/")
		if parent == "" || !(mainPkg == parent || strings.HasPrefix(mainPkg, parent+"/")) {
			return what + " is in an internal package that package main cannot import"
		}
	}
	return ""
}

// validImportPath reports whether p is a path a Go file can import: slash
// separated elements of letters, digits and . _ ~ + - , none empty or "..".
func validImportPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.HasSuffix(p, "/") {
		return false
	}
	for _, elem := range strings.Split(p, "/") {
		if elem == "" || elem == "." || elem == ".." {
			return false
		}
		for _, r := range elem {
			switch {
			case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
			case strings.ContainsRune("._~+-", r):
			default:
				return false
			}
		}
	}
	return true
}

// identifierOf turns the last element of an import path into an identifier.
func identifierOf(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "pkg"
	}
	return b.String()
}

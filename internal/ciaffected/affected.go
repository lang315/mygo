package main

import (
	"path"
	"slices"
	"strings"
)

// Package is what `go list -json` says of a package.
type Package struct {
	ImportPath   string
	Dir          string // relative to the module's root, "." for its package
	Deps         []string
	TestImports  []string
	XTestImports []string
	// The files the package embeds, relative to its directory.
	EmbedFiles, TestEmbedFiles, XTestEmbedFiles []string
}

// Affected returns the import paths of the packages the changed files
// (slash-separated, relative to the module's root) affect, in the order of
// pkgs, or full when they all are.
func Affected(changed []string, pkgs []Package) (affected []string, full bool) {
	byDir := map[string]*Package{}
	byPath := map[string]*Package{}
	for i := range pkgs {
		byDir[pkgs[i].Dir] = &pkgs[i]
		byPath[pkgs[i].ImportPath] = &pkgs[i]
	}
	self := map[string]bool{}     // changed packages whose dependents are not
	changed2 := map[string]bool{} // changed packages and their dependents
	for _, f := range changed {
		p, selfOnly, ok := owner(f, byDir, pkgs)
		switch {
		case !ok:
			return nil, true
		case p == nil:
		case selfOnly:
			self[p.ImportPath] = true
		default:
			changed2[p.ImportPath] = true
		}
	}
	for _, p := range pkgs {
		if self[p.ImportPath] || changed2[p.ImportPath] ||
			slices.ContainsFunc(dependencies(&p, byPath), func(d string) bool { return changed2[d] }) {
			affected = append(affected, p.ImportPath)
		}
	}
	return affected, false
}

// dependencies are the packages p's build and its tests need: its imports,
// transitively, and its tests' imports, with theirs.
func dependencies(p *Package, byPath map[string]*Package) []string {
	deps := slices.Clone(p.Deps)
	for _, t := range slices.Concat(p.TestImports, p.XTestImports) {
		deps = append(deps, t)
		if tp := byPath[t]; tp != nil {
			deps = append(deps, tp.Deps...)
		}
	}
	return deps
}

// owner returns the package a changed file belongs to: nil when the file
// changes nothing that is tested, and ok false when no package owns it, so
// that every package is affected. A test file affects its package alone (selfOnly).
func owner(file string, byDir map[string]*Package, pkgs []Package) (p *Package, selfOnly, ok bool) {
	switch {
	case strings.HasPrefix(file, "docs/"), strings.HasPrefix(file, "website/"):
		return nil, false, true
	case file == "go.mod", file == "go.sum", strings.HasPrefix(file, ".github/"),
		strings.HasPrefix(file, "packages/"), strings.HasPrefix(file, "scripts/"):
		return nil, false, false
	}
	dir := path.Dir(file)
	if strings.HasSuffix(file, ".go") {
		// A directory without a package here, such as another system's, or
		// deleted, changes nothing for this system: what imported it
		// changed too.
		return byDir[dir], strings.HasSuffix(file, "_test.go"), true
	}
	if strings.HasPrefix(file, "plugins/") {
		return nil, false, false // the pinned natives and manifests are everybody's
	}
	// A file a package embeds belongs to it, in whichever directory it is.
	for i := range pkgs {
		q := &pkgs[i]
		rel, found := strings.CutPrefix(file, q.Dir+"/")
		if !found {
			continue
		}
		if slices.Contains(q.EmbedFiles, rel) || slices.Contains(q.TestEmbedFiles, rel) || slices.Contains(q.XTestEmbedFiles, rel) {
			return q, false, true
		}
	}
	// Else the nearest package above it, but not the module's: the files of
	// the root directory itself are everybody's.
	for d := dir; d != "."; d = path.Dir(d) {
		if q := byDir[d]; q != nil {
			return q, false, true
		}
	}
	return nil, false, false
}

package main

import (
	"slices"
	"testing"
)

// The graph: root imports a; b imports a; c imports b; d imports nothing and
// has a test importing c (as an external test of its own); e imports nothing.
var graph = []Package{
	{ImportPath: "m", Dir: "."},
	{ImportPath: "m/a", Dir: "a"},
	{ImportPath: "m/b", Dir: "b", Deps: []string{"m/a"}},
	{ImportPath: "m/c", Dir: "c", Deps: []string{"m/b", "m/a"}, EmbedFiles: []string{"tmpl/x.txt"}},
	{ImportPath: "m/d", Dir: "d", XTestImports: []string{"m/c"}},
	{ImportPath: "m/e", Dir: "e"},
	{ImportPath: "m/plugins/p", Dir: "plugins/p"},
}

func TestAffected(t *testing.T) {
	for _, tc := range []struct {
		name    string
		changed []string
		want    []string
		full    bool
	}{
		{"nothing", nil, nil, false},
		{"a test file affects its package alone", []string{"a/a_test.go"}, []string{"m/a"}, false},
		{"a Go file affects its dependents", []string{"b/b.go"}, []string{"m/b", "m/c", "m/d"}, false},
		{"and the packages whose tests import them", []string{"c/c.go"}, []string{"m/c", "m/d"}, false},
		{"a leaf", []string{"e/e.go"}, []string{"m/e"}, false},
		{"a file embedded by a package, in another directory of it", []string{"c/tmpl/x.txt"}, []string{"m/c", "m/d"}, false},
		{"testdata of a package", []string{"e/testdata/f.json"}, []string{"m/e"}, false},
		{"docs and the website", []string{"docs/a.md", "website/x.ts"}, nil, false},
		{"a Go file of a directory without a package here", []string{"a/gone/x_windows.go"}, nil, false},
		{"go.mod", []string{"go.mod"}, nil, true},
		{"go.sum", []string{"e/e_test.go", "go.sum"}, nil, true},
		{"a workflow", []string{".github/workflows/ci.yml"}, nil, true},
		{"the JavaScript packages", []string{"packages/bridge/src/x.ts"}, nil, true},
		{"scripts", []string{"scripts/version.ts"}, nil, true},
		{"a non-Go file of a plugin", []string{"plugins/p/mygo-plugin.json"}, nil, true},
		{"a Go file of a plugin", []string{"plugins/p/p.go"}, []string{"m/plugins/p"}, false},
		{"a file of the root directory", []string{"package.json"}, nil, true},
		{"a file under no package", []string{"tools/x/y.txt"}, nil, true},
		{"the root package's Go file affects its dependents", []string{"mygo.go"}, []string{"m"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, full := Affected(tc.changed, graph)
			if full != tc.full || !slices.Equal(got, tc.want) {
				t.Errorf("Affected(%q) = %q, full %v; want %q, full %v", tc.changed, got, full, tc.want, tc.full)
			}
		})
	}
}

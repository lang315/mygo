package main

import (
	"slices"
	"testing"
)

// The graph: root imports a; b imports a; c imports b; d imports nothing and
// has a test importing c (as an external test of its own); e imports nothing;
// the command line tool, ui and the two tests that always run are apart.
// internal/web has no package here, as on any system but the browsers'.
var graph = []Package{
	{ImportPath: "m", Dir: "."},
	{ImportPath: "m/a", Dir: "a"},
	{ImportPath: "m/b", Dir: "b", Deps: []string{"m/a"}},
	{ImportPath: "m/c", Dir: "c", Deps: []string{"m/b", "m/a"}, EmbedFiles: []string{"tmpl/x.txt"}},
	{ImportPath: "m/d", Dir: "d", XTestImports: []string{"m/c"}},
	{ImportPath: "m/e", Dir: "e"},
	{ImportPath: "m/plugins/p", Dir: "plugins/p"},
	{ImportPath: "m/website/docs-demo", Dir: "website/docs-demo"},
	{ImportPath: "m/ui", Dir: "ui", Deps: []string{"m/e"}},
	{ImportPath: "m/cmd/mygo", Dir: "cmd/mygo"},
	{ImportPath: "m/internal/skillsdocs", Dir: "internal/skillsdocs"},
	{ImportPath: "m/internal/twclass", Dir: "internal/twclass"},
}

// tested are the packages that run whatever changed.
var tested = []string{"m/internal/skillsdocs", "m/internal/twclass"}

func TestAffected(t *testing.T) {
	for _, tc := range []struct {
		name    string
		changed []string
		want    []string // and tested
		full    bool
	}{
		{"nothing", nil, nil, false},
		{"a test file affects its package alone", []string{"a/a_test.go"}, []string{"m/a"}, false},
		{"a Go file affects its dependents", []string{"b/b.go"}, []string{"m/b", "m/c", "m/d"}, false},
		{"and the packages whose tests import them", []string{"c/c.go"}, []string{"m/c", "m/d"}, false},
		{"a leaf ui and so the command build", []string{"e/e.go"}, []string{"m/e", "m/ui", "m/cmd/mygo"}, false},
		{"a file embedded by a package, in another directory of it", []string{"c/tmpl/x.txt"}, []string{"m/c", "m/d"}, false},
		{"testdata of a package", []string{"e/testdata/f.json"}, []string{"m/e", "m/ui", "m/cmd/mygo"}, false},
		{"a Go file in the testdata of a package affects it alone", []string{"a/testdata/app/main.go"}, []string{"m/a"}, false},
		{"and of the root package", []string{"testdata/x.go"}, []string{"m"}, false},
		{"docs", []string{"docs/a.md"}, nil, false},
		{"the website's own package", []string{"website/docs-demo/demo.go"}, []string{"m/website/docs-demo"}, false},
		{"the website's pages", []string{"website/src/pages/index.astro", "website/package.json"}, nil, false},
		{"a Go file of a directory without a package here goes to the command", []string{"internal/web/x_js.go"}, []string{"m/cmd/mygo"}, false},
		{"the command builds the root package", []string{"mygo.go"}, []string{"m", "m/cmd/mygo"}, false},
		{"and ui, with what ui needs", []string{"e/e.go", "ui/ui.go"}, []string{"m/e", "m/ui", "m/cmd/mygo"}, false},
		{"go.mod", []string{"go.mod"}, nil, true},
		{"go.sum", []string{"e/e_test.go", "go.sum"}, nil, true},
		{"a workflow", []string{".github/workflows/ci.yml"}, nil, true},
		{"the JavaScript packages", []string{"packages/bridge/src/x.ts"}, nil, true},
		{"scripts", []string{"scripts/version.ts"}, nil, true},
		{"a non-Go file of a plugin", []string{"plugins/p/mygo-plugin.json"}, nil, true},
		{"a Go file of a plugin", []string{"plugins/p/p.go"}, []string{"m/plugins/p"}, false},
		{"a file of the root directory", []string{"package.json"}, nil, true},
		{"a file under no package", []string{"tools/x/y.txt"}, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, full := Affected(tc.changed, graph)
			want := tc.want
			if !tc.full {
				want = nil
				for _, p := range graph {
					if slices.Contains(tc.want, p.ImportPath) || slices.Contains(tested, p.ImportPath) {
						want = append(want, p.ImportPath)
					}
				}
			}
			if full != tc.full || !slices.Equal(got, want) {
				t.Errorf("Affected(%q) = %q, full %v; want %q, full %v", tc.changed, got, full, want, tc.full)
			}
		})
	}
}

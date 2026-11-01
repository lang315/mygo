package main

import (
	"bytes"
	"flag"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

var updateBindgenGolden = flag.Bool("update-bindgen-golden", false, "rewrite testdata/bindgen/*.go.golden")

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := "testdata/bindgen/" + name
	if *updateBindgenGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s: got\n%s\nwant\n%s", name, got, want)
	}
}

func TestBindingsGoNamesEachMethod(t *testing.T) {
	src, reason := bindingsGo("example.com/app", []boundType{
		{Pkg: "main", Type: "Todos", Pointer: true, Methods: []string{"Add", "List"}},
		{Pkg: "example.com/app/svc", Type: "Clock", Pointer: false, Methods: []string{"Now"}},
	})
	if reason != "" {
		t.Fatal(reason)
	}
	checkGolden(t, "two.go.golden", src)
	if _, err := parser.ParseFile(token.NewFileSet(), "0mygo_bind_gen.go", src, 0); err != nil {
		t.Errorf("the generated file does not parse: %v\n%s", err, src)
	}
}

// A type of the main package is named without a qualifier, unexported or not
// (reflect says "main" for the main package of any program), and a package
// whose last element two packages share gets an alias for each.
func TestBindingsGoMainPackageAndClashingNames(t *testing.T) {
	src, reason := bindingsGo("example.com/app", []boundType{
		{Pkg: "main", Type: "todos", Pointer: true, Methods: []string{"Add"}},
		{Pkg: "example.com/app", Type: "Notes", Methods: []string{"List"}},
		{Pkg: "example.com/app/a/util", Type: "Helper", Methods: []string{"Run"}},
		{Pkg: "example.com/app/b/util", Type: "Helper", Pointer: true, Methods: []string{"Run", "Stop"}},
	})
	if reason != "" {
		t.Fatal(reason)
	}
	checkGolden(t, "main_and_clash.go.golden", src)
}

func TestGeneratorFallsBackForUnnamedTypes(t *testing.T) {
	for name, b := range map[string]boundType{
		"unexported, other package":    {Pkg: "example.com/app/svc", Type: "clock", Methods: []string{"Now"}},
		"generic":                      {Pkg: "example.com/app/svc", Type: "Box[int]", Methods: []string{"Get"}},
		"generic in main":              {Pkg: "main", Type: "Box[string]", Methods: []string{"Get"}},
		"unnamed":                      {Pkg: "", Type: "", Methods: []string{"X"}},
		"no package":                   {Pkg: "", Type: "Todos", Methods: []string{"Add"}},
		"not an identifier":            {Pkg: "main", Type: "to-dos", Methods: []string{"Add"}},
		"internal package of a module": {Pkg: "example.org/lib/internal/x", Type: "Svc", Methods: []string{"A"}},
		"method that is not exported":  {Pkg: "main", Type: "Svc", Methods: []string{"add"}},
		"a method with no name":        {Pkg: "main", Type: "Svc", Methods: []string{""}},
		"package path that is not Go":  {Pkg: "example.com/app/s v", Type: "Svc", Methods: []string{"A"}},
		"package path with a quote":    {Pkg: `example.com/a"b`, Type: "Svc", Methods: []string{"A"}},
	} {
		src, reason := bindingsGo("example.com/app", []boundType{b})
		if src != nil || reason == "" {
			t.Errorf("%s: %+v: want a fallback reason, got a file (%q)", name, b, reason)
		}
	}
	// One type that cannot be named turns the whole file off, even beside
	// ones that can: a build is all or nothing.
	src, reason := bindingsGo("example.com/app", []boundType{
		{Pkg: "main", Type: "Todos", Pointer: true, Methods: []string{"Add"}},
		{Pkg: "example.com/app/svc", Type: "clock", Methods: []string{"Now"}},
	})
	if src != nil || !strings.Contains(reason, "clock") {
		t.Errorf("a mix: src %v, reason %q", src != nil, reason)
	}
}

// The internal package of the app's own module can be named from its main package.
func TestBindingsGoNamesInternalPackagesOfTheSameModule(t *testing.T) {
	src, reason := bindingsGo("example.com/app", []boundType{
		{Pkg: "example.com/app/internal/store", Type: "Store", Pointer: true, Methods: []string{"Get"}},
	})
	if reason != "" || !bytes.Contains(src, []byte(`"example.com/app/internal/store"`)) {
		t.Errorf("reason %q, src %s", reason, src)
	}
}

// An app that binds nothing of its own has no file.
func TestBindingsGoWithNothingBound(t *testing.T) {
	if src, reason := bindingsGo("example.com/app", nil); src != nil || reason != "" {
		t.Errorf("src %v, reason %q, want neither", src != nil, reason)
	}
}

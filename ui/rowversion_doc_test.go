package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// The first sentence of RowVersion's documentation is the contract an app
// signs: the row's build reads nothing but the row's own data (which
// RowVersion versions) and the list's state. A rewording that loses it fails.
func TestRowVersionDocsStateTheContract(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "list.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "ListState" {
			for _, f := range ts.Type.(*ast.StructType).Fields.List {
				for _, name := range f.Names {
					if name.Name == "RowVersion" && f.Doc != nil {
						doc = f.Doc.Text()
					}
				}
			}
		}
		return true
	})
	if doc == "" {
		t.Fatal("ListState.RowVersion has no documentation")
	}
	first, _, _ := strings.Cut(strings.Join(strings.Fields(doc), " "), ". ")
	for _, want := range []string{
		"RowVersion, when set, lets the list keep a row it built in the last frame instead of building it again",
		"as long as RowVersion(i) returns what it did then",
		"the row's build must read nothing but the row's own data, which RowVersion versions, and the list's state",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("the first sentence of RowVersion's documentation lacks %q:\n%s", want, first)
		}
	}
	if !strings.Contains(doc, "leave RowVersion nil") {
		t.Errorf("RowVersion's documentation does not say what to do with state it does not version")
	}
}

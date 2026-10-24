package uivet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

type pathImporter map[string]*types.Package

func (p pathImporter) Import(path string) (*types.Package, error) { return p[path], nil }

func TestBadTwClasses(t *testing.T) {
	tw := types.NewPackage(twPath, "tw")
	str := types.Typ[types.String]
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, tw, "classes", str)), types.NewTuple(types.NewVar(token.NoPos, tw, "", str)), false)
	tw.Scope().Insert(types.NewFunc(token.NoPos, tw, "S", sig))
	tw.Scope().Insert(types.NewFunc(token.NoPos, tw, "Parse", sig))
	tw.MarkComplete()
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "app.go", `package app
import "github.com/egoist/mygo/ui/widget/tw"
const card = "p-2 "
func view(dyn string) {
	tw.S("p-2 mt-3 hover:bg-blue-600")
	tw.S("p-2 text-green")
	tw.Parse(card + "hover:hexpand")
	tw.S(dyn + "bogus")
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	pkg, err := (&types.Config{Importer: pathImporter{twPath: tw}}).Check("example/app", fs, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range Analyze(fs, []*ast.File{f}, pkg, info) {
		got = append(got, d.Message)
	}
	want := []string{`tw: unknown class "text-green"`, `tw: layout in a variant "hover:hexpand"`}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %q, want %q", got, want)
	}
}

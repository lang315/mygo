package uivet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

type pathImporter map[string]*types.Package

func (p pathImporter) Import(path string) (*types.Package, error) { return p[path], nil }

// fakeTw is a stand-in for package tw: the functions S and Parse, and a type
// Sheet whose method Parse must not count.
func fakeTw() *types.Package {
	tw := types.NewPackage(twPath, "tw")
	str := types.Typ[types.String]
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, tw, "classes", str)), types.NewTuple(types.NewVar(token.NoPos, tw, "", str)), false)
	tw.Scope().Insert(types.NewFunc(token.NoPos, tw, "S", sig))
	tw.Scope().Insert(types.NewFunc(token.NoPos, tw, "Parse", sig))
	name := types.NewTypeName(token.NoPos, tw, "Sheet", nil)
	sheet := types.NewNamed(name, types.NewStruct(nil, nil), nil)
	recv := types.NewVar(token.NoPos, tw, "s", sheet)
	sheet.AddMethod(types.NewFunc(token.NoPos, tw, "Parse", types.NewSignatureType(recv, nil, nil, types.NewTuple(types.NewVar(token.NoPos, tw, "classes", str)), nil, false)))
	tw.Scope().Insert(name)
	tw.MarkComplete()
	return tw
}

// vetTw returns the diagnostics for src, a file of package app.
func vetTw(t *testing.T, src string) []Diagnostic {
	t.Helper()
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "app.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Types: map[ast.Expr]types.TypeAndValue{}}
	pkg, err := (&types.Config{Importer: pathImporter{twPath: fakeTw()}}).Check("example/app", fs, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	return Analyze(fs, []*ast.File{f}, pkg, info)
}

func messages(ds []Diagnostic) []string {
	var out []string
	for _, d := range ds {
		out = append(out, d.Message)
	}
	return out
}

func TestBadTwClasses(t *testing.T) {
	got := messages(vetTw(t, `package app
import "github.com/egoist/mygo/ui/widget/tw"
const card = "p-2 "
func view(dyn string) {
	tw.S("p-2 mt-3 hover:bg-blue-600")
	tw.S("p-2 text-green")
	tw.Parse(card + "hover:hexpand")
	tw.S(dyn + "bogus")
}
`))
	want := []string{`tw: unknown class "text-green"`, `tw: layout in a variant "hover:hexpand"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTwReportsAtTheArgument(t *testing.T) {
	ds := vetTw(t, "package app\nimport \"github.com/egoist/mygo/ui/widget/tw\"\nfunc view() {\n\t_ = tw.S(\"bogus\")\n}\n")
	if len(ds) != 1 || ds[0].Position.Line != 4 || ds[0].Position.Column != 11 {
		t.Fatalf("got %v, want one report at 4:11, the argument", ds)
	}
}

func TestTwKeepsTheOrderOfBadClassesInOneArgument(t *testing.T) {
	got := messages(vetTw(t, "package app\nimport \"github.com/egoist/mygo/ui/widget/tw\"\nfunc view() {\n\ttw.S(\"zz p-2 aa mm\")\n}\n"))
	want := []string{`tw: unknown class "zz"`, `tw: unknown class "aa"`, `tw: unknown class "mm"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTwTypedConstantAndDotImport(t *testing.T) {
	got := messages(vetTw(t, `package app
import . "github.com/egoist/mygo/ui/widget/tw"
const typed string = "bogus"
func view() {
	S(typed)
	Parse("also-bogus")
}
`))
	want := []string{`tw: unknown class "bogus"`, `tw: unknown class "also-bogus"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTwParenthesizedCall(t *testing.T) {
	got := messages(vetTw(t, "package app\nimport \"github.com/egoist/mygo/ui/widget/tw\"\nfunc view() {\n\t(tw.S)(\"bogus\")\n}\n"))
	if want := []string{`tw: unknown class "bogus"`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTwMethodsAreNotChecked(t *testing.T) {
	got := vetTw(t, "package app\nimport \"github.com/egoist/mygo/ui/widget/tw\"\nfunc view(s tw.Sheet) {\n\ts.Parse(\"bogus\")\n}\n")
	if len(got) != 0 {
		t.Fatalf("a method named Parse was reported: %v", got)
	}
}

package widget

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// literalExempt are the files whose literal colors are not a look: markup's
// named colors are the app's content, stylevalue's are the Hex and Fixed
// constructors and the destructive mixes, colordialog's is the default
// value.
var literalExempt = []string{"markup.go", "stylevalue.go", "colordialog.go"}

// literalColors returns, as file:line, each call to ui.Hex, ui.RGB or
// ui.RGBA in the source of the file name, but those of a declaration whose
// name starts with "plain" (the look without a skin) and black with alpha
// (shadows and scrims).
func literalColors(name string, src []byte) ([]string, error) {
	if slices.Contains(literalExempt, filepath.Base(name)) {
		return nil, nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	visit := func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); !ok || x.Name != "ui" {
				return true
			}
			switch sel.Sel.Name {
			case "Hex", "RGB":
			case "RGBA":
				if blackWithAlpha(call.Args) {
					return true
				}
			default:
				return true
			}
			out = append(out, fmt.Sprintf("%s:%d", filepath.Base(name), fset.Position(call.Pos()).Line))
			return true
		})
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if !strings.HasPrefix(d.Name.Name, "plain") {
				visit(d)
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				if v, ok := s.(*ast.ValueSpec); ok && slices.ContainsFunc(v.Names, func(n *ast.Ident) bool {
					return strings.HasPrefix(n.Name, "plain")
				}) {
					continue
				}
				if t, ok := s.(*ast.TypeSpec); ok && strings.HasPrefix(t.Name.Name, "plain") {
					continue
				}
				visit(s)
			}
		}
	}
	return out, nil
}

// blackWithAlpha reports whether the arguments of ui.RGBA are 0, 0, 0 and
// an alpha.
func blackWithAlpha(args []ast.Expr) bool {
	if len(args) != 4 {
		return false
	}
	for _, a := range args[:3] {
		if l, ok := a.(*ast.BasicLit); !ok || l.Value != "0" {
			return false
		}
	}
	return true
}

// Every color a control of this package paints comes from the theme or the
// skin: a literal color is the plain look, named plain..., and nothing
// else.
func TestNoNewLiteralColors(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		found, err := literalColors(name, src)
		if err != nil {
			t.Fatal(err)
		}
		for _, at := range found {
			t.Errorf("%s: a literal color; take it from the theme or the skin, or name its declaration plain...", at)
		}
		n++
	}
	if n < 50 {
		t.Fatalf("only %d files read: not the package's directory?", n)
	}
}

// The rule itself: what it reports and what it lets pass.
func TestLiteralColorsRule(t *testing.T) {
	for _, c := range []struct {
		name, src string
		want      []string
	}{
		{"a.go", "func f() { _ = ui.Hex(\"#fff\") }", []string{"a.go:3"}},
		{"a.go", "var x = ui.RGB(1, 2, 3)", []string{"a.go:3"}},
		{"a.go", "var x = ui.RGBA(1, 0, 0, 0.5)", []string{"a.go:3"}},
		{"a.go", "var x = ui.RGBA(0, 0, 0, 0.5)", nil},
		{"a.go", "var plainX = ui.Hex(\"#fff\")", nil},
		{"a.go", "func plainY() ui.Color { return ui.RGB(1, 2, 3) }", nil},
		{"a.go", "var (\n\tplainA = ui.Hex(\"#fff\")\n\tb      = ui.Hex(\"#000\")\n)", []string{"a.go:5"}},
		{"a.go", "func f() { x := plainFoo; _ = x; _ = ui.Mix }", nil},
		{"markup.go", "var x = ui.RGB(1, 2, 3)", nil},
		{"stylevalue.go", "var x = ui.Hex(\"#123\")", nil},
		{"colordialog.go", "var x = ui.RGB(0, 0, 1)", nil},
	} {
		got, err := literalColors(c.name, []byte("package widget\n\n"+c.src+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s %q: got %v, want %v", c.name, c.src, got, c.want)
		}
	}
}

// testTokens are light and dark tokens whose every color the tests below
// read differs from the others and from the plain look, so that a token
// read in place of another, or the plain color kept, fails.
func testTokens() (light, dark *skinhook.Tokens) {
	fam := func(base, on string) skinhook.Family {
		return skinhook.Family{Base: ui.Hex(base), On: ui.Hex(on)}
	}
	light = &skinhook.Tokens{
		Accent: fam("#7b1fa2", "#ffeb3b"), OnAccent: ui.Hex("#00e5ff"),
		Info: fam("#00695c", "#ff80ab"), Warning: fam("#e65100", "#1a237e"), Danger: fam("#880e4f", "#76ff03"),
		Chart: [8]ui.Color{ui.Hex("#e6194b"), ui.Hex("#3cb44b"), ui.Hex("#ffe119"), ui.Hex("#4363d8"),
			ui.Hex("#f58231"), ui.Hex("#911eb4"), ui.Hex("#42d4f4"), ui.Hex("#f032e6")},
	}
	dark = &skinhook.Tokens{
		Accent: fam("#ce93d8", "#3e2723"), OnAccent: ui.Hex("#004d40"),
		Info: fam("#80cbc4", "#4a148c"), Warning: fam("#ffb74d", "#b71c1c"), Danger: fam("#f48fb1", "#1b5e20"),
		Chart: [8]ui.Color{ui.Hex("#bfef45"), ui.Hex("#fabed4"), ui.Hex("#469990"), ui.Hex("#dcbeff"),
			ui.Hex("#9a6324"), ui.Hex("#800000"), ui.Hex("#aaffc3"), ui.Hex("#808000")},
	}
	return light, dark
}

// useTestSkin turns testTokens on until the test ends.
func useTestSkin(t *testing.T) (light, dark *skinhook.Tokens) {
	light, dark = testTokens()
	skinhook.Set(light, dark)
	t.Cleanup(func() { skinhook.Set(nil, nil) })
	return light, dark
}

// nearTol reports whether a is within tol of c on every channel.
func nearTol(a color.RGBA, c ui.Color, tol int) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= tol && int(y)-int(x) <= tol }
	return d(a.R, c.R) && d(a.G, c.G) && d(a.B, c.B)
}

// countNear counts the pixels of img inside r (outside it when out) within
// tol of c.
func countNear(img *image.RGBA, r ui.Rect, out bool, c ui.Color, tol int) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			in := float32(x) >= r.X && float32(x) < r.X+r.W && float32(y) >= r.Y && float32(y) < r.Y+r.H
			if in != out && nearTol(img.RGBAAt(x, y), c, tol) {
				n++
			}
		}
	}
	return n
}

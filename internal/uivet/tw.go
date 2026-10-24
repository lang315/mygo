package uivet

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/egoist/mygo/internal/twclass"
)

const twPath = "github.com/egoist/mygo/ui/widget/tw"

// twClasses reports the bad classes of a constant string given to tw.S
// or tw.Parse. A string built at run time is left to tw's log.
func twClasses(call *ast.CallExpr, info *types.Info, report func(token.Pos, string)) {
	var id *ast.Ident
	switch f := call.Fun.(type) {
	case *ast.SelectorExpr:
		id = f.Sel
	case *ast.Ident:
		id = f
	}
	if id == nil || len(call.Args) != 1 {
		return
	}
	fn, ok := info.Uses[id].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != twPath || fn.Name() != "S" && fn.Name() != "Parse" {
		return
	}
	v := info.Types[call.Args[0]].Value
	if v == nil || v.Kind() != constant.String {
		return
	}
	_, bad := twclass.Parse(constant.StringVal(v))
	for _, b := range bad {
		report(call.Args[0].Pos(), fmt.Sprintf("tw: %s %q", b.Reason, b.Class))
	}
}

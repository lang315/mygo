//go:build mygo_static && !js

package mygo

import "reflect"

// goldenService's methods, as mygo build lists them: the IPC goldens run on
// the table under mygo_static, and must record the same bytes as by reflection.
func init() {
	BindMethods(reflect.TypeFor[goldenService](),
		BoundMethod{Name: "Any", Func: goldenService.Any},
		BoundMethod{Name: "Big", Func: goldenService.Big},
		BoundMethod{Name: "Bool", Func: goldenService.Bool},
		BoundMethod{Name: "Bytes", Func: goldenService.Bytes},
		BoundMethod{Name: "Ctx", Func: goldenService.Ctx},
		BoundMethod{Name: "Dur", Func: goldenService.Dur},
		BoundMethod{Name: "Duration", Func: goldenService.Duration},
		BoundMethod{Name: "F64", Func: goldenService.F64},
		BoundMethod{Name: "Fail", Func: goldenService.Fail},
		BoundMethod{Name: "Float", Func: goldenService.Float},
		BoundMethod{Name: "HTML", Func: goldenService.HTML},
		BoundMethod{Name: "I8", Func: goldenService.I8},
		BoundMethod{Name: "Int", Func: goldenService.Int},
		BoundMethod{Name: "Invalid", Func: goldenService.Invalid},
		BoundMethod{Name: "Map", Func: goldenService.Map},
		BoundMethod{Name: "NaN", Func: goldenService.NaN},
		BoundMethod{Name: "Nested", Func: goldenService.Nested},
		BoundMethod{Name: "Nil", Func: goldenService.Nil},
		BoundMethod{Name: "NilMap", Func: goldenService.NilMap},
		BoundMethod{Name: "Nothing", Func: goldenService.Nothing},
		BoundMethod{Name: "Optional", Func: goldenService.Optional},
		BoundMethod{Name: "Panic", Func: goldenService.Panic},
		BoundMethod{Name: "Point", Func: goldenService.Point},
		BoundMethod{Name: "PointPtr", Func: goldenService.PointPtr},
		BoundMethod{Name: "Small", Func: goldenService.Small},
		BoundMethod{Name: "Str", Func: goldenService.Str},
		BoundMethod{Name: "Strs", Func: goldenService.Strs},
		BoundMethod{Name: "Time", Func: goldenService.Time},
		BoundMethod{Name: "U64", Func: goldenService.U64},
		BoundMethod{Name: "Variadic", Func: goldenService.Variadic},
	)
}

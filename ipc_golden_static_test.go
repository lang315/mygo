//go:build mygo_static && !js

package mygo

import "reflect"

// goldenService's methods, as mygo build lists them: the IPC goldens run on
// the table under mygo_static, and must record the same bytes as by reflection.
func init() {
	BindMethods(reflect.TypeFor[goldenService](),
		BoundMethod{"Any", goldenService.Any},
		BoundMethod{"Big", goldenService.Big},
		BoundMethod{"Bool", goldenService.Bool},
		BoundMethod{"Bytes", goldenService.Bytes},
		BoundMethod{"Ctx", goldenService.Ctx},
		BoundMethod{"Dur", goldenService.Dur},
		BoundMethod{"Duration", goldenService.Duration},
		BoundMethod{"F64", goldenService.F64},
		BoundMethod{"Fail", goldenService.Fail},
		BoundMethod{"Float", goldenService.Float},
		BoundMethod{"HTML", goldenService.HTML},
		BoundMethod{"I8", goldenService.I8},
		BoundMethod{"Int", goldenService.Int},
		BoundMethod{"Invalid", goldenService.Invalid},
		BoundMethod{"Map", goldenService.Map},
		BoundMethod{"NaN", goldenService.NaN},
		BoundMethod{"Nested", goldenService.Nested},
		BoundMethod{"Nil", goldenService.Nil},
		BoundMethod{"NilMap", goldenService.NilMap},
		BoundMethod{"Nothing", goldenService.Nothing},
		BoundMethod{"Optional", goldenService.Optional},
		BoundMethod{"Panic", goldenService.Panic},
		BoundMethod{"Point", goldenService.Point},
		BoundMethod{"PointPtr", goldenService.PointPtr},
		BoundMethod{"Small", goldenService.Small},
		BoundMethod{"Str", goldenService.Str},
		BoundMethod{"Strs", goldenService.Strs},
		BoundMethod{"Time", goldenService.Time},
		BoundMethod{"U64", goldenService.U64},
		BoundMethod{"Variadic", goldenService.Variadic},
	)
}

//go:build windows && !mygo_noinspector

package d3d11

import (
	_ "embed"
	"fmt"

	"github.com/egoist/mygo/internal/gpu"
)

// Builds with the developer tools (not mygo_noinspector, as mygo build
// makes) keep classes.hlsl: a developer who changes it runs the program
// without go generate, and the shaders are compiled at startup. A release
// build leaves the source out (shader_release.go): shaders.z always holds
// bytecode that matches it, which TestShaderBytecode checks.

//go:embed classes.hlsl
var classSource string

// sourceStale reports whether shader.hlsl or classes.hlsl changed since
// shaders.go was generated.
func sourceStale() bool { return gpu.SourceSum(shaderSource+classSource) != shaderSum }

// compileClassesNow reports whether the shaders of classes are compiled from
// their sources, for shaders.go older than them, or for tests.
func compileClassesNow() bool { return sourceStale() || compileShaders }

// classDefines returns what classes.hlsl needs defined for the classes cl (a
// bit, 1 << gpu.Class, for each), as gen.go compiles it.
func classDefines(cl uint32) string {
	s := fmt.Sprintf("#define CLASSES %d\n", cl)
	for _, c := range []struct {
		name  string
		class gpu.Class
	}{{"PLAIN", gpu.ClassPlain}, {"FILL", gpu.ClassFill}, {"SHADOW", gpu.ClassShadow}, {"GLYPH", gpu.ClassGlyph}, {"SUBPIXEL", gpu.ClassSubpixel}, {"IMAGE", gpu.ClassImage}} {
		s += fmt.Sprintf("#define %s %d\n", c.name, cl>>c.class&1)
	}
	return s
}

// compileClass compiles the shaders of the classes cl from their sources, for
// shaders.go older than them.
func compileClass(cl uint32) (vs, ps []byte, err error) {
	src := classDefines(cl) + shaderSource + "\n" + classSource
	if vs, err = compile(src, "classvs", "vs_4_0"); err != nil {
		return nil, nil, err
	}
	if ps, err = compile(src, "classps", "ps_4_0"); err != nil {
		return nil, nil, err
	}
	return vs, ps, nil
}

//go:build windows && mygo_noinspector

package d3d11

import "testing"

// In a release build compileShaders, which makes a development build compile
// the shaders from their sources, has no effect: the shaders are those of
// shaders.z, and nothing is compiled (there is no classes.hlsl to compile).
func TestReleaseBuildTakesShadersFromTheBytecode(t *testing.T) {
	compileShaders = true
	t.Cleanup(func() { compileShaders = false })
	if compileFromSource() || sourceStale() {
		t.Fatal("a release build compiles shaders from their sources")
	}
	packed, err := unpackedShaders()
	if err != nil {
		t.Fatal(err)
	}
	code, err := shaderCode()
	if err != nil {
		t.Fatal(err)
	}
	// The slices of the bytecode itself, not the output of a compiler.
	if &code.vs[0] != &packed.vs[0] || &code.ps[0] != &packed.ps[0] || &code.blur[0] != &packed.blur[0] {
		t.Fatal("shaderCode compiled shaders although compileShaders says nothing in a release build")
	}
	if _, _, err := compileClass(classShaders[0].classes); err == nil {
		t.Fatal("a release build compiled the shaders of a class from a source it does not have")
	}
	if len(classShaders[0].vs) == 0 || len(classShaders[0].ps) == 0 {
		t.Fatal("the shaders of classes are not those of shaders.z")
	}
}

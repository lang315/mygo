//go:build windows && mygo_noinspector

package d3d11

import "errors"

// A release build has no classes.hlsl (see shader_dev.go): the shaders of
// classes are those of shaders.z.

// sourceStale is false: shaders.z matches the sources (TestShaderBytecode).
func sourceStale() bool { return false }

// compileClassesNow is false: there is no source to compile.
func compileClassesNow() bool { return false }

func compileClass(uint32) (vs, ps []byte, err error) {
	return nil, nil, errors.New("d3d11: the shaders of classes are compiled from classes.hlsl in builds with the developer tools only")
}

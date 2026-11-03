//go:build windows && mygo_noinspector

package d3d11

import "errors"

// A release build has no classes.hlsl (see shader_dev.go): the shaders of
// classes are those of shaders.z.

// sourceStale is false: shaders.z matches the sources (TestShaderBytecode).
func sourceStale() bool { return false }

// compileFromSource is false whatever compileShaders says (it only has an
// effect in builds with the developer tools): there is no source of the
// classes to compile, and the bytecode of shaders.z matches the other.
func compileFromSource() bool { return false }

func compileClass(uint32) (vs, ps []byte, err error) {
	return nil, nil, errors.New("d3d11: the shaders of classes are compiled from classes.hlsl in builds with the developer tools only")
}

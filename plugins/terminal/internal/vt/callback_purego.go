//go:build darwin || (linux && (amd64 || arm64))

package vt

import "github.com/ebitengine/purego"

func newCallback(fn any) uintptr { return purego.NewCallback(fn) }

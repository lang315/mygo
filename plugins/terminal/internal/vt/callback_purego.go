//go:build (darwin && !ios) || (linux && !android && (amd64 || arm64))

package vt

import "github.com/ebitengine/purego"

func newCallback(fn any) uintptr { return purego.NewCallback(fn) }

//go:build windows && (amd64 || arm64)

package e2e

import "github.com/egoist/mygo/internal/gpu/d3d11"

func stressState() string { return d3d11.StressState() }

//go:build !js && !(windows && (amd64 || arm64))

package e2e

import "time"

// PROBE (win-copy-flake): the clipboard sampler is Windows's only.
type copyProbe struct{}

func startCopyProbe() *copyProbe  { return nil }
func (*copyProbe) sent(time.Time) {}
func (*copyProbe) stop() string   { return "" }

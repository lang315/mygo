//go:build !js && !(windows && (amd64 || arm64))

package e2e

import "time"

// The clipboard sampler is Windows's only: with none, copydrop.Skip never
// skips.
type copyProbe struct{}

func startCopyProbe() *copyProbe                               { return nil }
func (*copyProbe) sent(time.Time)                              {}
func (*copyProbe) stop() string                                { return "" }
func (*copyProbe) facts() (time.Duration, time.Duration, bool) { return 0, 0, false }
func (*copyProbe) webView2() string                            { return "" }

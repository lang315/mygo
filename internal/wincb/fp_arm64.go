//go:build windows && arm64

package wincb

// The test of the floating point registers (fp_arm64_test.go), in assembly.

// fpProbe is a C function: fpProbe(f) sets the registers F8-F15 to known
// values, calls the C function f as native code does, and returns a bit for
// each register f changed. It keeps the registers it was called with.
func fpProbe()

// fpProbeAddr returns the address of fpProbe.
func fpProbeAddr() uintptr

// clobberFP sets F8-F15, which Go code may do anywhere.
func clobberFP()

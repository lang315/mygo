//go:build !darwin

package e2e

// inputDiag describes the last input for a failing wait; only macOS records it.
func inputDiag() string { return "" }

//go:build !darwin

package e2e

// inputDiag describes the last input for a failing wait; only macOS records it.
func inputDiag() string { return "" }

// frontmostBundleID is the frontmost app's identifier; only macOS locks the
// desktop in a way the tests see.
func frontmostBundleID() string { return "" }

// keepFront brings the window of the last input back to the front; only
// macOS tests lose it to other apps.
func keepFront() {}

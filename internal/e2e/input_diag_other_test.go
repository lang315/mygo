//go:build !darwin && !js

package e2e

// inputDiag describes the last input for a failing wait; only macOS records it.
func inputDiag() string { return "" }

// frontmostBundleID is the frontmost app's identifier; only macOS locks the
// desktop in a way the tests see.
func frontmostBundleID() string { return "" }

// keepFront gets the front back for the waits that poll it. macOS tests lose
// it to other apps and focus the window of the last input. Windows gives the
// front of a window it destroys to the next one down, which can be another
// app's, and only a process that received the last input takes it back.
func keepFront() { regrantForeground() }

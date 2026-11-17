//go:build !unix && !windows

package mygo

// crossDevice reports whether a rename failed because it crossed volumes.
func crossDevice(error) bool { return false }

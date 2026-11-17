//go:build !unix && !js

package mygo

// socketOwnedByMe is true: the socket's directory is the user's own.
func socketOwnedByMe(string) bool { return true }

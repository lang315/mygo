//go:build !unix && !js

package mygo

import "os"

// The socket's and lock's directory is the user's own.
func socketOwnedByMe(string) bool { return true }
func fileOwnedByMe(string) bool   { return true }
func ownedByMe(*os.File) bool     { return true }

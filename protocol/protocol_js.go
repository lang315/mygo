//go:build js

// Package protocol serves custom URL schemes from Go, and the app's frontend
// at mygo://, for web pages, which browsers do not show: the handlers need
// net/http, which builds for browsers leave out, so Handle, HandleFunc and
// FileServer are not here, and what is, does nothing. An app that imports the
// package builds for browsers all the same.
package protocol

import "io/fs"

// SetFrontend does nothing: there is no web page to serve a frontend to.
func SetFrontend(fsys fs.FS) {}

// Unhandle does nothing: no scheme has a handler.
func Unhandle(scheme string) {}

// IsHandled reports whether scheme has a handler: none has.
func IsHandled(scheme string) bool { return false }

//go:build !js

// Package protocol serves custom URL schemes from Go, and the app's frontend
// at mygo://:
//
//	protocol.Handle("thumbs", protocol.FileServer(os.DirFS(cacheDir)))
//	win.Page().LoadURL("thumbs://localhost/")
//
// It is a package of its own, and package mygo does not import it, because it
// imports net/http, which costs an app 2.2 MB whatever of it the app calls:
// apps of native UI, and web pages that load their own HTML, link none of it.
// An app that serves a scheme or a frontend imports it. `mygo build` does so
// for the apps with a frontendDist, with a generated file that imports this
// package and calls SetFrontend.
//
// Before MyGo moved it here, these were mygo.Protocol.Handle,
// mygo.Protocol.HandleFunc, mygo.Protocol.Unhandle, mygo.Protocol.IsHandled,
// mygo.FileServer and mygo.SetFrontend.
package protocol

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/egoist/mygo/internal/schemehook"
)

var (
	mu       sync.RWMutex
	handlers = map[string]http.Handler{}
)

func init() {
	schemehook.Linked.Store(&schemehook.Serving{Serve: serve, Download: download})
}

var reservedSchemes = map[string]bool{
	"http": true, "https": true, "file": true, "about": true, "data": true, "blob": true,
	"javascript": true, "ws": true, "wss": true, "ftp": true, "mailto": true, "webkit": true,
}

// Handle serves requests for scheme with an http.Handler, for example
// files generated at run time:
//
//	protocol.Handle("thumbs", protocol.FileServer(os.DirFS(cacheDir)))
//	win.Page().LoadURL("thumbs://localhost/")
//
// Pages served this way are loaded like regular web pages: they can use
// fetch, ES modules and relative URLs. Schemes must be registered before
// the windows that use them are created. Handlers run on their own
// goroutines and responses are streamed to the page. mygo.CallerPage and
// mygo.CallerWindow of the request's Context tell which page asks.
//
// The mygo scheme serves the app's frontend (see SetFrontend) unless it is
// handled here.
func Handle(scheme string, handler http.Handler) error {
	scheme = strings.ToLower(scheme)
	if !validScheme(scheme) {
		return fmt.Errorf("mygo: invalid scheme %q", scheme)
	}
	if reservedSchemes[scheme] {
		return fmt.Errorf("mygo: scheme %q is reserved", scheme)
	}
	// The names package mygo keeps change with the handlers, under the same
	// lock: apart, a Handle and an Unhandle of one scheme could leave the two
	// lists saying different things.
	mu.Lock()
	handlers[scheme] = handler
	schemehook.Register(scheme)
	mu.Unlock()
	return nil
}

// HandleFunc is Handle for a handler function.
func HandleFunc(scheme string, fn func(http.ResponseWriter, *http.Request)) error {
	return Handle(scheme, http.HandlerFunc(fn))
}

// Unhandle removes the handler of scheme. Pages requesting it get an error.
func Unhandle(scheme string) {
	scheme = strings.ToLower(scheme)
	mu.Lock()
	delete(handlers, scheme)
	schemehook.Unregister(scheme)
	mu.Unlock()
}

// IsHandled reports whether scheme has a handler.
func IsHandled(scheme string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := handlers[strings.ToLower(scheme)]
	return ok
}

func validScheme(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

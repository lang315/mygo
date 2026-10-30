// Package schemehook is where package protocol, which serves custom URL
// schemes with net/http, gives package mygo, which must not import net/http
// (it costs an app 2.2 MB), the serving of the requests and downloads of the
// webview, and where the names of the schemes the app handles are kept for
// both. Nothing here depends on net/http, mygo or protocol.
package schemehook

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/internal/platform"
)

// FrontendScheme is the scheme that serves the app's frontend.
const FrontendScheme = "mygo"

// Call is what package mygo gives the serving of a request: the context of
// the request with the page that asks (mygo.CallerPage, mygo.CallerWindow),
// the id of its window, the CORS policy of the window, and the main thread.
type Call struct {
	Context context.Context
	Window  int
	// CORS sets the CORS headers of a response for the request with the given
	// method and headers, and reports whether the request was a preflight it
	// answered.
	CORS func(header map[string][]string, method string, request map[string][]string) (preflight bool)
	// OnMain runs fn on the main thread and waits; PostMain queues it and
	// reports whether it was.
	OnMain   func(fn func())
	PostMain func(fn func()) bool
}

// Download is the download of a URL of a custom scheme.
type Download struct {
	URL string
	// Started asks the window where to save the file called name, on the
	// main thread: "" cancels it.
	Started func(name string) string
	// Finished reports the end of the download to the window.
	Finished func(path string, err error)
}

// Serving serves what the webview asks of custom schemes.
type Serving struct {
	// Serve answers a request, which it is given on the main thread.
	Serve func(req *platform.SchemeRequest, call Call)
	// Download runs on a goroutine of its own.
	Download func(d Download)
}

// Linked is the serving, set by package protocol when the app imports it:
// nil until then, and in apps that do not, those of native UI among them.
// It is read at every request, from any goroutine.
var Linked atomic.Pointer[Serving]

var (
	mu      sync.RWMutex
	handled = map[string]bool{}
)

// Register notes that the app handles scheme (any case).
func Register(scheme string) {
	mu.Lock()
	handled[strings.ToLower(scheme)] = true
	mu.Unlock()
}

// Unregister notes that the app no longer handles scheme.
func Unregister(scheme string) {
	mu.Lock()
	delete(handled, strings.ToLower(scheme))
	mu.Unlock()
}

// Handled reports whether the app handles scheme (any case).
func Handled(scheme string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return handled[strings.ToLower(scheme)]
}

// Schemes returns the schemes the webviews of the app must route to Go: the
// ones it handles and the frontend's.
func Schemes() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(handled)+1)
	for s := range handled {
		out = append(out, s)
	}
	if !handled[FrontendScheme] {
		out = append(out, FrontendScheme)
	}
	slices.Sort(out)
	return out
}

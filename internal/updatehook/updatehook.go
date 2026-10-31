// Package updatehook is where package updatefeed, which imports net/http,
// gives package mygo, which must not (importing net/http costs an app 2.2
// MB), the source of the update feed that mygo.config.ts configures.
package updatehook

import (
	"context"
	"io"
	"sync/atomic"
)

// Source is updates.Source: where updates come from.
type Source interface {
	Manifest(ctx context.Context, prereleases bool) (io.ReadCloser, error)
	Open(ctx context.Context, url string) (io.ReadCloser, error)
}

// Feed makes the source of the feed at a URL. It is nil until package
// updatefeed is linked into the app, which `mygo build` does for the apps
// with updates in mygo.config.ts.
var Feed atomic.Pointer[func(feed string) Source]

// RelaunchStaged is set by package updates, which stages updates: it asks the
// helper of an update staged on macOS to open the app once it has installed the
// update, and reports whether there is one. Package mygo, which relaunches the
// app, calls it if the app links package updates.
var RelaunchStaged atomic.Pointer[func() bool]

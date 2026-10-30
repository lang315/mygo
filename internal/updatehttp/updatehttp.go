// Package updatehttp is the HTTP source of updates, the one package updatefeed
// gives the app and the tests of package mygo use: the files of an update
// fetched over HTTPS through the system's proxy. It imports net/http, which
// package mygo does not.
package updatehttp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/egoist/mygo/internal/update"
	"github.com/egoist/mygo/netproxy"
)

// Feed is the update feed that mygo.config.ts configures, the URL of a
// manifest or a feed that update.ResolveFeed resolves. It is a
// mygo.UpdateSource.
type Feed string

// Manifest returns the manifest of the feed, resolved to the newest release
// when the feed is a GitHub one.
func (f Feed) Manifest(ctx context.Context, pre bool) (io.ReadCloser, error) {
	resolve := update.ResolveFeed
	if pre {
		resolve = update.ResolvePrereleaseFeed
	}
	feed, err := resolve(ctx, string(f), Get)
	if err != nil {
		return nil, err
	}
	return Get(ctx, feed)
}

// Open returns the file at url, which the manifest names.
func (f Feed) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	return Get(ctx, url)
}

// Get fetches a URL over HTTPS (or plain HTTP from the loopback
// interface, for tests), through the system's proxy (package netproxy).
func Get(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil, fmt.Errorf("refusing to download %s: updates need HTTPS", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	// Not the app's name: GitHub's release assets come from a store that
	// answers 400 to a User-Agent with bytes outside ASCII, as 云音乐, for
	// every file its CDN has not cached yet.
	req.Header.Set("User-Agent", "mygo-updater")
	resp, err := (&http.Client{Transport: netproxy.Transport()}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", rawURL, resp.Status)
	}
	return resp.Body, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

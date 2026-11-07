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
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/internal/update"
	"github.com/egoist/mygo/netproxy"
)

// Feed is the update feed that mygo.config.ts configures, the URL of a
// manifest or a feed that update.ResolveFeed resolves. It is a
// updates.Source.
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
// A failure that a moment may mend, as a gateway timing out in front of
// the store of a release's assets, is tried again (httpRetries).
func Get(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil, fmt.Errorf("refusing to download %s: updates need HTTPS", rawURL)
	}
	client := &http.Client{Transport: netproxy.Transport()}
	for try := 0; ; try++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		// Not the app's name: GitHub's release assets come from a store that
		// answers 400 to a User-Agent with bytes outside ASCII, as 云音乐, for
		// every file its CDN has not cached yet.
		req.Header.Set("User-Agent", "mygo-updater")
		resp, err := client.Do(req)
		var wait time.Duration
		switch {
		case err != nil:
			if ctx.Err() != nil || try >= len(httpRetries) {
				return nil, err
			}
		case resp.StatusCode == http.StatusOK:
			return resp.Body, nil
		default:
			resp.Body.Close()
			if !transientStatus(resp.StatusCode) || try >= len(httpRetries) {
				return nil, fmt.Errorf("GET %s: %s", rawURL, resp.Status)
			}
			wait = retryAfter(resp.Header.Get("Retry-After"))
		}
		wait = max(wait, httpRetries[try])
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// httpRetries are how long Get waits before each try after the first of a
// request that failed for a while: three tries in all.
var httpRetries = []time.Duration{time.Second, 3 * time.Second}

// transientStatus reports whether a response of status code may succeed
// when asked again: too many requests, or a server or gateway failing for
// now.
func transientStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryAfter is the wait of a Retry-After header in seconds, up to 30
// seconds; 0 for none, or for a date.
func retryAfter(h string) time.Duration {
	n, err := strconv.Atoi(strings.TrimSpace(h))
	if err != nil || n <= 0 {
		return 0
	}
	return min(time.Duration(n)*time.Second, 30*time.Second)
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

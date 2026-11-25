// Package updatehttp is the HTTP source of updates, the one package updatefeed
// gives the app and the tests of package mygo use: the files of an update
// fetched over HTTPS through the system's proxy. It imports net/http, which
// package mygo does not.
package updatehttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
//
// The deadlines of the request are these: the headers must arrive within
// httpHeaderTimeout, and a body that sends nothing for httpStallTimeout is
// given up. Without them a connection that the network dropped without a
// reset holds the caller until the socket times out, which keepalive may
// never do, and a check in the background has no window to cancel it from
// and keeps the updater from scheduling the next one.
func Get(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil, fmt.Errorf("refusing to download %s: updates need HTTPS", rawURL)
	}
	client := httpClient()
	for try := 0; ; try++ {
		// Canceled when the body closes, or when nothing arrives: the one
		// cancels the request of a body that stopped, the other ends a
		// request whose response was refused.
		reqCtx, cancel := context.WithCancel(ctx)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
		if err != nil {
			cancel()
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
			cancel()
			if ctx.Err() != nil || try >= len(httpRetries) || errors.Is(err, errRedirect) {
				return nil, err
			}
		case resp.StatusCode == http.StatusOK:
			return newStalledBody(resp.Body, cancel), nil
		default:
			resp.Body.Close()
			cancel()
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

// httpHeaderTimeout is how long the headers of a response may take to
// arrive: the client has no deadline above the socket. Five minutes of a
// check in the background would be five minutes without the next check.
var httpHeaderTimeout = 30 * time.Second

// httpStallTimeout is how long a body may send nothing before the download
// is given up. A slow download is fine, however long it takes; one that
// stopped is not, and its reads never return.
var httpStallTimeout = time.Minute

// httpClient is the client of the updater's requests: the transport of
// package netproxy, so that the system's proxy settings apply, with the
// deadline for the headers that it leaves to its callers. A test that needs
// another deadline changes the transport for its run.
var updateHTTPClient struct {
	once   sync.Once
	client *http.Client
}

func httpClient() *http.Client {
	updateHTTPClient.once.Do(func() { updateHTTPClient.client = newHTTPClient() })
	return updateHTTPClient.client
}

func newHTTPClient() *http.Client {
	transport := netproxy.Transport().Clone()
	transport.ResponseHeaderTimeout = httpHeaderTimeout
	return &http.Client{Transport: transport, CheckRedirect: checkRedirect}
}

// checkRedirect follows a redirect of an update's request only to HTTPS,
// as Get asks of the address it starts from: a feed or an archive served
// over HTTPS that redirects to plain HTTP would otherwise be read from
// anyone on the way. A test's server on the loopback may redirect over
// HTTP, as it may serve over it.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("%w: stopped after 10", errRedirect)
	}
	if req.URL.Scheme != "https" && !(req.URL.Scheme == "http" && isLoopback(req.URL.Hostname())) {
		return fmt.Errorf("%w to %s: updates need HTTPS", errRedirect, req.URL.Redacted())
	}
	return nil
}

// errRedirect is the error of a redirect checkRedirect refuses, which Get
// does not try again: the server would only redirect again.
var errRedirect = errors.New("refusing the redirect")

// stalledBody cancels the request of a body that stops arriving, and says
// so instead of the canceled request it would otherwise be read as.
type stalledBody struct {
	body   io.ReadCloser
	cancel context.CancelFunc
	// d is the timeout it was made with, which the watch reads in its own
	// goroutine: reading the setting there would race with a test's change.
	d time.Duration
	// last is when the last byte arrived, in Unix nanoseconds; stalled is
	// set once cancel was called, so that the error can say why; done ends
	// the watch when the body closes.
	last    atomic.Int64
	stalled atomic.Bool
	done    atomic.Bool
	timer   *time.Timer
}

func newStalledBody(body io.ReadCloser, cancel context.CancelFunc) *stalledBody {
	b := &stalledBody{body: body, cancel: cancel, d: httpStallTimeout}
	b.last.Store(time.Now().UnixNano())
	// Publish the timer before its callback can read it to rearm it.
	b.timer = time.AfterFunc(1<<63-1, b.check)
	b.timer.Stop()
	b.timer.Reset(b.d / 4)
	return b
}

// check cancels the request when nothing arrived for its timeout, and looks
// again a quarter of that later otherwise: often enough to stop a download
// soon after it stops, rarely enough to cost nothing while it runs.
func (b *stalledBody) check() {
	if b.done.Load() {
		return
	}
	if time.Since(time.Unix(0, b.last.Load())) >= b.d {
		b.stalled.Store(true)
		b.cancel()
		return
	}
	b.timer.Reset(b.d / 4)
}

func (b *stalledBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if n > 0 {
		b.last.Store(time.Now().UnixNano())
	}
	if err != nil && b.stalled.Load() {
		return n, errors.New("the download stalled: the connection sent nothing")
	}
	return n, err
}

func (b *stalledBody) Close() error {
	b.done.Store(true)
	b.timer.Stop()
	b.cancel()
	return b.body.Close()
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

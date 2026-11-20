package updatehttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/wait"
)

// A gateway failing for a moment is asked again, and an error that stays
// is returned.
func TestGetRetries(t *testing.T) {
	defer func(r []time.Duration) { httpRetries = r }(httpRetries)
	httpRetries = []time.Duration{time.Millisecond, time.Millisecond}
	var tries atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/missing":
			tries.Add(1)
			http.NotFound(w, r)
		case r.URL.Path == "/down":
			tries.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		case tries.Add(1) < 3:
			w.WriteHeader(http.StatusGatewayTimeout)
		default:
			io.WriteString(w, "ok")
		}
	}))
	defer srv.Close()
	body, err := Get(t.Context(), srv.URL+"/file")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(body)
	body.Close()
	if string(b) != "ok" || tries.Load() != 3 {
		t.Errorf("read %q in %d tries", b, tries.Load())
	}
	tries.Store(0)
	if _, err := Get(t.Context(), srv.URL+"/down"); err == nil || !strings.Contains(err.Error(), "503") || tries.Load() != 3 {
		t.Errorf("down: %v in %d tries", err, tries.Load())
	}
	tries.Store(0)
	if _, err := Get(t.Context(), srv.URL+"/missing"); err == nil || tries.Load() != 1 {
		t.Errorf("missing: %v in %d tries", err, tries.Load())
	}
	if got := retryAfter(" 5 "); got != 5*time.Second {
		t.Errorf("Retry-After 5: %v", got)
	}
}

// A server that accepts the connection and never answers is given up on,
// rather than holding a check open until the socket times out.
func TestGetHeaderTimeout(t *testing.T) {
	transport := httpClient().Transport.(*http.Transport)
	defer func(r []time.Duration, d time.Duration) {
		httpRetries, transport.ResponseHeaderTimeout = r, d
	}(httpRetries, transport.ResponseHeaderTimeout)
	httpRetries = []time.Duration{time.Millisecond}
	transport.ResponseHeaderTimeout = 50 * time.Millisecond

	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-block
	}))
	defer srv.Close()
	defer close(block)

	// Get returns by itself: its own timeout, not a canceled context, gives up.
	errs := make(chan error, 1)
	go func() {
		_, err := Get(t.Context(), srv.URL+"/file")
		errs <- err
	}()
	if err := wait.For(t, "Get to give up on a server that never answers", errs); err == nil {
		t.Fatal("a server that never answers returned no error")
	}
}

// A body that stops arriving is given up on, rather than blocking the read
// of a download until the socket times out.
func TestGetStallGivesUp(t *testing.T) {
	defer func(d time.Duration) { httpStallTimeout = d }(httpStallTimeout)
	httpStallTimeout = 100 * time.Millisecond

	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		io.WriteString(w, "start")
		w.(http.Flusher).Flush()
		<-block
	}))
	defer srv.Close()
	defer close(block)

	body, err := Get(t.Context(), srv.URL+"/file")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if _, err := io.ReadAll(body); err == nil || !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("reading a body that stopped ended with %v", err)
	}
}

// Even an immediately firing callback sees a fully initialized body.
func TestStalledBodyShortTimeout(t *testing.T) {
	defer func(d time.Duration) { httpStallTimeout = d }(httpStallTimeout)
	httpStallTimeout = time.Nanosecond
	for range 100 {
		ctx, cancel := context.WithCancel(t.Context())
		body := newStalledBody(io.NopCloser(strings.NewReader("")), cancel)
		wait.For(t, "the stalled body to cancel its request", ctx.Done())
		body.Close()
		if !body.stalled.Load() {
			t.Fatal("the body did not report a stall")
		}
	}
}

// A body that keeps arriving is not given up, however slowly it sends.
func TestGetKeepsASlowBody(t *testing.T) {
	defer func(d time.Duration) { httpStallTimeout = d }(httpStallTimeout)
	// Each byte comes well within the stall timeout, even on a loaded
	// runner whose sleeps overrun, and all of them take longer than it.
	httpStallTimeout = 400 * time.Millisecond

	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 10; i++ {
			io.WriteString(w, "x")
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
		<-block
	}))
	defer srv.Close()
	defer close(block)

	body, err := Get(t.Context(), srv.URL+"/file")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	b, err := io.ReadAll(io.LimitReader(body, 10))
	if err != nil || len(b) != 10 {
		t.Fatalf("read %d bytes of a slow body: %v", len(b), err)
	}
}

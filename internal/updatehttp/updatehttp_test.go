package updatehttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

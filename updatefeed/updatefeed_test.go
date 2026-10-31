//go:build !js

package updatefeed

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/updatehook"
)

// TestImportingGivesTheFeed: importing the package is what links the HTTP
// feed into package updates, and the source it gives fetches the manifest and
// the files over HTTP (loopback only, https otherwise).
func TestImportingGivesTheFeed(t *testing.T) {
	feed := updatehook.Feed.Load()
	if feed == nil {
		t.Fatal("importing updatefeed did not set the feed of package mygo")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "mygo-updater" {
			http.Error(w, "agent", http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/update.json":
			io.WriteString(w, `{"version":"1.2.0"}`)
		case "/app.tar.gz":
			io.WriteString(w, "archive")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	for name, src := range map[string]updatehook.Source{
		"the hook": (*feed)(srv.URL + "/update.json"),
		"Source":   Source(srv.URL + "/update.json"),
	} {
		body, err := src.Manifest(t.Context(), false)
		if err != nil {
			t.Fatalf("%s: Manifest: %v", name, err)
		}
		got, _ := io.ReadAll(body)
		body.Close()
		if string(got) != `{"version":"1.2.0"}` {
			t.Errorf("%s: manifest %q", name, got)
		}
		file, err := src.Open(t.Context(), srv.URL+"/app.tar.gz")
		if err != nil {
			t.Fatalf("%s: Open: %v", name, err)
		}
		got, _ = io.ReadAll(file)
		file.Close()
		if string(got) != "archive" {
			t.Errorf("%s: file %q", name, got)
		}
		if _, err := src.Open(t.Context(), srv.URL+"/missing"); err == nil {
			t.Errorf("%s: a missing file opened", name)
		}
	}
	// Plain HTTP is for the loopback interface only.
	if _, err := Source("http://example.com/update.json").Manifest(t.Context(), false); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Errorf("a feed over plain HTTP: %v", err)
	}
}

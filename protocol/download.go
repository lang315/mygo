//go:build !js

package protocol

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/egoist/mygo/internal/schemehook"
)

// download serves a URL of a custom scheme into a download.
func download(d schemehook.Download) {
	rawURL := d.URL
	scheme, _, _ := strings.Cut(rawURL, ":")
	scheme = strings.ToLower(scheme)
	mu.RLock()
	handler := handlers[scheme]
	mu.RUnlock()
	if handler == nil && scheme == schemehook.FrontendScheme {
		handler = frontendHandler()
	}
	finish := d.Finished
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil || handler == nil {
		finish("", fmt.Errorf("mygo: cannot download %s", rawURL))
		return
	}
	rw := &downloadWriter{d: d, header: http.Header{}}
	if !serveDownload(handler, rw, req) {
		if rw.file != nil {
			rw.file.Close()
			os.Remove(rw.path)
		}
		finish("", fmt.Errorf("mygo: downloading %s: the handler panicked", rawURL))
		return
	}
	rw.WriteHeader(http.StatusOK) // nothing written
	if rw.file == nil {
		if rw.err != nil {
			finish("", rw.err)
		}
		return // canceled
	}
	if err := rw.file.Close(); rw.err == nil {
		rw.err = err
	}
	if rw.err != nil {
		os.Remove(rw.path)
	}
	finish(rw.path, rw.err)
}

// serveDownload runs the handler, reporting false if it panicked: a page
// can start a download, so a handler's panic must not end the app.
func serveDownload(handler http.Handler, rw http.ResponseWriter, req *http.Request) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mygo: scheme handler panic for %s: %v", req.URL, r)
			ok = false
		}
	}()
	handler.ServeHTTP(rw, req)
	return true
}

// downloadWriter writes a response into the file OnWillDownload chose.
type downloadWriter struct {
	d      schemehook.Download
	header http.Header
	wrote  bool
	path   string
	file   *os.File
	err    error
}

func (d *downloadWriter) Header() http.Header { return d.header }

func (d *downloadWriter) WriteHeader(status int) {
	if d.wrote {
		return
	}
	d.wrote = true
	if status >= 400 {
		d.err = fmt.Errorf("mygo: downloading %s: %d %s", d.d.URL, status, http.StatusText(status))
		return
	}
	name := ""
	if _, params, err := mime.ParseMediaType(d.header.Get("Content-Disposition")); err == nil {
		name = params["filename"]
	}
	if name == "" {
		if u, err := url.Parse(d.d.URL); err == nil {
			name = path.Base(u.Path)
		}
	}
	d.path = d.d.Started(name)
	if d.path == "" {
		return
	}
	d.file, d.err = os.Create(d.path)
	if d.err != nil {
		d.file = nil
	}
}

func (d *downloadWriter) Write(b []byte) (int, error) {
	d.WriteHeader(http.StatusOK)
	if d.file == nil || d.err != nil {
		return len(b), nil // canceled: drain
	}
	n, err := d.file.Write(b)
	if err != nil {
		d.err = err
	}
	return n, err
}

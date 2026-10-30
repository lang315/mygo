//go:build !js

package protocol

import (
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/schemehook"
)

// serve runs on the main thread when the webview requests a custom scheme.
func serve(req *platform.SchemeRequest, call schemehook.Call) {
	scheme, _, _ := strings.Cut(req.URL, ":")
	scheme = strings.ToLower(scheme)
	mu.RLock()
	h := handlers[scheme]
	mu.RUnlock()
	if h == nil && scheme == schemehook.FrontendScheme {
		h = frontendHandler()
	}
	go serveScheme(h, req, call)
}

func serveScheme(h http.Handler, req *platform.SchemeRequest, call schemehook.Call) {
	rw := &schemeWriter{req: req, call: call, header: http.Header{}}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mygo: panic serving %s: %v\n%s", req.URL, r, debug.Stack())
			if !rw.wroteHeader {
				rw.header = http.Header{}
				rw.WriteHeader(http.StatusInternalServerError)
			}
		}
		rw.finish()
	}()
	if h == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	body := req.Body
	if body == nil {
		body = http.NoBody
	}
	// Handlers tell the page asking with mygo.CallerPage(r.Context()).
	r, err := http.NewRequestWithContext(call.Context, req.Method, req.URL, body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Header = req.Header
	if r.Header == nil {
		r.Header = http.Header{}
	}
	r.RequestURI = r.URL.RequestURI()
	r.RemoteAddr = "mygo:" + strconv.Itoa(call.Window)
	if cl := r.Header.Get("Content-Length"); cl != "" {
		r.ContentLength, _ = strconv.ParseInt(cl, 10, 64)
	}
	if call.CORS(rw.header, r.Method, r.Header) {
		rw.WriteHeader(http.StatusNoContent) // a preflight, answered here
		return
	}
	h.ServeHTTP(rw, r)
}

// schemeWriter is the http.ResponseWriter for custom scheme requests. Body
// bytes are buffered and handed to the main thread in chunks, or written
// to backends that take them on this goroutine.
type schemeWriter struct {
	req         *platform.SchemeRequest
	call        schemehook.Call
	header      http.Header
	status      int
	wroteHeader bool
	buf         []byte
	// chunked is set once the body filled a chunk: the next ones are
	// allocated whole.
	chunked    bool
	sentHeader bool
	finished   bool
	// body takes the body from this goroutine, if the backend can
	// (platform.SchemeBodyWriter); err is its first error.
	body platform.SchemeBodyWriter
	err  error
	// queued holds a token per chunk waiting for the main thread.
	queued chan struct{}
}

const (
	schemeChunk = 256 << 10
	// A handler faster than the main thread waits once this many chunks
	// wait for it, so a large body does not pile up in memory.
	schemeQueued = 4
)

func (w *schemeWriter) Header() http.Header { return w.header }

func (w *schemeWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
}

func (w *schemeWriter) Write(p []byte) (int, error) {
	if err := w.writeErr(); err != nil {
		return 0, err
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	for n := 0; n < len(p); {
		if w.buf == nil && w.chunked {
			w.buf = make([]byte, 0, schemeChunk)
		}
		k := min(len(p)-n, schemeChunk-len(w.buf))
		w.buf = append(w.buf, p[n:n+k]...)
		n += k
		if len(w.buf) == schemeChunk {
			w.chunked = true
			w.Flush()
			if err := w.writeErr(); err != nil {
				return n, err
			}
		}
	}
	return len(p), nil
}

// writeErr reports why the body can no longer be written.
func (w *schemeWriter) writeErr() error {
	if w.err != nil {
		return w.err
	}
	return w.req.Context.Err()
}

// Flush implements http.Flusher, sending buffered data to the page.
func (w *schemeWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if !w.sentHeader {
		w.sentHeader = true
		if w.header.Get("Content-Type") == "" && w.status != http.StatusNoContent && w.status != http.StatusNotModified {
			w.header.Set("Content-Type", http.DetectContentType(w.buf))
		}
		status, header, resp := w.status, w.header.Clone(), w.req.Responder
		if body, ok := resp.(platform.SchemeBodyWriter); ok {
			// The body goes to the backend from here, once it responded.
			w.call.OnMain(func() { resp.Respond(status, header) })
			w.body = body
		} else {
			w.call.PostMain(func() { resp.Respond(status, header) })
		}
	}
	switch {
	case len(w.buf) == 0:
	case w.body != nil:
		if w.err == nil {
			w.err = w.body.WriteBody(w.buf)
		}
		w.buf = w.buf[:0] // written: the buffer is free again
	default:
		chunk, resp := w.buf, w.req.Responder
		w.buf = nil
		if w.queued == nil {
			w.queued = make(chan struct{}, schemeQueued)
		}
		w.queued <- struct{}{}
		if !w.call.PostMain(func() { resp.Write(chunk); <-w.queued }) {
			<-w.queued
		}
	}
}

func (w *schemeWriter) finish() {
	if w.finished {
		return
	}
	w.finished = true
	w.Flush()
	resp := w.req.Responder
	w.call.PostMain(resp.Finish)
}

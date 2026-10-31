//go:build !js

package protocol

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/schemehook"
)

// rec is the responder of a request, which tells what the page got.
type rec struct {
	mu       sync.Mutex
	status   int
	header   map[string][]string
	body     []byte
	finished chan struct{}
	once     sync.Once
}

func newRec() *rec { return &rec{finished: make(chan struct{})} }

func (r *rec) Respond(status int, h map[string][]string) {
	r.mu.Lock()
	r.status, r.header = status, h
	r.mu.Unlock()
}
func (r *rec) Write(p []byte) { r.mu.Lock(); r.body = append(r.body, p...); r.mu.Unlock() }
func (r *rec) Finish()        { r.once.Do(func() { close(r.finished) }) }
func (r *rec) Fail(error)     { r.Finish() }

// call is what package mygo gives the serving, with the main thread being any
// goroutine and no CORS.
func call(ctx context.Context, window int, cors func(map[string][]string, string, map[string][]string) bool) schemehook.Call {
	if cors == nil {
		cors = func(map[string][]string, string, map[string][]string) bool { return false }
	}
	return schemehook.Call{
		Context:  ctx,
		Window:   window,
		CORS:     cors,
		OnMain:   func(fn func()) { fn() },
		PostMain: func(fn func()) bool { fn(); return true },
	}
}

func serveRequest(t *testing.T, method, url string, c schemehook.Call) *rec {
	t.Helper()
	r := newRec()
	req := &platform.SchemeRequest{Context: c.Context, Method: method, URL: url, Header: map[string][]string{}, Responder: r}
	schemehook.Linked.Load().Serve(req, c)
	select {
	case <-r.finished:
	case <-time.After(time.Minute):
		t.Fatal("the request was not answered")
	}
	return r
}

type ctxKey struct{}

// TestImportingLinksTheServing: importing the package is what gives package
// mygo something to serve with.
func TestImportingLinksTheServing(t *testing.T) {
	if l := schemehook.Linked.Load(); l == nil || l.Serve == nil || l.Download == nil {
		t.Fatalf("the serving after the import: %+v", l)
	}
}

func TestHandleChecksTheScheme(t *testing.T) {
	for _, scheme := range []string{"http", "HTTPS", "file", "javascript", "Bad Scheme", "", "1app", "app_x"} {
		if err := Handle(scheme, http.NotFoundHandler()); err == nil {
			t.Errorf("Handle(%q) worked", scheme)
		}
	}
	if err := Handle("Thumbs", http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	defer Unhandle("thumbs")
	if !IsHandled("THUMBS") || !schemehook.Handled("thumbs") {
		t.Error("a handled scheme is not known (case-insensitively), to the package and to package mygo")
	}
	if !slices.Contains(schemehook.Schemes(), "thumbs") || !slices.Contains(schemehook.Schemes(), "mygo") {
		t.Errorf("the schemes of a window: %v", schemehook.Schemes())
	}
	Unhandle("Thumbs")
	if IsHandled("thumbs") || schemehook.Handled("thumbs") || slices.Contains(schemehook.Schemes(), "thumbs") {
		t.Error("an unhandled scheme is still known")
	}
	// Handling mygo is the app's frontend of its own: the scheme is listed once.
	if err := HandleFunc("mygo", func(w http.ResponseWriter, r *http.Request) {}); err != nil {
		t.Fatal(err)
	}
	defer Unhandle("mygo")
	n := 0
	for _, s := range schemehook.Schemes() {
		if s == "mygo" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("mygo is in the schemes %d times: %v", n, schemehook.Schemes())
	}
}

// TestServe: a request gets the handler of its scheme, the context and the
// window id package mygo gave it, CORS before the handler, and a 500 for a
// handler that panics.
func TestServe(t *testing.T) {
	var seen struct {
		remote string
		value  any
		calls  int
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		seen.calls++
		seen.remote, seen.value = r.RemoteAddr, r.Context().Value(ctxKey{})
		w.Header().Set("X-Test", r.URL.Query().Get("q"))
		io.WriteString(w, "<p>hi</p>")
	})
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) { panic("oops") })
	if err := Handle("servetest", mux); err != nil {
		t.Fatal(err)
	}
	defer Unhandle("servetest")

	ctx := context.WithValue(context.Background(), ctxKey{}, "caller")
	r := serveRequest(t, "GET", "servetest://localhost/hello?q=1", call(ctx, 7, nil))
	if r.status != 200 || string(r.body) != "<p>hi</p>" || r.header["X-Test"][0] != "1" {
		t.Errorf("GET: %d %q %v", r.status, r.body, r.header)
	}
	if seen.remote != "mygo:7" || seen.value != "caller" {
		t.Errorf("the handler saw RemoteAddr %q and context value %v", seen.remote, seen.value)
	}

	// A preflight is answered by package mygo's CORS policy, not the handler.
	seen.calls = 0
	preflight := call(ctx, 7, func(h map[string][]string, method string, req map[string][]string) bool {
		h["Access-Control-Allow-Origin"] = []string{"app"}
		return true
	})
	r = serveRequest(t, "OPTIONS", "servetest://localhost/hello", preflight)
	if r.status != 204 || seen.calls != 0 || r.header["Access-Control-Allow-Origin"][0] != "app" {
		t.Errorf("preflight: %d %v, handler called %d times", r.status, r.header, seen.calls)
	}

	if r = serveRequest(t, "GET", "servetest://localhost/panic", call(ctx, 7, nil)); r.status != 500 {
		t.Errorf("a panic: %d", r.status)
	}
	if r = serveRequest(t, "GET", "unknownscheme://localhost/", call(ctx, 7, nil)); r.status != 404 {
		t.Errorf("a scheme with no handler: %d", r.status)
	}
	// With no frontend, mygo:// says so.
	SetFrontend(nil)
	if r = serveRequest(t, "GET", "mygo://localhost/", call(ctx, 7, nil)); r.status != 404 || !strings.Contains(string(r.body), "No frontend") {
		t.Errorf("mygo:// without a frontend: %d %q", r.status, r.body)
	}
	SetFrontend(fstest.MapFS{"index.html": {Data: []byte("<p>app</p>")}, "app.js": {Data: []byte("go()")}})
	defer SetFrontend(nil)
	for url, want := range map[string]string{
		"mygo://localhost/":         "<p>app</p>",
		"mygo://localhost/app.js":   "go()",
		"mygo://localhost/settings": "<p>app</p>", // a route of the frontend
	} {
		if r = serveRequest(t, "GET", url, call(ctx, 7, nil)); r.status != 200 || string(r.body) != want {
			t.Errorf("%s: %d %q, want %q", url, r.status, r.body, want)
		}
	}
	if r = serveRequest(t, "GET", "mygo://localhost/missing.css", call(ctx, 7, nil)); r.status != 404 {
		t.Errorf("a file the frontend lacks: %d", r.status)
	}
}

// TestDownload: the file is saved where the window says, named by the
// handler, and the window is told; a refusal of the window ends it silently.
func TestDownload(t *testing.T) {
	if err := HandleFunc("dl", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gone" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="data.json"`)
		io.WriteString(w, `{"ok":true}`)
	}); err != nil {
		t.Fatal(err)
	}
	defer Unhandle("dl")
	dir := t.TempDir()
	run := func(url string, started func(name string) string) (path string, err error, done bool) {
		finished := make(chan struct{})
		var d schemehook.Download
		d = schemehook.Download{
			URL:     url,
			Started: started,
			Finished: func(p string, e error) {
				path, err, done = p, e, true
				close(finished)
			},
		}
		go schemehook.Linked.Load().Download(d)
		select {
		case <-finished:
		case <-time.After(2 * time.Second):
		}
		return
	}
	var name string
	path, err, done := run("dl://localhost/export", func(n string) string { name = n; return filepath.Join(dir, n) })
	if !done || err != nil || name != "data.json" || filepath.Base(path) != "data.json" {
		t.Fatalf("download: %q %v %v, asked for %q", path, err, done, name)
	}
	if b, _ := os.ReadFile(path); string(b) != `{"ok":true}` {
		t.Errorf("saved %q", b)
	}
	if _, err, done := run("dl://localhost/gone", func(string) string { return filepath.Join(dir, "x") }); !done || err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("a download that is a 404: %v %v", err, done)
	}
	if _, _, done := run("dl://localhost/export", func(string) string { return "" }); done {
		t.Error("a download the window refused was reported")
	}
	if _, err, done := run("nohandler://localhost/f", func(string) string { return "x" }); !done || err == nil {
		t.Errorf("a scheme without handler: %v %v", err, done)
	}
}

func TestFileServer(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html>index")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	if err := Handle("fs", FileServer(fsys)); err != nil {
		t.Fatal(err)
	}
	defer Unhandle("fs")
	for url, want := range map[string]string{
		"fs://localhost/":              "<!doctype html>index",
		"fs://localhost/assets/app.js": "console.log(1)",
		"fs://localhost/settings/me":   "<!doctype html>index",
	} {
		if r := serveRequest(t, "GET", url, call(context.Background(), 1, nil)); r.status != 200 || string(r.body) != want {
			t.Errorf("%s: %d %q, want %q", url, r.status, r.body, want)
		}
	}
	if r := serveRequest(t, "GET", "fs://localhost/missing.js", call(context.Background(), 1, nil)); r.status != 404 {
		t.Errorf("a missing file: %d", r.status)
	}
}

// TestHandleAndUnhandleKeepTheListsTogether: the handlers and the names that
// package mygo keeps (for the schemes of the windows and for trust) change
// together, under one lock, so a reader that holds it never sees them say
// different things while Handle and Unhandle of one scheme run in several
// goroutines.
func TestHandleAndUnhandleKeepTheListsTogether(t *testing.T) {
	var writers sync.WaitGroup
	for g := range 4 {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for i := range 20000 {
				if (i+g)%2 == 0 {
					_ = Handle("listsrace", http.NotFoundHandler())
				} else {
					Unhandle("listsrace")
				}
			}
		}()
	}
	done := make(chan struct{})
	go func() { writers.Wait(); close(done) }()
	disagreements, checks := 0, 0
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
		}
		mu.RLock()
		_, handled := handlers["listsrace"]
		named := schemehook.Handled("listsrace")
		mu.RUnlock()
		checks++
		if handled != named {
			disagreements++
		}
	}
	Unhandle("listsrace")
	if disagreements > 0 {
		t.Errorf("the handlers and the names disagreed %d times of %d", disagreements, checks)
	}
}

// TestBuildsForBrowsers: an app that imports the package builds for browsers
// (GOOS=js), where it serves nothing: it has what package mygo has there.
func TestBuildsForBrowsers(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool")
	}
	cmd := exec.Command(goTool, "build", "-o", os.DevNull, ".")
	cmd.Env = append(cmd.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build for js/wasm: %v\n%s", err, out)
	}
}

// TestProtocolDoesNotImportMygo: package mygo gives the serving what it needs
// (schemehook.Call), so this package depends on the platform's types only, and
// package mygo's own tests can import it.
func TestProtocolDoesNotImportMygo(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool")
	}
	out, err := exec.Command(goTool, "list", "-deps", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == "github.com/egoist/mygo" {
			t.Errorf("package protocol depends on package mygo")
		}
	}
}

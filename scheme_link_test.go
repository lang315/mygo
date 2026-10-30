//go:build !js

package mygo

import (
	"context"
	"io"
	"net/http"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/schemehook"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/protocol"
)

// unlinkedScheme puts the app back to where it is before it imports package
// protocol: no serving linked, no frontend. It returns the serving, which the
// test binary links by importing protocol, and puts it back when the test ends.
func unlinkedScheme(t *testing.T) *schemehook.Serving {
	t.Helper()
	linked := schemehook.Linked.Load()
	if linked == nil {
		t.Fatal("importing package protocol did not link the serving")
	}
	protocol.SetFrontend(nil)
	schemehook.Linked.Store(nil)
	t.Cleanup(func() {
		schemehook.Linked.Store(linked)
		protocol.SetFrontend(nil)
	})
	return linked
}

// corsOracle is allowCORS as it was written with net/http's types, kept to
// show that the version on maps, which serves a request of an app without
// net/http, grants the same.
func corsOracle(win *Window, header http.Header, r *http.Request) (preflight bool) {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" || win.noBindings || !win.isTrusted(appOrigin(origin)) {
		return false
	}
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Expose-Headers", "*")
	header.Add("Vary", "Origin")
	method := r.Header.Get("Access-Control-Request-Method")
	if r.Method != http.MethodOptions || method == "" {
		return false
	}
	header.Set("Access-Control-Allow-Methods", method)
	if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
		header.Set("Access-Control-Allow-Headers", h)
	}
	header.Set("Access-Control-Max-Age", "600")
	return true
}

// TestAllowCORSGrantsWhatItDidWithNetHTTP: for requests of every origin the
// app sees (its own, a custom scheme's, a trusted site, a stranger, none),
// method, header spelling and window, the headers allowCORS sets are those
// of the version on net/http's types, byte for byte.
func TestAllowCORSGrantsWhatItDidWithNetHTTP(t *testing.T) {
	if err := protocol.Handle("corsapp", http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	defer protocol.Unhandle("corsapp")
	defer func(v bool) { schemeHosts = v }(schemeHosts)

	origins := []string{"", "null", "mygo://localhost", "mygo://localhost/path", "http://mygo.localhost",
		"http://sub.mygo.localhost", "corsapp://localhost", "http://corsapp.localhost", "http://x.corsapp.localhost",
		"https://partner.example", "https://PARTNER.example", "https://evil.example", "http://localhost:5173", "file://"}
	wins := []*Window{
		{trustedOrigins: []string{"https://partner.example"}},
		{noBindings: true, trustedOrigins: []string{"https://partner.example"}},
		{untrusted: true},
	}
	spellings := []func(string) string{
		func(s string) string { return s },
		strings.ToLower,
		strings.ToUpper,
	}
	prefilled := []map[string][]string{
		{},
		{"Vary": {"Accept"}, "Access-Control-Allow-Origin": {"x"}, "access-control-max-age": {"1"}},
	}
	var granted, preflights, refused int
	for _, hosts := range []bool{false, true} {
		schemeHosts = hosts
		for _, win := range wins {
			for _, origin := range origins {
				for _, method := range []string{"GET", "OPTIONS", "POST"} {
					for _, asked := range []string{"", "PUT", "get"} {
						for _, headers := range []string{"", "X-A, content-type"} {
							for _, spell := range spellings {
								for _, pre := range prefilled {
									req := map[string][]string{}
									add := func(k, v string) {
										if v != "" {
											req[spell(k)] = []string{v}
										}
									}
									add("Origin", origin)
									add("Access-Control-Request-Method", asked)
									add("Access-Control-Request-Headers", headers)

									want := http.Header{}
									got := map[string][]string{}
									for k, v := range pre {
										want[k], got[k] = slices.Clone(v), slices.Clone(v)
									}
									wantPre := corsOracle(win, want, &http.Request{Method: method, Header: http.Header(req)})
									gotPre := allowCORS(win, got, method, req)
									if gotPre != wantPre || !reflect.DeepEqual(map[string][]string(want), got) {
										t.Fatalf("hosts=%v win=%+v %s Origin=%q ACRM=%q ACRH=%q spelled %q, prefilled %v:\nnet/http: %v %v\nmaps:     %v %v",
											hosts, win, method, origin, asked, headers, spell("Origin"), pre, wantPre, want, gotPre, got)
									}
									switch {
									case wantPre:
										preflights++
									case len(want) > len(pre):
										granted++
									default:
										refused++
									}
								}
							}
						}
					}
				}
			}
		}
	}
	// The table must have both kinds of answers, or it proves nothing.
	if preflights == 0 || granted == 0 || refused == 0 {
		t.Errorf("the table has %d preflights, %d grants and %d refusals", preflights, granted, refused)
	}
}

// schemeRec is the SchemeResponder of a request, which tells what the page
// got.
type schemeRec struct {
	mu       sync.Mutex
	status   int
	header   map[string][]string
	body     []byte
	failed   error
	finished chan struct{}
	once     sync.Once
}

func newSchemeRec() *schemeRec { return &schemeRec{finished: make(chan struct{})} }

func (r *schemeRec) Respond(status int, h map[string][]string) {
	r.mu.Lock()
	r.status, r.header = status, h
	r.mu.Unlock()
}
func (r *schemeRec) Write(p []byte) {
	r.mu.Lock()
	r.body = append(r.body, p...)
	r.mu.Unlock()
}
func (r *schemeRec) Finish()      { r.once.Do(func() { close(r.finished) }) }
func (r *schemeRec) Fail(e error) { r.failed = e; r.Finish() }

// bodyRec is a schemeRec that takes the body from the goroutine serving, as
// the backends that implement platform.SchemeBodyWriter do: its Write is
// not how the body arrives.
type bodyRec struct{ *schemeRec }

func (r bodyRec) Write([]byte) {}
func (r bodyRec) WriteBody(p []byte) error {
	r.schemeRec.Write(p)
	return nil
}

func awaitScheme(t *testing.T, r *schemeRec) {
	t.Helper()
	select {
	case <-r.finished:
	case <-time.After(wait.Guard):
		t.Fatal("the request was not answered")
	}
}

// TestUnlinkedSchemeAnswersAsServing: an app that does not import package
// protocol, so has no handler and no frontend, gives its pages what the
// serving gives an app with neither, for the requests it can get: the status,
// headers and body, to a responder that takes the body as Write or as
// WriteBody. The page of mygo:// is the exception: it says to import protocol.
func TestUnlinkedSchemeAnswersAsServing(t *testing.T) {
	linked := unlinkedScheme(t)
	defer func(v bool) { schemeHosts = v }(schemeHosts)
	win, _ := testWindow(t, WindowOptions{Page: PageOptions{TrustedOrigins: []string{"https://partner.example"}}})

	type tc struct {
		name, method, url string
		header            map[string][]string
		body              string
		hosts             bool
	}
	cases := []tc{
		{name: "index", method: "GET", url: "mygo://localhost/"},
		{name: "a file", method: "GET", url: "mygo://localhost/assets/app.js?v=1"},
		{name: "upper case scheme", method: "GET", url: "MYGO://localhost/x"},
		{name: "a post", method: "POST", url: "mygo://localhost/api", body: "data"},
		{name: "from the app", method: "GET", url: "mygo://localhost/", header: map[string][]string{"Origin": {"mygo://localhost"}}},
		{name: "from the app on windows", method: "GET", url: "mygo://localhost/", hosts: true, header: map[string][]string{"Origin": {"http://mygo.localhost"}}},
		{name: "from a trusted site", method: "GET", url: "mygo://localhost/", header: map[string][]string{"Origin": {"https://partner.example"}}},
		{name: "from a stranger", method: "GET", url: "mygo://localhost/", header: map[string][]string{"Origin": {"https://evil.example"}}},
		{name: "a preflight", method: "OPTIONS", url: "mygo://localhost/api", header: map[string][]string{
			"Origin": {"https://partner.example"}, "Access-Control-Request-Method": {"PUT"}, "Access-Control-Request-Headers": {"x-a"}}},
		{name: "an options that is no preflight", method: "OPTIONS", url: "mygo://localhost/api", header: map[string][]string{"Origin": {"https://partner.example"}}},
		{name: "a scheme without handler", method: "GET", url: "nohandler://localhost/x"},
		{name: "a url that does not parse", method: "GET", url: "mygo://localhost/%zz"},
		{name: "a url that does not parse, from the app", method: "GET", url: "mygo://localhost/%zz", header: map[string][]string{"Origin": {"mygo://localhost"}}},
		{name: "a method that is no token", method: "BAD METHOD", url: "mygo://localhost/"},
		{name: "a method with a character past ASCII", method: "G\u00c9T", url: "mygo://localhost/"},
		{name: "a preflight with a bad method", method: "OP TIONS", url: "mygo://localhost/", header: map[string][]string{
			"Origin": {"https://partner.example"}, "Access-Control-Request-Method": {"PUT"}}},
		{name: "a method of token characters", method: "M-SEARCH", url: "mygo://localhost/"},
		{name: "no method", method: "", url: "mygo://localhost/"},
		{name: "a url that does not parse, of a scheme without handler", method: "GET", url: "nohandler://localhost/%zz"},
		{name: "a bad method, of a scheme without handler", method: "BAD METHOD", url: "nohandler://localhost/"},
	}
	for _, c := range cases {
		for _, bodyWriter := range []bool{false, true} {
			schemeHosts = c.hosts
			answer := func(serve func(*platform.SchemeRequest)) *schemeRec {
				rec := newSchemeRec()
				var resp platform.SchemeResponder = rec
				if bodyWriter {
					resp = bodyRec{rec}
				}
				req := &platform.SchemeRequest{Context: context.Background(), Method: c.method, URL: c.url, Header: c.header, Responder: resp}
				if c.body != "" {
					req.Body = io.NopCloser(strings.NewReader(c.body))
				}
				onMain(func() { serve(req) })
				awaitScheme(t, rec)
				return rec
			}
			want := answer(func(req *platform.SchemeRequest) { linked.Serve(req, win.schemeCall(req)) })
			got := answer(func(req *platform.SchemeRequest) { serveUnlinked(win, req) })
			// A request of an unregistered scheme never reaches an app: the
			// backends route the ones in WindowOptions.Schemes only.
			sameBody := string(got.body) == string(want.body)
			page := len(want.body) > 0 && strings.Contains(string(want.body), "No frontend")
			if page {
				// the page of mygo:// of an app without package protocol says to import it
				sameBody = strings.Contains(string(got.body), "No frontend") && strings.Contains(string(got.body), "github.com/egoist/mygo/protocol")
			}
			if got.status != want.status || !reflect.DeepEqual(got.header, want.header) || !sameBody || got.failed != want.failed {
				t.Errorf("%s (body as WriteBody: %v):\nserving:  %d %v %q %v\nunlinked: %d %v %q %v",
					c.name, bodyWriter, want.status, want.header, want.body, want.failed, got.status, got.header, got.body, got.failed)
			}
		}
	}
}

// TestSchemeRequestOfAWindowOfAnUnlinkedApp: windowHandler takes the
// requests to the serving once package protocol is imported, and to the answer
// of an app that does not import it until then.
func TestSchemeRequestOfAWindowOfAnUnlinkedApp(t *testing.T) {
	linked := unlinkedScheme(t)
	_, fw := testWindow(t, WindowOptions{})
	ask := func() *schemeRec {
		rec := newSchemeRec()
		req := &platform.SchemeRequest{Context: context.Background(), Method: "GET", URL: "mygo://localhost/", Responder: rec}
		onMain(func() { fw.H.SchemeRequest(req) })
		awaitScheme(t, rec)
		return rec
	}
	if r := ask(); r.status != 404 || !strings.Contains(string(r.body), "No frontend") || !strings.Contains(string(r.body), "mygo/protocol") {
		t.Errorf("an app without package protocol answered %d %q", r.status, r.body)
	}
	schemehook.Linked.Store(linked)
	protocol.SetFrontend(fstest.MapFS{"index.html": {Data: []byte("<p>app</p>")}})
	if r := ask(); r.status != 200 || string(r.body) != "<p>app</p>" {
		t.Errorf("an app with a frontend answered %d %q", r.status, r.body)
	}
}

// TestUnlinkedSchemeDownloadsFailAsServing: the download of a URL of an app
// that does not import package protocol ends with the error the serving gives
// an app with nothing to serve.
func TestUnlinkedSchemeDownloadsFailAsServing(t *testing.T) {
	linked := unlinkedScheme(t)
	w, fw := testWindow(t, WindowOptions{})
	done := make(chan *Download, 1)
	defer w.Page().OnDownloadDone(func(d *Download) { done <- d })()
	finished := func() string {
		select {
		case d := <-done:
			if d.Err == nil {
				t.Fatalf("the download of %s worked: %+v", d.URL, d)
			}
			return d.Err.Error()
		case <-time.After(wait.Guard):
			t.Fatal("the download did not finish")
			return ""
		}
	}
	for _, url := range []string{"mygo://localhost/f.zip", "nohandler://localhost/f.zip", "mygo://localhost/%zz"} {
		schemehook.Linked.Store(linked)
		onMain(func() { fw.H.SchemeDownload(url) })
		want := finished()
		schemehook.Linked.Store(nil)
		onMain(func() { fw.H.SchemeDownload(url) })
		if got := finished(); got != want {
			t.Errorf("%s: unlinked %q, serving %q", url, got, want)
		}
	}
}

// TestIsTrustedNeedsNoServing: which pages may call bound methods does not
// depend on the serving being linked: the registered schemes are the app's
// whether or not package protocol serves them.
func TestIsTrustedNeedsNoServing(t *testing.T) {
	unlinkedScheme(t)
	if err := protocol.Handle("trustapp", http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	defer protocol.Unhandle("trustapp")
	win := &Window{}
	for url, want := range map[string]bool{
		"mygo://localhost/":      true,
		"about:blank":            true,
		"trustapp://localhost/x": true,
		"TRUSTAPP://localhost/x": true,
		"other://localhost/":     false,
		"https://evil.example/":  false,
	} {
		if got := win.isTrusted(url); got != want {
			t.Errorf("isTrusted(%q) = %v, want %v", url, got, want)
		}
	}
}

// TestPackageMygoDoesNotDependOnNetHTTP: the point of package protocol. Package
// mygo, and so every app of native UI, must not depend on net/http (2.2 MB):
// a file that imports it, or a package that does, brings it back. It is read
// with go list for the platforms of the backends.
func TestPackageMygoDoesNotDependOnNetHTTP(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool")
	}
	for _, goos := range []string{"darwin", "linux", "windows"} {
		cmd := exec.Command(goTool, "list", "-deps", ".")
		cmd.Env = append(cmd.Environ(), "GOOS="+goos, "CGO_ENABLED=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps (%s): %v\n%s", goos, err, out)
		}
		for _, pkg := range strings.Fields(string(out)) {
			if pkg == "net/http" || pkg == "github.com/egoist/mygo/protocol" || pkg == "github.com/egoist/mygo/netproxy" {
				t.Errorf("package mygo depends on %s on %s", pkg, goos)
			}
		}
	}
}

// TestSchemeLinkingRaces: windows ask for pages while the serving is linked
// and unlinked and the handlers change, from other goroutines, as a Tester or
// a goroutine of the app may; run with -race.
func TestSchemeLinkingRaces(t *testing.T) {
	linked := unlinkedScheme(t)
	var wins []*fake.Window
	for range 3 {
		_, fw := testWindow(t, WindowOptions{})
		wins = append(wins, fw)
	}
	var wg sync.WaitGroup
	for i := range wins {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				rec := newSchemeRec()
				req := &platform.SchemeRequest{Context: context.Background(), Method: "GET", URL: "mygo://localhost/", Responder: rec}
				onMain(func() { wins[i].H.SchemeRequest(req) })
				select {
				case <-rec.finished:
				case <-time.After(wait.Guard):
					t.Error("the request was not answered")
					return
				}
			}
		}()
	}
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch i % 3 {
			case 0:
				protocol.SetFrontend(fstest.MapFS{"index.html": {Data: []byte("x")}})
			case 1:
				_ = protocol.Handle("raceapp", http.NotFoundHandler())
				protocol.Unhandle("raceapp")
			default:
				for range 10 {
					schemehook.Linked.Store(linked)
					schemehook.Linked.Store(nil)
				}
			}
		}()
	}
	wg.Wait()
}

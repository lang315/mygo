//go:build !js

package mygo

import (
	"context"
	"fmt"
	"net/textproto"
	"net/url"
	"runtime"
	"strings"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/schemehook"
	"github.com/egoist/mygo/internal/schemehost"
)

// The serving of custom schemes and of the frontend is package protocol,
// which imports net/http and so is not imported here (2.2 MB of every app):
// it sets schemehook.Linked when an app imports it. Until then
// serveUnlinked answers.

// schemeCall is what the serving of a request of the window gets from it.
func (w *Window) schemeCall(req *platform.SchemeRequest) schemehook.Call {
	return schemehook.Call{
		// Handlers tell the page asking with CallerPage(r.Context()).
		Context: context.WithValue(context.WithValue(req.Context, callerKey{}, w.top()), pageKey{}, w.pg),
		Window:  w.id,
		CORS: func(header map[string][]string, method string, request map[string][]string) bool {
			return allowCORS(w, header, method, request)
		},
		OnMain:   onMain,
		PostMain: postMain,
	}
}

func (h *windowHandler) SchemeRequest(req *platform.SchemeRequest) {
	if l := schemehook.Linked.Load(); l != nil {
		l.Serve(req, h.w.schemeCall(req))
		return
	}
	serveUnlinked(h.w, req)
}

// serveUnlinked answers a request of an app that does not import package
// protocol, so it has no handler and no frontend, as the serving would
// without any (a test checks that the answers are the same, the page of mygo://
// apart): the only scheme such an app has is mygo, which says it has no
// frontend, and a request of another gets what a scheme without a handler
// gets.
func serveUnlinked(win *Window, req *platform.SchemeRequest) {
	resp := req.Responder
	if scheme, _, _ := strings.Cut(req.URL, ":"); !strings.EqualFold(scheme, frontendScheme) {
		resp.Respond(404, map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}})
		resp.Finish()
		return
	}
	if !validRequest(req) {
		resp.Respond(400, map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}})
		resp.Finish()
		return
	}
	header := map[string][]string{}
	if allowCORS(win, header, req.Method, req.Header) {
		resp.Respond(204, header) // a preflight, answered here
		resp.Finish()
		return
	}
	header["Content-Type"] = []string{"text/html; charset=utf-8"}
	resp.Respond(404, header)
	page := []byte(noFrontendPage)
	if body, ok := resp.(platform.SchemeBodyWriter); ok {
		// The body goes to the backend off the main thread.
		go func() {
			_ = body.WriteBody(page)
			postMain(resp.Finish)
		}()
		return
	}
	resp.Write(page)
	resp.Finish()
}

// noFrontendPage is what mygo:// answers in an app that does not import
// package protocol.
const noFrontendPage = `<!doctype html>
<meta charset="utf-8">
<title>No frontend</title>
<style>body{font:15px/1.5 system-ui,sans-serif;margin:3em;color:#444}code{font-size:90%}</style>
<h1>No frontend</h1>
<p>Run the app with <code>mygo dev</code>, which loads <code>devUrl</code> from mygo.config.ts,
or build it with <code>mygo build</code>, which embeds <code>frontendDist</code>.</p>
<p>An app built another way imports <code>github.com/egoist/mygo/protocol</code>
and calls <code>protocol.SetFrontend</code>.</p>
`

// validRequest reports whether http.NewRequest would make a request of req:
// its method is a token and its URL parses. serveScheme answers 400 to the
// others, and so does serveUnlinked, which has no net/http to ask.
func validRequest(req *platform.SchemeRequest) bool {
	for _, r := range req.Method {
		if r >= 128 || !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", r)) {
			return false
		}
	}
	_, err := url.Parse(req.URL)
	return err == nil
}

func (h *windowHandler) SchemeDownload(rawURL string) {
	if l := schemehook.Linked.Load(); l != nil {
		go l.Download(schemehook.Download{
			URL: rawURL,
			Started: func(name string) string {
				return onMainValue(func() string { return h.DownloadStarted(rawURL, name) })
			},
			Finished: func(path string, err error) {
				postMain(func() { h.DownloadFinished(rawURL, path, err) })
			},
		})
		return
	}
	// What the serving reports of an app that has no handler and no frontend.
	err := fmt.Errorf("mygo: cannot download %s", rawURL)
	if scheme, _, _ := strings.Cut(rawURL, ":"); strings.EqualFold(scheme, frontendScheme) && validRequest(&platform.SchemeRequest{URL: rawURL}) {
		err = fmt.Errorf("mygo: downloading %s: 404 Not Found", rawURL)
	}
	postMain(func() { h.DownloadFinished(rawURL, "", err) })
}

// allowCORS lets the app's own pages read responses of custom schemes,
// which are other origins than theirs: the frontend, another scheme, or
// the dev server fetching thumbs://localhost/ for example. It sets the
// CORS headers for those origins, which handlers may still override, and
// reports whether the request is a preflight it answered. Pages of other
// sites get no CORS headers, unless the window trusts them
// (PageOptions.TrustedOrigins); a view without bindings gets none.
func allowCORS(win *Window, header map[string][]string, requestMethod string, request map[string][]string) (preflight bool) {
	origin := headerGet(request, "Origin")
	if origin == "" || origin == "null" || win.noBindings || !win.isTrusted(appOrigin(origin)) {
		return false
	}
	headerSet(header, "Access-Control-Allow-Origin", origin)
	headerSet(header, "Access-Control-Expose-Headers", "*")
	key := textproto.CanonicalMIMEHeaderKey("Vary")
	header[key] = append(header[key], "Origin")
	method := headerGet(request, "Access-Control-Request-Method")
	if requestMethod != "OPTIONS" || method == "" {
		return false
	}
	headerSet(header, "Access-Control-Allow-Methods", method)
	if h := headerGet(request, "Access-Control-Request-Headers"); h != "" {
		headerSet(header, "Access-Control-Allow-Headers", h)
	}
	headerSet(header, "Access-Control-Max-Age", "600")
	return true
}

// headerGet and headerSet are http.Header's Get and Set, on the map type
// the backends and platform use, so that answering a request does not need
// net/http (serveUnlinked).
func headerGet(h map[string][]string, key string) string {
	if v := h[textproto.CanonicalMIMEHeaderKey(key)]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func headerSet(h map[string][]string, key, value string) {
	h[textproto.CanonicalMIMEHeaderKey(key)] = []string{value}
}

// schemeHosts is set where the webview serves custom schemes at
// http://[<host>.]<scheme>.localhost (WebView2). Elsewhere, such an origin
// is a server on the loopback interface, not the app's.
var schemeHosts = runtime.GOOS == "windows"

// appOrigin maps the origin WebView2 gives pages of a custom scheme,
// http://<scheme>.localhost or http://<host>.<scheme>.localhost, back to
// <scheme>://localhost or <scheme>://<host>, as the rest of MyGo names it.
func appOrigin(origin string) string {
	if !schemeHosts {
		return origin
	}
	return schemehost.AppOrigin(origin, isAppScheme)
}

func isAppScheme(scheme string) bool { return scheme == frontendScheme || schemehook.Handled(scheme) }

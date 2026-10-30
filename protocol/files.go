//go:build !js

package protocol

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
)

// The app's frontend, for SetFrontend.
var frontend struct {
	sync.Mutex
	fsys fs.FS
}

// SetFrontend serves fsys as the app's frontend at mygo://localhost/. Paths
// without a file extension that match no file serve index.html, for
// client-side routing.
//
// `mygo build` calls it with the frontendDist files it embeds into the app,
// from a file it adds that imports this package, so apps rarely need to; and
// `mygo dev` without a dev server calls it with the files on disk. Call it to
// ship a frontend without the mygo CLI, e.g. from an embed.FS.
func SetFrontend(fsys fs.FS) {
	frontend.Lock()
	frontend.fsys = fsys
	frontend.Unlock()
}

// FileServer serves files from fsys and falls back to index.html for
// unknown paths without an extension, so client side routers work. Files
// are served with their content type and support range requests. Pages
// cannot play audio and video from a custom scheme on Linux: serve them
// over loopback HTTP, with net/http and this handler (docs/frontend.md).
func FileServer(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "."
		}
		if _, err := fs.Stat(fsys, name); err != nil && path.Ext(name) == "" {
			index, err := fs.ReadFile(fsys, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		files.ServeHTTP(w, r)
	})
}

// frontendHandler serves mygo:// when the app has no handler of its own.
func frontendHandler() http.Handler {
	frontend.Lock()
	fsys := frontend.fsys
	frontend.Unlock()
	if fsys == nil {
		return http.HandlerFunc(noFrontend)
	}
	return FileServer(fsys)
}

func noFrontend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(noFrontendPage))
}

// noFrontendPage is what mygo:// answers when the app has no frontend.
const noFrontendPage = `<!doctype html>
<meta charset="utf-8">
<title>No frontend</title>
<style>body{font:15px/1.5 system-ui,sans-serif;margin:3em;color:#444}code{font-size:90%}</style>
<h1>No frontend</h1>
<p>Run the app with <code>mygo dev</code>, which loads <code>devUrl</code> from mygo.config.ts,
or build it with <code>mygo build</code>, which embeds <code>frontendDist</code>.</p>
`

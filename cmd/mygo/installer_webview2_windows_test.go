package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// webView2Server stands in for Microsoft's download: it serves an
// executable-looking file that no one signed, and counts the requests.
func webView2Server(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("MZ not signed by anyone"))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// setWebView2Test names the registry key and the download of the next
// installer builds, as tests have them, and the key's version if it is not
// "": a machine's own runtime, which the installers of these tests would
// find, does not hide the paths under test.
func setWebView2Test(t *testing.T, url, version string) {
	t.Helper()
	client, from := webView2Client, webView2URL
	t.Cleanup(func() { webView2Client, webView2URL = client, from })
	webView2Client = fmt.Sprintf("{%08X-0000-4000-8000-000000000000}", uint32(len(t.Name()))*2654435761+uint32(os.Getpid()))
	webView2URL = url
	key := `HKCU\Software\Microsoft\EdgeUpdate\Clients\` + webView2Client
	t.Cleanup(func() { exec.Command("reg", "delete", key, "/f").Run() })
	if version != "" {
		if out, err := exec.Command("reg", "add", key, "/v", "pv", "/d", version, "/f").CombinedOutput(); err != nil {
			t.Fatalf("reg add: %v\n%s", err, out)
		}
	}
}

// installWebView2Test builds the installer of an app with the given
// windows.webview2, installs it silently and returns the installer's log
// of the runtime. The app must be installed whatever happened to the
// runtime.
func installWebView2Test(t *testing.T, identifier, mode string) string {
	t.Helper()
	_, _, artifacts := buildTestInstaller(t, map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"mygo.json": `{"name": "Runtime Test", "identifier": "` + identifier + `", "version": "1.0.0", "windows": {"webview2": "` + mode + `"}}`,
	})
	install := filepath.Join(t.TempDir(), "RuntimeTest")
	log := filepath.Join(os.TempDir(), identifier+"-webview2.log")
	os.Remove(log)
	t.Cleanup(func() {
		if fileExists(filepath.Join(install, "Uninstall.exe")) {
			exec.Command(filepath.Join(install, "Uninstall.exe"), "/S").Run()
		}
		os.Remove(log)
	})
	if out, err := exec.Command(artifacts[len(artifacts)-1], "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("a silent install must succeed, whatever happens to the runtime: %v\n%s", err, out)
	}
	if !fileExists(filepath.Join(install, "Runtime Test.exe")) {
		t.Fatal("the app is not installed")
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("the installer wrote no log: %v", err)
	}
	return string(b)
}

// TestInstallerWebView2 runs the installer's WebView2 paths on Windows
// that do not need a machine without the runtime: one that has it
// downloads nothing, and a bootstrapper no one signed, downloaded or
// embedded, is not run and does not keep the app from being installed; nor
// does a download that fails. Silent installs show no message. The
// bootstrapper of Microsoft and its elevation are not tested here.
func TestInstallerWebView2(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		srv, hits := webView2Server(t)
		setWebView2Test(t, srv.URL, "1.2.3.4")
		log := installWebView2Test(t, "com.example.rtfound", "download")
		if !strings.Contains(log, "WebView2 Runtime found: 1.2.3.4") {
			t.Errorf("log %q: want the runtime found", log)
		}
		if n := hits.Load(); n != 0 {
			t.Errorf("the installer downloaded %d times with a runtime installed", n)
		}
	})
	t.Run("unsigned download", func(t *testing.T) {
		srv, hits := webView2Server(t)
		setWebView2Test(t, srv.URL, "")
		log := installWebView2Test(t, "com.example.rtunsigned", "download")
		if !strings.Contains(log, "not installed: signature ") {
			t.Errorf("log %q: want the signature refused", log)
		}
		if n := hits.Load(); n != 1 {
			t.Errorf("%d downloads, want 1", n)
		}
	})
	t.Run("unsigned embedded", func(t *testing.T) {
		srv, hits := webView2Server(t)
		setWebView2Test(t, srv.URL, "")
		fetch := fetchWebView2Setup
		t.Cleanup(func() { fetchWebView2Setup = fetch })
		fetchWebView2Setup = func(dir string) (string, error) {
			p := filepath.Join(dir, "MicrosoftEdgeWebview2Setup.exe")
			return p, os.WriteFile(p, []byte("MZ not signed by anyone"), 0o644)
		}
		log := installWebView2Test(t, "com.example.rtembed", "embed")
		if !strings.Contains(log, "not installed: signature ") {
			t.Errorf("log %q: want the signature refused", log)
		}
		if n := hits.Load(); n != 0 {
			t.Errorf("the installer downloaded %d times with the bootstrapper embedded", n)
		}
	})
	t.Run("download fails", func(t *testing.T) {
		srv, hits := webView2Server(t)
		srv.Close()
		setWebView2Test(t, srv.URL, "")
		log := installWebView2Test(t, "com.example.rtdown", "download")
		if !strings.Contains(log, "not installed: download failed") {
			t.Errorf("log %q: want the download to fail", log)
		}
		if n := hits.Load(); n != 0 {
			t.Errorf("%d requests reached a closed server", n)
		}
	})
	t.Run("none", func(t *testing.T) {
		srv, hits := webView2Server(t)
		setWebView2Test(t, srv.URL, "")
		_, _, artifacts := buildTestInstaller(t, map[string]string{
			"main.go":   "package main\n\nfunc main() {}\n",
			"mygo.json": `{"name": "Runtime Test", "identifier": "com.example.rtnone", "version": "1.0.0", "windows": {"webview2": "none"}}`,
		})
		install := filepath.Join(t.TempDir(), "RuntimeTest")
		t.Cleanup(func() { exec.Command(filepath.Join(install, "Uninstall.exe"), "/S").Run() })
		if out, err := exec.Command(artifacts[len(artifacts)-1], "/S", "/D="+install).CombinedOutput(); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		if n := hits.Load(); n != 0 || fileExists(filepath.Join(os.TempDir(), "com.example.rtnone-webview2.log")) {
			t.Errorf("none touched the runtime: %d downloads", n)
		}
	})
}

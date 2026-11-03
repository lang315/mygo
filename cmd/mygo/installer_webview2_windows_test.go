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

	"github.com/egoist/mygo/internal/wait"
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
	t.Run("signed by another", func(t *testing.T) {
		// A file of Windows with a valid signature of another signer than
		// Microsoft Corporation, as the bootstrapper of someone else.
		other := filepath.Join(t.TempDir(), "setup.exe")
		src, err := os.ReadFile(filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(other, src, 0o755); err != nil {
			t.Fatal(err)
		}
		if got := signatureOf(t, other); !strings.HasPrefix(got, "Valid|") || got == "Valid|Microsoft Corporation" {
			t.Skipf("cmd.exe is signed %q here, not by another signer", got)
		}
		srv, _ := webView2Server(t)
		setWebView2Test(t, srv.URL, "")
		fetch := fetchWebView2Setup
		t.Cleanup(func() { fetchWebView2Setup = fetch })
		fetchWebView2Setup = func(string) (string, error) { return other, nil }
		log := installWebView2Test(t, "com.example.rtother", "embed")
		if !strings.Contains(log, "not installed: signed by ") {
			t.Errorf("log %q: want the signer refused", log)
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

// signatureOf returns "<status>|<signer>" of an executable's Authenticode
// signature.
func signatureOf(t *testing.T, path string) string {
	t.Helper()
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"$s = Get-AuthenticodeSignature -LiteralPath '"+path+"'; $s.Status.ToString() + '|' + $s.SignerCertificate.GetNameInfo('SimpleName', $false)").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// runWebView2Step runs the step that holds the bootstrapper and runs it
// (webView2HoldPS1 and webView2RunPS1, those of the installer) on a script
// of cmd.exe in PowerShell 5.1, which is the installer's, and returns its
// output and exit code.
func runWebView2Step(t *testing.T, script string) (string, int) {
	t.Helper()
	dir := t.TempDir()
	bat := filepath.Join(dir, "setup.bat")
	if err := os.WriteFile(bat, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	ps1 := filepath.Join(dir, "step.ps1")
	if err := os.WriteFile(ps1, []byte("$file = $env:MYGO_WV2_FILE\n"+webView2HoldPS1+webView2RunPS1), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", ps1)
	cmd.Env = append(os.Environ(), "MYGO_WV2_FILE="+bat)
	type result struct {
		out  []byte
		code int
	}
	done := make(chan result, 1)
	go func() {
		out, err := cmd.CombinedOutput()
		code := 0
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else if err != nil {
			code = -1
		}
		done <- result{out, code}
	}()
	r := wait.For(t, "the step to end while the bootstrapper's child lives", done)
	return string(r.out), r.code
}

// TestWebView2RunDoesNotWaitForChildren has a stand-in for the
// bootstrapper start a child that lives on, as the updater it leaves
// running, and exit with 0: the step ends with 0 at once, and does not
// wait for the child, as Start-Process -Wait does, which would hold the
// installer, a silent update's too.
func TestWebView2RunDoesNotWaitForChildren(t *testing.T) {
	const marker = "-n 97 127.0.0.1"
	t.Cleanup(func() {
		exec.Command("powershell", "-NoProfile", "-Command",
			"Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -like '*"+marker+"*' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }").Run()
	})
	out, code := runWebView2Step(t, "@echo off\r\nstart \"\" /b ping "+marker+" >nul\r\nexit /b 0\r\n")
	if code != 0 {
		t.Errorf("exit code %d, output %q, want 0", code, out)
	}
	alive, _ := exec.Command("powershell", "-NoProfile", "-Command",
		"@(Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -like '*"+marker+"*' }).Count").Output()
	if strings.TrimSpace(string(alive)) == "0" {
		t.Error("the child did not outlive the stand-in: the test proves nothing")
	}
	// The bootstrapper's own failure is the step's.
	out, code = runWebView2Step(t, "@echo off\r\nexit /b 3\r\n")
	if code != 4 || !strings.Contains(out, "bootstrapper exit code 3") {
		t.Errorf("a failing bootstrapper: exit code %d, output %q, want 4 and its code", code, out)
	}
}

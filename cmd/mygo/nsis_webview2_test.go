package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWebView2Config checks windows.webview2: "download" by default, a
// value of the list, and "none" always for an app of native UI.
func TestWebView2Config(t *testing.T) {
	for _, tc := range []struct{ json, want, err string }{
		{`{}`, "download", ""},
		{`{"windows": {"webview2": "embed"}}`, "embed", ""},
		{`{"windows": {"webview2": "none"}}`, "none", ""},
		{`{"nativeOnly": true}`, "none", ""},
		{`{"nativeOnly": true, "windows": {"webview2": "embed"}}`, "none", ""},
		{`{"windows": {"webview2": "bundle"}}`, "", `windows.webview2 is "download", "embed" or "none", not "bundle"`},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, jsonConfig), []byte(tc.json), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := loadConfig(dir)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%s: error %v, want %q", tc.json, err, tc.err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", tc.json, err)
		}
		if c.Windows.WebView2 != tc.want {
			t.Errorf("%s: windows.webview2 is %q, want %q", tc.json, c.Windows.WebView2, tc.want)
		}
	}
}

// TestNSISWebView2 checks the installer's text for each windows.webview2:
// the check of the registry in the three places, the bootstrapper's
// signature checked before it runs, the call after the app's files and
// before the app's own steps, and nothing at all for "none".
func TestNSISWebView2(t *testing.T) {
	script := func(mode string) string {
		t.Helper()
		c := &Config{Name: "Notes", Identifier: "com.example.notes", Version: "1.0.0", Windows: Windows{WebView2: mode, InstallerInclude: ""}, webView2Setup: `C:\work\MicrosoftEdgeWebview2Setup.exe`}
		s, err := installerScript(c, `C:\out\setup.exe`, "Notes.exe", "  File \"C:\\stage\\Notes.exe\"\n", "", "")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	common := []string{
		`Function EnsureWebView2`,
		`Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`,
		`SetRegView 32`, `SetRegView 64`, `ReadRegStr $0 HKLM`, `ReadRegStr $0 HKCU`,
		`"pv"`, `"0.0.0.0"`,
		`Get-AuthenticodeSignature`, `-ne 'Valid'`, `'Microsoft Corporation'`, `'/silent', '/install'`,
		`/SD IDNO`, `"$TEMP\com.example.notes-webview2.log"`,
		"  Call EnsureWebView2\n",
	}
	for mode, extra := range map[string][]string{
		"download": {`"MYGO_WV2_URL", t "https://go.microsoft.com/fwlink/p/?LinkId=2124703"`},
		"embed":    {`File "/oname=wv2setup.exe" "C:\work\MicrosoftEdgeWebview2Setup.exe"`, `"MYGO_WV2_URL", t ""`},
		"":         {`"MYGO_WV2_URL", t "https://go.microsoft.com/fwlink/p/?LinkId=2124703"`}, // the default
	} {
		s := script(mode)
		for _, want := range append(common, extra...) {
			if !strings.Contains(s, want) {
				t.Errorf("%q: no %q in the script", mode, want)
			}
		}
		// After the files, so that a failure leaves the app installed.
		if i, j := strings.Index(s, "File \"C:\\stage\\Notes.exe\""), strings.Index(s, "  Call EnsureWebView2"); i < 0 || j < i {
			t.Errorf("%q: the call is not after the app's files", mode)
		}
		if strings.Contains(s, "File \"/oname=wv2setup.exe\"") != (mode == "embed") {
			t.Errorf("%q: the bootstrapper is carried in the installer only with embed", mode)
		}
	}
	none := script("none")
	for _, s := range []string{"EnsureWebView2", "EdgeUpdate", "webview2", "Authenticode"} {
		if strings.Contains(none, s) {
			t.Errorf("windows.webview2 none leaves %q in the script", s)
		}
	}
}

// TestWebView2Compiles has makensis compile the installer of each mode,
// without warnings, with a bootstrapper that tests stand in for.
func TestWebView2Compiles(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles an installer")
	}
	tool, err := nsisCompiler()
	if err != nil {
		t.Fatal(err)
	} else if tool == "" {
		t.Skip("NSIS is not installed")
	}
	fetch := fetchWebView2Setup
	t.Cleanup(func() { fetchWebView2Setup = fetch })
	fetchWebView2Setup = func(dir string) (string, error) {
		p := filepath.Join(dir, "MicrosoftEdgeWebview2Setup.exe")
		return p, os.WriteFile(p, []byte("not a bootstrapper"), 0o644)
	}
	for _, mode := range []string{"download", "embed", "none"} {
		c := project(t, `{"name": "Notes", "identifier": "com.example.notes", "version": "1.0.0", "windows": {"webview2": "`+mode+`"}}`, nil)
		stage, work := t.TempDir(), t.TempDir()
		writeFiles(t, stage, map[string]string{"Notes.exe": "app"})
		if _, err := writeInstaller(c, stage, work, "Notes.exe", "amd64", []string{"Notes.exe"}); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
	}
}

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// TestUpdaterSmoke is the update of an installed app on Windows, with the
// files mygo build writes and the code of package updates: the app that the
// installer put in a folder checks a local server for an update, downloads
// it, replaces its own files while it runs (the running executable is
// renamed away), relaunches into the new version, which removes what the
// update left, and Settings > Apps lists the new version. An update signed
// with another key, an archive that was changed on the way and a manifest
// of a version that is not newer install nothing and leave nothing; the
// last downloads no archive.
//
// The app is cmd/mygo/testdata/updatesmoke. Its updates.url is an HTTPS host
// that does not exist: mygo build refuses another scheme, and the app's
// source (updates.SetSource) reads the files the manifest names from the
// local server. The feed that mygo.json configures, fetched over HTTPS, is
// not covered.
//
// It runs only with MYGO_E2E=1, on windows/amd64.
func TestUpdaterSmoke(t *testing.T) {
	if os.Getenv("MYGO_E2E") != "1" {
		t.Skip("opens windows: set MYGO_E2E=1 on Windows")
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("the installer is built for amd64")
	}
	app, err := os.ReadFile(filepath.Join("testdata", "updatesmoke", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	const identifier = "com.example.updatesmoke"
	type signer struct {
		public  string
		private ed25519.PrivateKey
	}
	newKey := func() signer {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return signer{base64.StdEncoding.EncodeToString(pub), priv}
	}
	keyA, keyB := newKey(), newKey()
	// build returns the installer of the app of version, whose updates are
	// signed with the key (mygo build checks that the app's updates.publicKey
	// is the key's), and the directory that holds the archive and the
	// manifest.
	build := func(version string, k signer) (setup, files string) {
		t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(k.private))
		_, _, artifacts := buildTestInstaller(t, map[string]string{
			"main.go":   string(app),
			"mygo.json": `{"name": "Update Smoke", "identifier": "` + identifier + `", "version": "` + version + `", "updates": {"publicKey": "` + k.public + `", "url": "https://updates.example.test/app", "deltas": 0}}`,
		})
		archive := slices.IndexFunc(artifacts, func(a string) bool { return strings.HasSuffix(a, ".tar.gz") })
		manifest := slices.IndexFunc(artifacts, func(a string) bool { return filepath.Base(a) == "update-windows-amd64.json" })
		if archive < 0 || manifest < 0 || filepath.Dir(artifacts[archive]) != filepath.Dir(artifacts[manifest]) {
			t.Fatalf("artifacts of %s = %q, want an archive and update-windows-amd64.json in one directory", version, artifacts)
		}
		return artifacts[len(artifacts)-1], filepath.Dir(artifacts[archive])
	}
	// The installed app trusts key A; the third build is an update that only
	// key B signed.
	setup1, files1 := build("1.0.0", keyA)
	_, files2 := build("2.0.0", keyA)
	_, filesWrongKey := build("2.0.0", keyB)

	// The archive of 2.0.0 with one byte changed: as long as before, so
	// that it is its signature that refuses it.
	filesChanged := t.TempDir()
	archives, _ := filepath.Glob(filepath.Join(files2, "*.tar.gz"))
	if len(archives) != 1 {
		t.Fatalf("archives of 2.0.0: %q", archives)
	}
	for _, name := range []string{"update-windows-amd64.json", filepath.Base(archives[0])} {
		b, err := os.ReadFile(filepath.Join(files2, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".tar.gz") {
			b[len(b)/2] ^= 0xff
		}
		if err := os.WriteFile(filepath.Join(filesChanged, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The server of the update files, whose directory the test changes, and
	// the paths it was asked for.
	var served atomic.Pointer[string]
	var mu sync.Mutex
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, path.Base(r.URL.Path))
		mu.Unlock()
		http.ServeFile(w, r, filepath.Join(*served.Load(), path.Base(r.URL.Path)))
	}))
	t.Cleanup(srv.Close)
	serve := func(dir string) {
		served.Store(&dir)
		mu.Lock()
		requests = nil
		mu.Unlock()
	}

	install := filepath.Join(t.TempDir(), "UpdateSmoke")
	exe := filepath.Join(install, "Update Smoke.exe")
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\` + identifier
	displayVersion := func() string {
		out, _ := exec.Command("reg", "query", key, "/v", "DisplayVersion").CombinedOutput()
		return string(out)
	}
	t.Cleanup(func() {
		if fileExists(filepath.Join(install, "Uninstall.exe")) {
			exec.Command(filepath.Join(install, "Uninstall.exe"), "/S").Run()
		}
	})
	if out, err := exec.Command(setup1, "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("installing: %v\n%s", err, out)
	}
	if !strings.Contains(displayVersion(), "1.0.0") {
		t.Fatalf("Settings > Apps lists %q, want 1.0.0", displayVersion())
	}

	// names lists the app's folder and the folder around it, where an update
	// works.
	names := func() []string {
		var all []string
		for _, dir := range []string{install, filepath.Dir(install)} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				all = append(all, filepath.Join(filepath.Base(dir), e.Name()))
			}
		}
		return all
	}
	// run starts the installed app and returns the lines it reports; for the
	// attempts that end with the app quitting.
	run := func(out string) *smokeApp {
		return startApp(t, exe, "MYGO_SMOKE_OUT="+out, "MYGO_SMOKE_SERVER="+srv.URL+"/")
	}
	lines := func(out string) string {
		b, _ := os.ReadFile(out)
		return string(b)
	}
	before := fileHash(t, exe)

	// An update that is refused leaves the app as it was.
	for _, c := range []struct{ name, dir string }{
		{"signed with another key", filesWrongKey},
		{"changed on the way", filesChanged},
	} {
		t.Run(c.name, func(t *testing.T) {
			serve(c.dir)
			listed, out := names(), filepath.Join(t.TempDir(), "out.txt")
			if err := wait.For(t, "the app to quit", run(out).done); err != nil {
				t.Errorf("the app quit with %v", err)
			}
			got := lines(out)
			for _, want := range []string{"started 1.0.0 enabled=true", "found 2.0.0", "error install: ", "not signed with the app's key"} {
				if !strings.Contains(got, want) {
					t.Errorf("the app reported %q, want %q", got, want)
				}
			}
			if strings.Contains(got, "installed") {
				t.Errorf("the app reported %q", got)
			}
			if fileHash(t, exe) != before || !slices.Equal(names(), listed) {
				t.Errorf("the refused update changed the app's folder: %q, was %q", names(), listed)
			}
			if !strings.Contains(displayVersion(), "1.0.0") {
				t.Errorf("Settings > Apps lists %q, want 1.0.0", displayVersion())
			}
		})
	}

	// A manifest of the version that runs: nothing is downloaded.
	t.Run("not newer", func(t *testing.T) {
		serve(files1)
		out := filepath.Join(t.TempDir(), "out.txt")
		if err := wait.For(t, "the app to quit", run(out).done); err != nil {
			t.Errorf("the app quit with %v", err)
		}
		if got := lines(out); !strings.Contains(got, "uptodate") || strings.Contains(got, "found") {
			t.Errorf("the app reported %q", got)
		}
		mu.Lock()
		defer mu.Unlock()
		if !slices.Equal(requests, []string{"update-windows-amd64.json"}) {
			t.Errorf("the server was asked for %q, want only the manifest", requests)
		}
		if fileHash(t, exe) != before {
			t.Error("the app changed")
		}
	})

	// The update: installed while the app runs, then the new version.
	serve(files2)
	out := filepath.Join(t.TempDir(), "out.txt")
	first := run(out)
	wait.Until(t, "the app to relaunch into 2.0.0", func() bool {
		return strings.Contains(lines(out), "started 2.0.0 enabled=true")
	})
	wait.For(t, "the old app to quit", first.done)
	got := lines(out)
	for _, want := range []string{"started 1.0.0 enabled=true", "found 2.0.0", "installed", "uptodate"} {
		if !strings.Contains(got, want) {
			t.Errorf("the apps reported %q, want %q", got, want)
		}
	}
	if fileHash(t, exe) == before {
		t.Error("the update did not replace the executable")
	}
	if !strings.Contains(displayVersion(), "2.0.0") {
		t.Errorf("Settings > Apps lists %q, want 2.0.0", displayVersion())
	}
	// The new version removes what the update could not while the old one ran.
	wait.Until(t, "the update's leftovers to go", func() bool {
		for _, n := range names() {
			if strings.Contains(n, ".old-") || strings.Contains(n, ".update-") {
				return false
			}
		}
		return true
	})
	// The relaunched app quits by itself; the uninstaller quits it if it has
	// not yet.
	uninstall(t, install)
	if out, err := exec.Command("reg", "query", key).CombinedOutput(); err == nil {
		t.Errorf("the uninstaller left %s:\n%s", key, out)
	}
}

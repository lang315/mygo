package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// twoVersionProxy is a module proxy (a directory for GOPROXY=file://) of MyGo
// with v1.0.0, which has no package protocol, and v1.1.0, which has.
func twoVersionProxy(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	at := filepath.Join(dir, "github.com", "egoist", "mygo", "@v")
	if err := os.MkdirAll(at, 0o755); err != nil {
		t.Fatal(err)
	}
	gomod := "module github.com/egoist/mygo\n\ngo " + goVersion() + "\n"
	list := ""
	for _, v := range []string{"v1.0.0", "v1.1.0"} {
		files := map[string]string{"go.mod": gomod, "mygo.go": "package mygo\n"}
		if v == "v1.1.0" {
			files["protocol/protocol.go"] = "package protocol\n"
		}
		var zipped bytes.Buffer
		zw := zip.NewWriter(&zipped)
		for name, content := range files {
			w, err := zw.Create("github.com/egoist/mygo@" + v + "/" + name)
			if err != nil {
				t.Fatal(err)
			}
			io.WriteString(w, content)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		for name, content := range map[string]string{
			v + ".info": `{"Version":"` + v + `","Time":"2024-01-01T00:00:00Z"}`,
			v + ".mod":  gomod,
			v + ".zip":  zipped.String(),
		} {
			if err := os.WriteFile(filepath.Join(at, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		list += v + "\n"
	}
	if err := os.WriteFile(filepath.Join(at, "list"), []byte(list), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// coldCache points go at the proxy with an empty module cache, which go makes
// read-only: the test lets it be removed.
func coldCache(t *testing.T) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("GOMODCACHE", cache)
	t.Cleanup(func() {
		filepath.WalkDir(cache, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				os.Chmod(path, 0o755)
			}
			return nil
		})
	})
}

// fileURL is the GOPROXY URL of a directory: file:///C:/dir on Windows.
func fileURL(dir string) string {
	dir = filepath.ToSlash(dir)
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	return "file://" + dir
}

func applicationEnv(t *testing.T, flags string) {
	t.Helper()
	t.Setenv("GOFLAGS", flags)
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "")
	t.Setenv("GOPROXY", fileURL(twoVersionProxy(t)))
}

// requiringApp writes an app that requires MyGo at version (and extra lines of
// go.mod), with its go.sum, found with a warm module cache; the cache is cold
// when it returns.
func requiringApp(t *testing.T, root, version, extra string) *Config {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"go.mod":  "module demo\n\ngo " + goVersion() + "\n\nrequire github.com/egoist/mygo " + version + "\n" + extra,
		"main.go": "package main\n\nimport _ \"github.com/egoist/mygo\"\n\nfunc main() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(extra, "replace") {
		coldCache(t) // warm: this one fills go.sum
		if out, err := goOutput(root, "mod", "download", "github.com/egoist/mygo"); err != nil {
			t.Fatalf("go mod download: %v\n%s", err, out)
		}
		coldCache(t)
	}
	return &Config{root: root}
}

// TestAppHasPackageWithAColdModuleCache: go list -m gives no directory for a
// module that is not in the module cache yet. The module is fetched at the
// version the app requires, whichever -mod is in effect, and nothing is raised:
// the proxy has a v1.1.0 that has the package, the app requires v1.0.0, which
// has not, and go.mod stays as it was.
func TestAppHasPackageWithAColdModuleCache(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	for _, flags := range []string{"", "-mod=readonly", "-mod=mod"} {
		applicationEnv(t, flags)
		c := requiringApp(t, t.TempDir(), "v1.0.0", "")
		before, _ := os.ReadFile(filepath.Join(c.root, "go.mod"))
		if out, err := goOutput(c.root, "list", "-m", "-f", "{{.Dir}}", mygoModule); err != nil || strings.TrimSpace(out) != "" {
			t.Fatalf("GOFLAGS=%q: the module cache is not cold: go list -m gave %q, %v", flags, out, err)
		}
		has, err := appHasPackage(c, protocolPackage)
		if err != nil || has {
			t.Errorf("GOFLAGS=%q, MyGo v1.0.0 without protocol, v1.1.0 on the proxy: %v, %v; want false, nil", flags, has, err)
		}
		if after, _ := os.ReadFile(filepath.Join(c.root, "go.mod")); !bytes.Equal(before, after) {
			t.Errorf("GOFLAGS=%q: appHasPackage changed go.mod:\n%s\n->\n%s", flags, before, after)
		}
		if out, _ := goOutput(c.root, "list", "-m", "github.com/egoist/mygo"); !strings.Contains(out, "v1.0.0") {
			t.Errorf("GOFLAGS=%q: the app builds with %q, not v1.0.0", flags, strings.TrimSpace(out))
		}
	}
	// The version that has the package.
	applicationEnv(t, "")
	c := requiringApp(t, t.TempDir(), "v1.1.0", "")
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("MyGo v1.1.0: %v, %v", has, err)
	}
	// A module that cannot be fetched is an error, not a no.
	applicationEnv(t, "")
	c = requiringApp(t, t.TempDir(), "v1.0.0", "")
	t.Setenv("GOPROXY", fileURL(filepath.Join(t.TempDir(), "nothing")))
	moduleDirs.m = map[string]moduleDirEntry{}
	if has, err := appHasPackage(c, protocolPackage); err == nil || has {
		t.Errorf("a module that cannot be fetched: %v, %v", has, err)
	}
}

// TestAppHasPackageIgnoresAVendorDirectoryUnderMod: with -mod=mod, a vendor
// directory next to go.mod is not the module; without it, it is.
func TestAppHasPackageIgnoresAVendorDirectoryUnderMod(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	applicationEnv(t, "-mod=mod")
	c := requiringApp(t, t.TempDir(), "v1.1.0", "")
	if err := os.MkdirAll(filepath.Join(c.root, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.root, "vendor", "modules.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("-mod=mod with a stale vendor directory: %v, %v", has, err)
	}
	t.Setenv("GOFLAGS", "")
	if has, err := appHasPackage(c, protocolPackage); err == nil || has {
		t.Errorf("a vendor directory that is the module and lacks the package: %v, %v", has, err)
	}
}

// localMygo writes a MyGo with package protocol in base/mygo.
func localMygo(t *testing.T, base string, withProtocol bool) string {
	t.Helper()
	local := filepath.Join(base, "mygo")
	files := map[string]string{
		"go.mod":  "module github.com/egoist/mygo\n\ngo " + goVersion() + "\n",
		"mygo.go": "package mygo\n",
	}
	if withProtocol {
		files["protocol/protocol.go"] = "package protocol\n"
	}
	for name, content := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(local, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(local, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return local
}

// TestAppHasPackageSeesAReplacedDirectoryChange: a MyGo that is a directory
// (a replace) gains the package while go.mod stays, as in a git checkout
// during `mygo dev`: only the directory of the module is remembered, and the
// package is looked for in it every time.
func TestAppHasPackageSeesAReplacedDirectoryChange(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	applicationEnv(t, "")
	base := t.TempDir()
	local := localMygo(t, base, false)
	c := requiringApp(t, filepath.Join(base, "app"), "v0.0.0", "\nreplace github.com/egoist/mygo => ../mygo\n")
	if has, err := appHasPackage(c, protocolPackage); err != nil || has {
		t.Fatalf("before: %v, %v", has, err)
	}
	if err := os.MkdirAll(filepath.Join(local, "protocol"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "protocol", "protocol.go"), []byte("package protocol\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("after the directory gained protocol: %v, %v", has, err)
	}
}

// TestAppHasPackageFollowsAWorkspace: a go.work added during the run, whose
// MyGo is a local directory with the package, changes the answer; and in
// workspace mode go ignores the vendor directory of the app's module, so a
// vendor directory without the package is no error there.
func TestAppHasPackageFollowsAWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	applicationEnv(t, "")
	base := t.TempDir()
	localMygo(t, base, true)
	c := requiringApp(t, filepath.Join(base, "app"), "v1.0.0", "")
	if has, err := appHasPackage(c, protocolPackage); err != nil || has {
		t.Fatalf("before go.work, MyGo v1.0.0: %v, %v", has, err)
	}
	work := "go " + goVersion() + "\n\nuse (\n\t./app\n\t./mygo\n)\n"
	if err := os.WriteFile(filepath.Join(base, "go.work"), []byte(work), 0o644); err != nil {
		t.Fatal(err)
	}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("after go.work: %v, %v", has, err)
	}
	// A vendor directory of the module, from v1.0.0.
	if out, err := goOutput(c.root, "mod", "vendor"); err != nil {
		t.Logf("go mod vendor in a workspace: %v\n%s", err, out)
		t.Setenv("GOWORK", "off")
		if out, err := goOutput(c.root, "mod", "vendor"); err != nil {
			t.Fatalf("go mod vendor: %v\n%s", err, out)
		}
		t.Setenv("GOWORK", "")
	}
	if _, err := os.Stat(filepath.Join(c.root, "vendor", "modules.txt")); err != nil {
		t.Fatal("no vendor directory to ignore")
	}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("a workspace over a module with a vendor directory: %v, %v", has, err)
	}
	// Without the workspace, that vendor directory is the module, and lacks it.
	t.Setenv("GOWORK", "off")
	if has, err := appHasPackage(c, protocolPackage); err == nil || has {
		t.Errorf("GOWORK=off with that vendor directory: %v, %v", has, err)
	}
}

// goCounter puts a go in front of the real one that counts the commands it
// runs in a file, and returns the lines that match one of the words.
func goCounter(t *testing.T) func(words ...string) int {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a shell script stands for go")
	}
	real, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool")
	}
	dir, log := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\necho \"$@\" >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func(words ...string) int {
		data, _ := os.ReadFile(log)
		n := 0
		for _, line := range strings.Split(string(data), "\n") {
			for _, w := range words {
				if strings.HasPrefix(line, w) {
					n++
					break
				}
			}
		}
		return n
	}
}

// TestAppHasPackageAsksGoForTheModuleOnce: the builds of `mygo dev` ask go for
// the directory of MyGo's module (list -m, mod download) once while go.mod,
// go.sum and go.work stay as they were, and again once one is written.
func TestAppHasPackageAsksGoForTheModuleOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	applicationEnv(t, "-mod=mod")
	c := requiringApp(t, t.TempDir(), "v1.1.0", "")
	moduleDirs.m = map[string]moduleDirEntry{}
	calls := goCounter(t)
	for i := range 4 {
		if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
			t.Fatalf("answer %d: %v, %v", i, has, err)
		}
	}
	if n := calls("list -m"); n != 1 {
		t.Errorf("go list -m was run %d times in 4 answers, want 1", n)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(c.root, "go.mod"), later, later); err != nil {
		t.Fatal(err)
	}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Fatalf("after go.mod was written: %v, %v", has, err)
	}
	if n := calls("list -m"); n != 2 {
		t.Errorf("go list -m was run %d times after go.mod was written, want 2", n)
	}
}

// workspaceApp writes a workspace (go.work next to the app) over an app that
// requires MyGo at version, and vendors it with go work vendor; imports are
// the lines of the app's main.go imports.
func workspaceApp(t *testing.T, base, version, imports string) *Config {
	t.Helper()
	applicationEnv(t, "")
	c := requiringApp(t, filepath.Join(base, "app"), version, "")
	main := "package main\n\nimport _ \"github.com/egoist/mygo\"\n" + imports + "\nfunc main() {}\n"
	for name, content := range map[string]string{
		filepath.Join(c.root, "main.go"): main,
		filepath.Join(base, "go.work"):   "go " + goVersion() + "\n\nuse ./app\n",
	} {
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := goOutput(base, "work", "vendor"); err != nil {
		t.Fatalf("go work vendor: %v\n%s", err, out)
	}
	return c
}

// TestAppHasPackageVendorsOfAWorkspace: go work vendor puts the vendor
// directory next to go.work, not next to the module's go.mod, and a build in
// the workspace uses it (go 1.22 and later). It holds what the app's code
// imports, as go mod vendor does, so a MyGo that has protocol in the module
// cache is no answer: the vendor directory has to.
func TestAppHasPackageVendorsOfAWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	for _, flags := range []string{"", "-mod=vendor"} {
		base := t.TempDir()
		c := workspaceApp(t, base, "v1.1.0", "")
		applicationEnv(t, flags)
		moduleDirs.m = map[string]moduleDirEntry{}
		_, err := appHasPackage(c, protocolPackage)
		if err == nil || !strings.Contains(err.Error(), "go work vendor") || !strings.Contains(err.Error(), filepath.Join(base, "vendor")) {
			t.Errorf("GOFLAGS=%q, a workspace vendor directory without protocol: %v; want an error that says go work vendor and where", flags, err)
		}

		// The app imports the package; go work vendor copies it.
		c = workspaceApp(t, base, "v1.1.0", "import _ \"github.com/egoist/mygo/protocol\"\n")
		applicationEnv(t, flags)
		moduleDirs.m = map[string]moduleDirEntry{}
		if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
			t.Errorf("GOFLAGS=%q, a workspace vendor directory with protocol: %v, %v", flags, has, err)
		}
	}
}

// TestAppHasPackageWorkspaceModuleIsNotVendored: go work vendor copies the
// dependencies of the workspace, not its modules; a MyGo that the workspace
// uses as a directory is built from there, with a vendor directory or not. It
// holds on the base too (a workspace was never vendor mode there): it guards the
// vendor mode of a workspace against reading that directory.
func TestAppHasPackageWorkspaceModuleIsNotVendored(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	applicationEnv(t, "")
	base := t.TempDir()
	localMygo(t, base, true)
	c := requiringApp(t, filepath.Join(base, "app"), "v1.0.0", "")
	work := "go " + goVersion() + "\n\nuse (\n\t./app\n\t./mygo\n)\n"
	if err := os.WriteFile(filepath.Join(base, "go.work"), []byte(work), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := goOutput(base, "work", "vendor"); err != nil {
		t.Fatalf("go work vendor: %v\n%s", err, out)
	}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("a workspace that uses MyGo as a directory, vendored: %v, %v", has, err)
	}
}

// writeTestFile writes a file of a test app, and the directories above it.
func writeTestFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAppHasPackageIgnoresAVendorDirectoryGoIgnores: go uses a vendor directory
// by default only when the first line of its modules.txt says "## workspace" if
// and only if the build is in a workspace (modulesTextIsForWorkspace), so the
// vendor directory of go mod vendor next to a go.work, and that of go work
// vendor with GOWORK=off, are ignored: go builds MyGo, v1.1.0 with protocol,
// from the module cache, and so does the answer.
func TestAppHasPackageIgnoresAVendorDirectoryGoIgnores(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	for _, tc := range []struct{ name, vendor, workspace string }{
		{"go mod vendor next to go.work", "mod", "on"},
		{"go work vendor with GOWORK=off", "work", "off"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			applicationEnv(t, "")
			root := t.TempDir()
			c := requiringApp(t, root, "v1.1.0", "")
			work := filepath.Join(root, "go.work")
			if tc.vendor == "mod" {
				t.Setenv("GOWORK", "off")
				if out, err := goOutput(root, "mod", "vendor"); err != nil {
					t.Fatalf("go mod vendor: %v\n%s", err, out)
				}
				t.Setenv("GOWORK", "")
				writeTestFile(t, work, "go "+goVersion()+"\n\nuse .\n")
			} else {
				writeTestFile(t, work, "go "+goVersion()+"\n\nuse .\n")
				if out, err := goOutput(root, "work", "vendor"); err != nil {
					t.Fatalf("go work vendor: %v\n%s", err, out)
				}
				t.Setenv("GOWORK", "off")
			}
			moduleDirs.m = map[string]moduleDirEntry{}
			if out, err := goOutput(root, "build", "-o", os.DevNull, "."); err != nil {
				t.Fatalf("go builds the app (it ignores that vendor directory): %v\n%s", err, out)
			}
			if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
				t.Errorf("MyGo v1.1.0 from the module cache has protocol: %v, %v", has, err)
			}
		})
	}
}

// TestAppHasPackageReadsTheModuleLineAsGoDoes: a MyGo the workspace uses as a
// directory is found whatever its module line carries, a comment or quotes.
func TestAppHasPackageReadsTheModuleLineAsGoDoes(t *testing.T) {
	if testing.Short() {
		t.Skip("go")
	}
	applicationEnv(t, "")
	base := t.TempDir()
	local := localMygo(t, base, true)
	writeTestFile(t, filepath.Join(local, "go.mod"), "module \"github.com/egoist/mygo\" // MyGo\n\ngo "+goVersion()+"\n")
	c := requiringApp(t, filepath.Join(base, "app"), "v1.0.0", "")
	writeTestFile(t, filepath.Join(base, "go.work"), "go "+goVersion()+"\n\nuse (\n\t./app\n\t./mygo\n)\n")
	if out, err := goOutput(base, "work", "vendor"); err != nil {
		t.Fatalf("go work vendor: %v\n%s", err, out)
	}
	moduleDirs.m = map[string]moduleDirEntry{}
	if has, err := appHasPackage(c, protocolPackage); err != nil || !has {
		t.Errorf("the used directory has protocol: %v, %v", has, err)
	}
}

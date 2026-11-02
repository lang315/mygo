package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// staticBuild does what a release build does for an app with no frontend
// (listBoundTypes, bindFiles for the target, the overlay and the tag mygo_static)
// for goos on this computer's architecture, and, when goos is this computer's,
// runs the result in generate mode, where Bind and Use run before App.Run
// returns. It returns the generated file, the reason Bind keeps reflection, the
// build's error and output, and the run's.
func staticBuild(t *testing.T, goos string, files map[string]string) (gen, reason string, buildErr error, buildOut string, runErr error, runOut string) {
	t.Helper()
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, files["main.go"])
	for name, src := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BINDTEST_MARKER", filepath.Join(t.TempDir(), "marker"))
	if err := listBoundTypes(context.Background(), c); err != nil {
		t.Fatalf("listBoundTypes: %v", err)
	}
	pkg, err := packageDir(c)
	if err != nil {
		t.Fatal(err)
	}
	mainPkg, err := packageImportPath(c)
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	table, reason, err := bindFiles(c, pkg, work, mainPkg, goos, runtime.GOARCH, "")
	if err != nil {
		t.Fatal(err)
	}
	if reason != "" || len(table) == 0 {
		return "", reason, nil, "", nil, ""
	}
	for _, f := range table {
		src, _ := os.ReadFile(f)
		gen = string(src)
	}
	overlay, err := writeOverlay(filepath.Join(work, "overlay.json"), table)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(work, "app")
	var out bytes.Buffer
	cmd := goCommand(dir, []string{"GOOS=" + goos, "GOARCH=" + runtime.GOARCH}, "build", "-trimpath", "-tags", "mygo_static", "-overlay", overlay, "-o", bin, ".")
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return gen, "", err, out.String(), nil, ""
	}
	if goos != runtime.GOOS {
		return gen, "", nil, out.String(), nil, ""
	}
	run := exec.Command(bin)
	run.Dir = dir
	run.Env = append(os.Environ(), "MYGO_GENERATE="+filepath.Join(work, "static.ts"))
	o, rerr := run.CombinedOutput()
	return gen, "", nil, out.String(), rerr, string(o)
}

// An app that binds a service and uses a plugin with a service (fetch, the most
// common) starts in a release build: the plugin gives Bind its own methods
// (BindMethods), where the build has none to name for a type it cannot.
func TestAppWithAPluginStartsWithTheTable(t *testing.T) {
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/fetch"
)

type Greeter struct{}

func (Greeter) Hello(name string) string { return name }

func main() {
	mygo.Bind(&Greeter{})
	mygo.Use(fetch.Plugin)
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
	if reason != "" || gen == "" {
		t.Fatalf("Bind kept reflection (%q): the plugin's service should not stop the table", reason)
	}
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	if rerr != nil {
		t.Errorf("the release build of an app that binds and uses fetch fails at start: %v\n%s", rerr, rout)
	}
	if strings.Contains(gen, "fetch") {
		t.Errorf("the table names the plugin's service, which the plugin gives itself:\n%s", gen)
	}
}

// A type declared inside a function has PkgPath "main" and a Name, but no name
// the generated file can use: Bind keeps reflection, and says why.
func TestALocalTypeKeepsReflection(t *testing.T) {
	gen, reason, _, _, _, _ := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import "github.com/egoist/mygo"

type Greeter struct{}

func (Greeter) Hello(name string) string { return name }

func main() {
	type API struct{ Greeter }
	mygo.Bind(&API{})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
	if gen != "" || !strings.Contains(reason, "API") {
		t.Errorf("gen %q, reason %q: want reflection kept, and the type named", gen, reason)
	}
}

// The list is made on this computer, the build is for another: the methods come
// from the target. A method only this computer's GOOS has is not in the table
// (the build for the target compiles); a method only the target has is.
func TestTheTableHasTheMethodsOfTheTarget(t *testing.T) {
	host := runtime.GOOS
	target := "linux"
	if host == "linux" {
		target = "windows"
	}
	gen, reason, berr, bout, _, _ := staticBuild(t, target, map[string]string{
		"main.go":                   bindTestMain,
		"greeter_" + host + ".go":   "package main\n\nfunc (g *Greeter) HostOnly() string { return \"host\" }\n",
		"greeter_" + target + ".go": "package main\n\nfunc (g *Greeter) TargetOnly() string { return \"target\" }\n",
	})
	if reason != "" {
		t.Fatalf("Bind kept reflection: %s", reason)
	}
	if berr != nil {
		t.Fatalf("cross-building for %s from %s: %v\n%s", target, host, berr, bout)
	}
	if strings.Contains(gen, "HostOnly") || !strings.Contains(gen, "TargetOnly") {
		t.Errorf("the table for %s names the wrong methods:\n%s", target, gen)
	}
}

// Control: an embedded interface's promoted method, a variadic and a context.
func TestPromotedInterfaceMethodsThroughTheTable(t *testing.T) {
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import (
	"context"

	"github.com/egoist/mygo"
)

type Helloer interface{ Hello(name string) string }

type impl struct{}

func (impl) Hello(name string) string { return name }

type API struct{ Helloer }

func (*API) Sum(ctx context.Context, xs ...int) int { return len(xs) }

func main() {
	mygo.Bind(&API{impl{}})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
	if reason != "" || gen == "" {
		t.Fatalf("fell back: %s", reason)
	}
	if berr != nil || rerr != nil {
		t.Fatalf("build %v\n%s\nrun %v\n%s", berr, bout, rerr, rout)
	}
	if !strings.Contains(gen, "Hello") || !strings.Contains(gen, "Sum") {
		t.Errorf("the table lacks a promoted method:\n%s", gen)
	}
}

// If the generated table still does not compile for the target, the build is
// made again without it and Bind keeps reflection: the release build does not
// fail for an optimization. Here a table that names a method the type does not have.
func TestABuildWhoseTableDoesNotCompileKeepsReflection(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	saved := targetTypes
	targetTypes = func(c *Config, goos, goarch, tags, mainPkg string, bound []boundType) ([]boundType, string, error) {
		out, reason, err := saved(c, goos, goarch, tags, mainPkg, bound)
		for i := range out {
			out[i].Methods = append(out[i].Methods, "Zzz") // there is no (*Greeter).Zzz
		}
		return out, reason, err
	}
	defer func() { targetTypes = saved }()
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, bindTestBinds)
	t.Setenv("BINDTEST_MARKER", filepath.Join(t.TempDir(), "marker"))
	out := t.TempDir()
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if err := runBuild([]string{"-platform", platform, "-skip-dmg", "-skip-installer", "-skip-notarize", "-sign", "-", "-o", out, dir}); err != nil {
		t.Fatalf("the build with a table that does not compile: %v", err)
	}
}

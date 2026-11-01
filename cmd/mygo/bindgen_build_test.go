package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// bindFiles decides, from the bound types generate mode listed, whether the
// build gets the file and the tag.
func TestBindFilesDecides(t *testing.T) {
	pkg, work := t.TempDir(), t.TempDir()
	c := &Config{root: pkg, boundRead: true, bound: []boundType{
		{Pkg: "main", Type: "Greeter", Pointer: true, Methods: []string{"Hello"}},
	}}

	files, reason, err := bindFiles(c, pkg, work, "example.com/app")
	if err != nil || reason != "" || len(files) != 1 {
		t.Fatalf("files %v, reason %q, err %v", files, reason, err)
	}
	gen := filepath.Join(pkg, bindGenFile)
	src, err := os.ReadFile(files[gen])
	if err != nil || !bytes.Contains(src, []byte("BindMethods(")) {
		t.Fatalf("the overlay entry %s: %v\n%s", files[gen], err, src)
	}

	// A type that cannot be named: no file, and the reason for the line.
	c.bound = append(c.bound, boundType{Pkg: "example.com/app/svc", Type: "clock", Methods: []string{"Now"}})
	files, reason, err = bindFiles(c, pkg, work, "example.com/app")
	if err != nil || len(files) != 0 || !strings.Contains(reason, "clock") {
		t.Fatalf("files %v, reason %q, err %v", files, reason, err)
	}

	// Nothing bound, or generate mode never ran (an app without a frontend):
	// neither a file nor a reason.
	for _, c := range []*Config{{root: pkg, boundRead: true}, {root: pkg}} {
		if files, reason, err = bindFiles(c, pkg, work, "example.com/app"); err != nil || len(files) != 0 || reason != "" {
			t.Errorf("%+v: files %v, reason %q, err %v", c, files, reason, err)
		}
	}

	// A file of the app in the way is an error, as for the frontend's.
	c.bound = c.bound[:1]
	if err := os.WriteFile(filepath.Join(pkg, bindGenFile), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := bindFiles(c, pkg, work, "example.com/app"); err == nil || !strings.Contains(err.Error(), "in the way") {
		t.Errorf("err = %v, want one that says the file is in the way", err)
	}
}

func TestWithTag(t *testing.T) {
	for _, c := range []struct{ in, want []string }{
		{nil, []string{"-tags", "mygo_static"}},
		{[]string{"-tags", "a,mygo_noinspector"}, []string{"-tags", "a,mygo_noinspector,mygo_static"}},
		{[]string{"-trimpath"}, []string{"-trimpath", "-tags", "mygo_static"}},
	} {
		if got := withTag(c.in, "mygo_static"); strings.Join(got, " ") != strings.Join(c.want, " ") {
			t.Errorf("withTag(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

const bindTestMain = `package main

import "github.com/egoist/mygo"

type Greeter struct{ greeting string }

func (g Greeter) Hello(name string) string { return g.greeting + name }
func (g *Greeter) Reset()                  { g.greeting = "" }

func main() {
	mygo.Bind(&Greeter{greeting: "hi "})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`

// bindTestApp writes an app that binds a service, with the config given, into
// a new directory, and sets the environment to build it offline.
func bindTestApp(t *testing.T, config, main string) string {
	t.Helper()
	dir := t.TempDir()
	checkout, _ := filepath.Abs(filepath.Join("..", ".."))
	sum, _ := os.ReadFile(filepath.Join(checkout, "go.sum"))
	gomod := "module bindtest\n\ngo " + goVersion() + "\n\nrequire github.com/egoist/mygo v0.0.0\n\nreplace github.com/egoist/mygo => " + checkout + "\n"
	for name, content := range map[string]string{
		"go.mod":    gomod,
		"go.sum":    string(sum),
		"main.go":   main,
		"mygo.json": config,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOWORK", "off")
	return dir
}

// Apps for the three ways listBoundTypes can go. Each writes a marker file
// first thing in main, so that a test can tell whether the build ran it.
const (
	bindTestHead = `package main

import (
	"os"

	"github.com/egoist/mygo"
)

type Greeter struct{}

func (Greeter) Hello(name string) string { return name }

func init() { os.WriteFile(os.Getenv("BINDTEST_MARKER"), []byte("ran"), 0o644) }
`
	bindTestNoBind = bindTestHead + `
func main() {
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`
	bindTestBinds = bindTestHead + `
func main() {
	mygo.Bind(&Greeter{})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`
	bindTestFails = bindTestHead + `
func main() {
	mygo.Bind(&Greeter{})
	if os.Getenv("MYGO_GENERATE") != "" {
		os.Exit(3)
	}
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`
)

// listBound lists the bound types of the app written by main, and says whether
// the build ran the app (its marker exists).
func listBound(t *testing.T, main string) (c *Config, ran bool, err error) {
	t.Helper()
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, main)
	marker := filepath.Join(t.TempDir(), "marker")
	t.Setenv("BINDTEST_MARKER", marker)
	c, lerr := loadConfig(dir)
	if lerr != nil {
		t.Fatal(lerr)
	}
	if c.Bindings != "" {
		t.Fatalf("Bindings = %q: the app was meant to have no client", c.Bindings)
	}
	err = listBoundTypes(context.Background(), c)
	_, serr := os.Stat(marker)
	if _, cerr := os.Stat(filepath.Join(dir, "mygo.ts")); cerr == nil {
		t.Error("a client was written into the project")
	}
	return c, serr == nil, err
}

// An app with no frontend, as hello is, has no client to write, but may bind
// services. The build runs the app in generate mode to list them, once, only if it
// binds: the code of an app that binds nothing (most apps of native UI) does not
// run at build time.
func TestListBoundTypesRunsOnlyAnAppThatBinds(t *testing.T) {
	t.Run("an app that binds nothing is not run", func(t *testing.T) {
		c, ran, err := listBound(t, bindTestNoBind)
		if err != nil || ran || len(c.bound) != 0 {
			t.Errorf("ran %v, bound %+v, err %v: want none, and the app not run", ran, c.bound, err)
		}
		if !c.boundRead {
			t.Error("nothing bound is a list read: the build can go on with no table")
		}
	})
	t.Run("an app that binds is run in generate mode", func(t *testing.T) {
		c, ran, err := listBound(t, bindTestBinds)
		if err != nil || !ran || !c.boundRead || len(c.bound) != 1 || c.bound[0].Type != "Greeter" {
			t.Errorf("ran %v, bound %+v (read %v), err %v: want the Greeter listed by a run", ran, c.bound, c.boundRead, err)
		}
	})
	t.Run("an app that fails in generate mode is an error and no list", func(t *testing.T) {
		c, ran, err := listBound(t, bindTestFails)
		if err == nil || !ran || c.boundRead || len(c.bound) != 0 {
			t.Errorf("ran %v, bound %+v (read %v), err %v: want an error and no list", ran, c.bound, c.boundRead, err)
		}
	})
}

// hasBindSymbol tells from the symbols of a host binary that is not stripped
// whether the app calls Bind.
func TestHasBindSymbol(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	for name, c := range map[string]struct {
		main string
		want bool
	}{"binds": {bindTestBinds, true}, "does not": {bindTestNoBind, false}} {
		t.Run(name, func(t *testing.T) {
			dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, c.main)
			cfg, err := loadConfig(dir)
			if err != nil {
				t.Fatal(err)
			}
			bin := filepath.Join(t.TempDir(), "app")
			if err := goBuild(context.Background(), cfg, nil, bin, nil, generatorTags(cfg.root)...); err != nil {
				t.Fatal(err)
			}
			got, err := bindsServices(bin)
			if err != nil || got != c.want {
				t.Errorf("bindsServices = %v, %v, want %v", got, err, c.want)
			}
		})
	}
}

// TestBuildNamesTheMethodsThroughTheTable builds an app that binds a service
// the way mygo build does, with a real go build: the generated file compiles
// against the app's types, and with the tag mygo_static the linker finds no
// reflect.Value.Method (-dumpdep), where without it, the same app has it.
func TestBuildNamesTheMethodsThroughTheTable(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest","bindings":"mygo.ts"}`, bindTestMain)

	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeClient(context.Background(), c, false); err != nil {
		t.Fatal(err)
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
	files, reason, err := bindFiles(c, pkg, work, mainPkg)
	if err != nil || reason != "" || len(files) != 1 {
		t.Fatalf("files %v, reason %q, err %v", files, reason, err)
	}
	overlay, err := writeOverlay(filepath.Join(work, "overlay.json"), files)
	if err != nil {
		t.Fatal(err)
	}

	reflectMethods := func(tags string, overlay string) int {
		t.Helper()
		args := []string{"build", "-trimpath", "-tags", tags, "-ldflags", "-s -w -dumpdep", "-o", filepath.Join(work, "app-"+tags)}
		if overlay != "" {
			args = append(args, "-overlay", overlay)
		}
		args = append(args, ".")
		var out bytes.Buffer
		cmd := goCommand(dir, nil, args...)
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("go build -tags %s: %v\n%s", tags, err, out.String())
		}
		return strings.Count(out.String(), "<ReflectMethod>")
	}
	if n := reflectMethods("mygo_noinspector", ""); n == 0 {
		t.Fatal("the build without the tag has no <ReflectMethod>: this test would not tell the table from reflection")
	}
	if n := reflectMethods("mygo_noinspector,mygo_static", overlay); n != 0 {
		t.Errorf("with the tag and the table the linker sees %d <ReflectMethod>", n)
	}

	// The app built with the tag binds its service from the table when it starts
	// (Bind runs before App.Run): generate mode, which returns before any window,
	// lists what it bound, the methods the table named.
	if n := reflectMethods("mygo_static", overlay); n != 0 {
		t.Errorf("the generator build with the tag sees %d <ReflectMethod>", n)
	}
	list := filepath.Join(work, "static-bound.json")
	run := exec.Command(filepath.Join(work, "app-mygo_static"))
	run.Dir = dir
	run.Env = append(os.Environ(), "MYGO_GENERATE="+filepath.Join(work, "static.ts"), "MYGO_GENERATE_BINDINGS="+list)
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("the app built with the table: %v\n%s", err, out)
	}
	got, err := readBoundTypes(list)
	if err != nil || len(got) != 1 || got[0].Type != "Greeter" || !got[0].Pointer || strings.Join(got[0].Methods, ",") != "Hello,Reset" {
		t.Errorf("the app bound %+v (%v), want *Greeter with Hello and Reset", got, err)
	}
}

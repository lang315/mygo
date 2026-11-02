package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// rev338bStatic lists the bound types as mygo build does, makes the table for
// this computer, builds the app with it and the release's ldflags (production=1),
// and runs it in generate mode, where Bind runs before App.Run returns.
func rev338bStatic(t *testing.T, main string) (reason string, buildErr error, runErr error, out string) {
	t.Helper()
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, main)
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
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
	table, reason, err := bindFiles(c, pkg, work, mainPkg, runtime.GOOS, runtime.GOARCH, "")
	if err != nil {
		t.Fatal(err)
	}
	if reason != "" || len(table) == 0 {
		return reason, nil, nil, ""
	}
	overlay, err := writeOverlay(filepath.Join(work, "overlay.json"), table)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(work, "app")
	var b bytes.Buffer
	cmd := goCommand(dir, nil, "build", "-trimpath", "-tags", "mygo_static", "-ldflags", "-X github.com/egoist/mygo.production=1", "-overlay", overlay, "-o", bin, ".")
	cmd.Stdout, cmd.Stderr = &b, &b
	if err := cmd.Run(); err != nil {
		return "", err, nil, b.String()
	}
	run := exec.Command(bin)
	run.Dir = dir
	run.Env = append(os.Environ(), "MYGO_GENERATE="+filepath.Join(work, "static.ts"))
	o, rerr := run.CombinedOutput()
	return "", nil, rerr, string(o)
}

// A service bound only in production (mygo.IsDev() false) is not in the list:
// generate mode runs a build without production=1, where IsDev is true. The
// release build has the tag, no table for ProdAPI, and fails at Bind.
func TestRev338bProductionOnlyServiceFailsAtStart(t *testing.T) {
	reason, berr, rerr, out := rev338bStatic(t, `package main

import "github.com/egoist/mygo"

type DevAPI struct{}

func (DevAPI) Dev() string { return "dev" }

type ProdAPI struct{}

func (ProdAPI) Prod() string { return "prod" }

func main() {
	if mygo.IsDev() {
		mygo.Bind(&DevAPI{})
	} else {
		mygo.Bind(&ProdAPI{})
	}
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`)
	if reason != "" {
		t.Skipf("kept reflection: %s", reason)
	}
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, out)
	}
	if rerr != nil {
		t.Errorf("the release build of an app that binds a service only when !mygo.IsDev() fails at start: %v\n%s", rerr, firstLine(out))
	}
}

// A service chosen by runtime.GOOS: the list comes from this computer's run, so
// the table for another GOOS names this computer's type and not the target's.
// The cross-built app panics at Bind on the target.
func TestRev338bServiceChosenByGOOSIsMissingFromTheTargetTable(t *testing.T) {
	target := "linux"
	if runtime.GOOS == "linux" {
		target = "windows"
	}
	gen, reason, berr, bout, _, _ := staticBuild(t, target, map[string]string{"main.go": `package main

import (
	"runtime"

	"github.com/egoist/mygo"
)

type HostAPI struct{}

func (HostAPI) Host() string { return "host" }

type TargetAPI struct{}

func (TargetAPI) Target() string { return "target" }

func main() {
	if runtime.GOOS == "` + runtime.GOOS + `" {
		mygo.Bind(&HostAPI{})
	} else {
		mygo.Bind(&TargetAPI{})
	}
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
	if reason != "" {
		t.Skipf("kept reflection: %s", reason)
	}
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	if !strings.Contains(gen, "TargetAPI") {
		t.Errorf("the table for %s has no TargetAPI, which the app binds there (Bind fails at start with the tag):\n%s", target, gen)
	}
}

// go/types' method sets for the target against reflection's, for types whose
// methods come through embedding (unexported types of other packages, pointers,
// interfaces with unexported methods, generics, ambiguous selectors) and
// non-ASCII names, in reflection's order.
func TestRev338bGoTypesMethodSetsMatchReflection(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, `package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"bindtest/sub"
)

type S1 struct{ sub.Outer }
type S2 struct{ sub.I }
type S3 struct{ sub.G[int] }
type S4 struct{ *sub.E }
type S5 struct{ sub.E }
type S6 sub.Base
type S7 struct {
	sub.Base
	sub.Outer
}
type A struct{}

func (A) Dup() {}

type B struct{}

func (B) Dup() {}

type S8 struct {
	A
	B
}
type S9 struct{}

func (S9) Ñandu()  {}
func (S9) Zeta()   {}
func (S9) Alpha()  {}
func (S9) Émile()  {}
func (S9) Δelta()  {}
func (S9) ǅx()     {}
func (S9) Z_()     {}
func (S9) Za()     {}
func (*S9) PtrM()  {}

type S10 = sub.Outer
type S11 struct {
	sync.Mutex
	*strings.Builder
}
type S12 struct{ sub.Deep }

// neverCalled names the types to the scan of mygo build, as a call of Bind does.
func neverCalled() {
	mygo.Bind(S1{}, &S1{}, S2{}, &S2{}, S3{}, &S3{}, S4{}, &S4{}, S5{}, &S5{}, S6{}, &S6{}, S7{}, &S7{}, S8{}, &S8{}, S9{}, &S9{}, S10{}, &S10{}, S11{}, &S11{}, S12{}, &S12{})
}

type entry struct {
	Pkg     string   `+"`json:\"pkg\"`"+`
	Type    string   `+"`json:\"type\"`"+`
	Pointer bool     `+"`json:\"pointer\"`"+`
	Methods []string `+"`json:\"methods\"`"+`
}

func main() {
	var out []entry
	for _, v := range []any{S1{}, S2{}, S3{}, S4{}, S5{}, S6{}, S7{}, S8{}, S9{}, S10{}, S11{}, S12{}} {
		for _, t := range []reflect.Type{reflect.TypeOf(v), reflect.PointerTo(reflect.TypeOf(v))} {
			e := entry{Pointer: t.Kind() == reflect.Pointer}
			n := t
			if e.Pointer {
				n = t.Elem()
			}
			e.Pkg, e.Type = n.PkgPath(), n.Name()
			for i := range t.NumMethod() {
				e.Methods = append(e.Methods, t.Method(i).Name)
			}
			out = append(out, e)
		}
	}
	json.NewEncoder(os.Stdout).Encode(out)
}
`)
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "sub.go"), []byte(`package sub

type inner struct{}

func (inner) FromInner()  {}
func (*inner) PtrInner()  {}
func (inner) unexported() {}

type Outer struct{ inner }

type I interface {
	IfaceM()
	hidden()
}

type G[T any] struct{ v T }

func (g G[T]) GenGet() T { return g.v }

type Base struct{}

func (Base) BaseM() {}

type E struct{}

func (*E) EPtr() {}
func (E) EVal()  {}

type deeper struct{ *Outer }

func (deeper) DeepM() {}

type Deep struct{ deeper }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	run := goCommand(dir, nil, "run", ".")
	run.Stdout, run.Stderr = &stdout, &stderr
	if err := run.Run(); err != nil {
		t.Fatalf("go run: %v\n%s", err, stderr.String())
	}
	var want []boundType
	if err := json.Unmarshal(stdout.Bytes(), &want); err != nil {
		t.Fatal(err)
	}
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	mainPkg, err := packageImportPath(c)
	if err != nil {
		t.Fatal(err)
	}
	in := make([]boundType, len(want))
	for i, w := range want {
		in[i] = boundType{Pkg: w.Pkg, Type: w.Type, Pointer: w.Pointer}
	}
	got, reason, err := targetBoundTypes(c, runtime.GOOS, runtime.GOARCH, "", mainPkg, in)
	if err != nil || reason != "" {
		t.Fatalf("targetBoundTypes: %v %s", err, reason)
	}
	if len(got) != len(want) {
		t.Fatalf("the scan found %d types, reflection %d", len(got), len(want))
	}
	for _, w := range want {
		i := slices.IndexFunc(got, func(g boundType) bool {
			return (g.Pkg == w.Pkg || w.Pkg == "main" && g.Pkg == mainPkg) && g.Type == w.Type && g.Pointer == w.Pointer
		})
		if i < 0 {
			t.Errorf("%s.%s (pointer %v) was not found by the scan", w.Pkg, w.Type, w.Pointer)
			continue
		}
		if !slices.Equal(got[i].Methods, w.Methods) && !(len(got[i].Methods) == 0 && len(w.Methods) == 0) {
			star := ""
			if w.Pointer {
				star = "*"
			}
			t.Errorf("%s%s.%s: go/types %v, reflect %v", star, w.Pkg, w.Type, got[i].Methods, w.Methods)
		}
	}
	t.Logf("compared %d types", len(want))
	_ = reflect.TypeOf
}

// An app that calls sqlite.Open from Go, and never Uses the plugin (no service
// bound), now links the plugin's service methods: bindtable.go's init names them.
func TestRev338bSqliteGoOnlyLinksServiceMethods(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, `package main

import (
	"context"

	"github.com/egoist/mygo/plugins/sqlite"
)

func main() {
	db, err := sqlite.Open(context.Background(), ":memory:", sqlite.OpenOptions{})
	if err == nil {
		db.Close()
	}
}
`)
	bin := filepath.Join(t.TempDir(), "app")
	var b bytes.Buffer
	cmd := goCommand(dir, nil, "build", "-trimpath", "-o", bin, ".")
	cmd.Stdout, cmd.Stderr = &b, &b
	if err := cmd.Run(); err != nil {
		t.Fatalf("build: %v\n%s", err, b.String())
	}
	names, err := symbolNames(bin)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if strings.HasSuffix(n, "plugins/sqlite.(*service).Query") || strings.HasSuffix(n, "mygo.BindMethods") {
			t.Errorf("an app that never uses the sqlite plugin's service links %s", n)
		}
	}
}

// Bind in a package-level var initializer of package main: the generated file's
// init runs after every var of the package is initialized, so Bind finds no table.
func TestRev338bBindInVarInitializer(t *testing.T) {
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import "github.com/egoist/mygo"

type API struct{}

func (API) Hello() string { return "hi" }

var api = bindAPI()

func bindAPI() *API {
	a := &API{}
	mygo.Bind(a)
	return a
}

func main() {
	_ = api
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
	if reason != "" || gen == "" {
		t.Skipf("kept reflection: %s", reason)
	}
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	if rerr != nil {
		t.Errorf("Bind in a var initializer of main fails at start with the table: %v\n%s", rerr, firstLine(rout))
	}
}

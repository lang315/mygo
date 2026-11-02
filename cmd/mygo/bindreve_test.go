package main

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Same rule as rev338d: A is bound always (so a table exists), the service S is
// bound in a branch the build's run does not take. The build must name *S in the
// table or keep reflection with a reason.
func rev338eCheck(t *testing.T, imports, decls, body string, extra map[string]string) {
	t.Helper()
	rev338eCheckCause(t, false, imports, decls, body, extra)
}

// rev338eCheckCause is rev338eCheck; with causeIsEnough a release app that panics at
// start with the error that says it has no list of the methods passes: it is the
// documented outcome of what the scan cannot see (reflect writing a field).
func rev338eCheckCause(t *testing.T, causeIsEnough bool, imports, decls, body string, extra map[string]string) {
	t.Helper()
	src := `package main

import (
	"os"
` + imports + `
	"github.com/egoist/mygo"
)

type A struct{}

func (*A) X() string { return "" }

` + decls + `

func main() {
	mygo.Bind(&A{})
	if os.Getenv("MYGO_GENERATE_BINDINGS") == "" {
` + body + `
	}
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`
	files := map[string]string{"main.go": src}
	for k, v := range extra {
		files[k] = v
	}
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, files)
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	switch {
	case reason != "":
		t.Logf("OK fallback: %s", reason)
	case rerr == nil:
		t.Logf("OK table, run fine:\n%s", gen)
	case causeIsEnough && strings.Contains(rout, "has no list of its methods"):
		t.Logf("OK documented: the app panics at start with the cause")
	default:
		t.Errorf("BUG: silent build, release run: %v: %s", rerr, firstLine(rout))
	}
}

const rev338eS = `type S struct{}

func (*S) Ping() string { return "p" }
`

func TestRev338eAlias(t *testing.T) {
	rev338eCheck(t, "", rev338eS+"type P = mygo.Plugin\n", `		mygo.Use(P{Name: "late", Service: &S{}})`, nil)
}

func TestRev338eDefinedType(t *testing.T) {
	rev338eCheck(t, "", rev338eS+"type P mygo.Plugin\n", `		mygo.Use(mygo.Plugin(P{Name: "late", Service: &S{}}))`, nil)
}

func TestRev338eDefinedTypeNoCall(t *testing.T) {
	// A conversion written as a composite assignment of the fields.
	rev338eCheck(t, "", rev338eS+"type P mygo.Plugin\n", `		q := P{Name: "late", Service: &S{}}
		mygo.Use(mygo.Plugin{Name: q.Name, Service: q.Service})`, nil)
}

func TestRev338eGenericWrapper(t *testing.T) {
	rev338eCheck(t, "", rev338eS+`func use[T any](v T) { mygo.Use(mygo.Plugin{Name: "late", Service: v}) }
`, `		use(&S{})`, nil)
}

// Package lib imports mygo, and offers the plugin type under an alias; package
// feature uses lib only, so it does not import mygo and the scan skips it.
const rev338eLib = `package lib

import "github.com/egoist/mygo"

type P = mygo.Plugin

func Use(p P) { mygo.Use(p) }
`

func TestRev338eAliasInOtherPackageLiteral(t *testing.T) {
	rev338eCheck(t, `	"bindtest/feature"
	"bindtest/lib"
`, "", `		lib.Use(feature.Plugin())`, map[string]string{"lib/lib.go": rev338eLib, "feature/feature.go": `package feature

import "bindtest/lib"

type S struct{}

func (*S) Ping() string { return "p" }

func Plugin() lib.P { return lib.P{Name: "late", Service: &S{}} }
`})
}

func TestRev338eAliasInOtherPackageAssign(t *testing.T) {
	rev338eCheck(t, `	"bindtest/feature"
	"bindtest/lib"
`, "", `		lib.Use(feature.Plugin())`, map[string]string{"lib/lib.go": rev338eLib, "feature/feature.go": `package feature

import "bindtest/lib"

type S struct{}

func (*S) Ping() string { return "p" }

func Plugin() lib.P {
	var p lib.P
	p.Name = "late"
	p.Service = &S{}
	return p
}
`})
}

func TestRev338eEmbeddedInOtherPackage(t *testing.T) {
	rev338eCheck(t, `	"bindtest/feature"
`, "", `		mygo.Use(feature.Plugin().Plugin)`, map[string]string{"lib/lib.go": `package lib

import "github.com/egoist/mygo"

type W struct{ mygo.Plugin }
`, "feature/feature.go": `package feature

import "bindtest/lib"

type S struct{}

func (*S) Ping() string { return "p" }

func Plugin() lib.W {
	var w lib.W
	w.Name = "late"
	w.Service = &S{}
	return w
}
`})
}

// The plugin's literal is fine and scanned, the service comes as a parameter.
func TestRev338eConstructorInOtherPackage(t *testing.T) {
	rev338eCheck(t, `	"bindtest/lib"
`, rev338eS, `		mygo.Use(lib.New(&S{}))`, map[string]string{"lib/lib.go": `package lib

import "github.com/egoist/mygo"

func New(s any) mygo.Plugin { return mygo.Plugin{Name: "late", Service: s} }
`})
}

func TestRev338eReflectSet(t *testing.T) {
	rev338eCheckCause(t, true, `	"reflect"
`, rev338eS, `		var p mygo.Plugin
		p.Name = "late"
		reflect.ValueOf(&p).Elem().FieldByName("Service").Set(reflect.ValueOf(&S{}))
		mygo.Use(p)`, nil)
}

func TestRev338eRangeAssign(t *testing.T) {
	rev338eCheck(t, "", rev338eS, `		p := mygo.Plugin{Name: "late"}
		for _, p.Service = range []any{&S{}} {
		}
		mygo.Use(p)`, nil)
}

func TestRev338eRangeAssignConcrete(t *testing.T) {
	rev338eCheck(t, "", rev338eS, `		p := mygo.Plugin{Name: "late"}
		for _, p.Service = range []*S{{}} {
		}
		mygo.Use(p)`, nil)
}

func TestRev338ePluginCopiedFromVarOfAny(t *testing.T) {
	// A Plugin stored as any and asserted back: no literal in that path.
	rev338eCheck(t, "", rev338eS, `		var x any = mygo.Plugin{Name: "late", Service: &S{}}
		mygo.Use(x.(mygo.Plugin))`, nil)
}

func TestRev338eStructCopyEmbedded(t *testing.T) {
	// Unkeyed literal of Plugin.
	rev338eCheck(t, "", rev338eS, `		mygo.Use(mygo.Plugin{"late", &S{}, nil})`, nil)
}

func TestRev338eBindInterfaceHolder(t *testing.T) {
	// Bind reached through an interface's method: the wrapper names it, called directly.
	rev338eCheck(t, "", rev338eS+`type binder interface{ B(any) }
type real struct{}

func (real) B(v any) { mygo.Bind(v) }
`, `		var b binder = real{}
		b.B(&S{})`, nil)
}

func TestRev338eParenCallee(t *testing.T) {
	rev338eCheck(t, "", rev338eS, `		(mygo.Bind)(&S{})`, nil)
}

// The file of a Bind for another GOOS is not in the build of this one, and a
// _test.go file is not in any build.
func TestRev338eOtherGOOSAndTestFile(t *testing.T) {
	other := "linux"
	if runtime.GOOS == "linux" {
		other = "darwin"
	}
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import "github.com/egoist/mygo"

type A struct{}

func (*A) X() string { return "" }

func main() {
	mygo.Bind(&A{})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`, "other.go": "//go:build " + other + `

package main

import "github.com/egoist/mygo"

type O struct{}

func (*O) Y() string { return "" }

func init() { mygo.Bind(&O{}) }
`, "x_test.go": `package main

import "github.com/egoist/mygo"

type T struct{}

func (*T) Z() string { return "" }

func init() { reg := mygo.Bind; reg(&T{}) }
`})
	if berr != nil || rerr != nil {
		t.Fatalf("berr=%v %s rerr=%v %s", berr, bout, rerr, rout)
	}
	if reason != "" || strings.Contains(gen, "[*O]") || strings.Contains(gen, "[*T]") {
		t.Errorf("reason=%q gen:\n%s", reason, gen)
	}
}

// What the scan costs: the packages it type-checks with bodies, and its time.
func TestRev338eScanCost(t *testing.T) {
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, `package main

import (
	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/ui"
	_ "github.com/egoist/mygo/ui/widget"
	_ "github.com/egoist/mygo/plugins/fetch"
	_ "github.com/egoist/mygo/plugins/sqlite"
)

type A struct{}

func (*A) X() string { return "" }

func main() {
	mygo.Bind(&A{})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`)
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := listBoundTypes(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	mainPkg, _ := packageImportPath(c)
	for i := range 2 {
		start := time.Now()
		out, reason, err := targetBoundTypes(c, runtime.GOOS, runtime.GOARCH, "", mainPkg, c.bound)
		t.Logf("run %d: %v, %d types, reason=%q err=%v", i, time.Since(start), len(out), reason, err)
	}
}

// A package that registers a table for one of its services still has its other
// services scanned: they get the table of the build.
func TestRev338eSecondServiceOfAPackageWithATable(t *testing.T) {
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import (
	"github.com/egoist/mygo"

	_ "bindtest/lib"
)

func main() {
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`, "lib/lib.go": `package lib

import (
	"reflect"

	"github.com/egoist/mygo"
)

type one struct{}

func (*one) A() string { return "" }

type Two struct{}

func (*Two) B() string { return "" }

func init() {
	mygo.BindMethods(reflect.TypeFor[*one](), mygo.BoundMethod{Name: "A", Func: (*one).A})
	mygo.BindAs("One", &one{})
	mygo.Bind(&Two{})
}
`})
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	if reason == "" && (rerr != nil || !strings.Contains(gen, "Two")) {
		t.Errorf("the table lacks Two, or the app fails: %v %s\n%s", rerr, firstLine(rout), gen)
	}
}

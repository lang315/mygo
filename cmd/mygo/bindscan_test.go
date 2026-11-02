package main

import (
	"runtime"
	"strings"
	"testing"
)

// The scan of mygo build reads every call of Bind in the packages of the build that
// import mygo, so a service bound from the init function of a library, or in a
// branch the run on this computer did not take, is in the table.
func TestScanFindsBindInALibraryInit(t *testing.T) {
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

import "github.com/egoist/mygo"

type Lib struct{}

func (Lib) Ping() string { return "pong" }

func init() { mygo.Bind(&Lib{}) }
`})
	if reason != "" || gen == "" {
		t.Fatalf("Bind kept reflection (%q) for a service bound by the init function of a library", reason)
	}
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	if rerr != nil {
		t.Errorf("the release build fails at start: %v\n%s", rerr, rout)
	}
	if !strings.Contains(gen, "Lib") {
		t.Errorf("the table does not name Lib:\n%s", gen)
	}
}

// A plugin of another author has an unexported service and no table: the scan
// reads the Service of its mygo.Plugin, and Bind keeps reflection, naming the line.
func TestScanThirdPartyPluginWithoutATableKeepsReflection(t *testing.T) {
	gen, reason, _, _, _, _ := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import (
	"github.com/egoist/mygo"

	"bindtest/plug"
)

func main() {
	mygo.Use(plug.New())
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`, "plug/plug.go": `package plug

import "github.com/egoist/mygo"

type service struct{}

func (*service) Ping() string { return "pong" }

func New() mygo.Plugin { return mygo.Plugin{Name: "plug", Service: &service{}} }
`})
	if gen != "" || !strings.Contains(reason, "plug/plug.go:") {
		t.Errorf("gen %q, reason %q: want reflection kept, and the line of the plugin's service", gen, reason)
	}
}

// Bind called with a value the build cannot see the type of (a type parameter, an
// interface, a slice of services) keeps reflection, and says where.
func TestScanUnknownTypesKeepReflection(t *testing.T) {
	for name, c := range map[string]struct{ decl, call, want string }{
		"type parameter": {"func register[T any](v T) { mygo.Bind(v) }", "register(&API{})", "main.go:"},
		"interface":      {"func pick() any { return &API{} }", "mygo.Bind(pick())", "not known"},
		"spread":         {"var all = []any{&API{}}", "mygo.Bind(all...)", "slice of services"},
	} {
		t.Run(name, func(t *testing.T) {
			gen, reason, _, _, _, _ := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import "github.com/egoist/mygo"

type API struct{}

func (API) Hello() string { return "hi" }

` + c.decl + `

func main() {
	` + c.call + `
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
			if gen != "" || !strings.Contains(reason, c.want) || !strings.Contains(reason, "main.go:") {
				t.Errorf("gen %q, reason %q: want reflection kept, with %q and the place of the call", gen, reason, c.want)
			}
		})
	}
}

// A package of the build for the target that does not type-check says nothing of
// what it binds: Bind keeps reflection, and the reason is the error.
func TestScanTypeErrorKeepsReflection(t *testing.T) {
	target := "windows"
	if runtime.GOOS == "windows" {
		target = "linux"
	}
	gen, reason, _, _, _, _ := staticBuild(t, target, map[string]string{"main.go": `package main

import "github.com/egoist/mygo"

type API struct{}

func (API) Hello() string { return "hi" }

func main() {
	mygo.Bind(&API{})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`, "bad_" + target + ".go": `package main

import "github.com/egoist/mygo"

var _ int = "not an int"

var _ = mygo.Version
`})
	if gen != "" || !strings.Contains(reason, "cannot type-check") {
		t.Errorf("gen %q, reason %q: want reflection kept, with the type error", gen, reason)
	}
}

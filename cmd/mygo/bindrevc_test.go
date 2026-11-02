package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// P1: the run's cross-check matches a bound type by package and name only, not
// by pointer-ness. A value S seen by the scan hides a *S the scan missed (a
// Plugin whose Service is set by assignment, which the scan does not read),
// so the table lacks *S and the tagged app panics at start, though the build
// said nothing.
func TestRev338cPointerMismatchHiddenByCrossCheck(t *testing.T) {
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import "github.com/egoist/mygo"

type S struct{}

func (S) Ping() string { return "v" }
func (*S) Pong() string { return "p" }

func main() {
	mygo.BindAs("Val", S{})
	p := mygo.Plugin{Name: "late"}
	p.Service = &S{}
	mygo.Use(p)
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`})
	t.Logf("reason=%q berr=%v", reason, berr)
	t.Logf("gen:\n%s", gen)
	if berr != nil {
		t.Fatalf("build: %s", bout)
	}
	if reason == "" && rerr != nil {
		t.Errorf("BUG: build kept the table (no reason), the release app fails at start: %v\n%s", rerr, rout)
	}
}

// P2: a service the scan does not see and the run does not bind (bound after
// Run, through a function value) is silently bound with no methods under the
// tag: pendBind registers it, checkPending already ran, nothing panics.
func TestRev338cBindAfterRunIsSilent(t *testing.T) {
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": `package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

type A struct{}

func (A) X() string { return "" }

type B struct{}

func (B) Y() string { return "" }

func main() {
	mygo.Bind(&A{})
	out := os.Getenv("MYGO_GENERATE")
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
	reg := mygo.Bind
	reg(&B{})
	p := filepath.Join(filepath.Dir(out), "after.ts")
	if err := mygo.WriteTypeScript(p); err != nil {
		panic(err)
	}
	b, _ := os.ReadFile(p)
	fmt.Printf("LATE BIND OK\n%s\n", b)
}
`})
	if berr != nil || reason != "" {
		t.Fatalf("reason=%q build: %v %s", reason, berr, bout)
	}
	t.Logf("gen:\n%s", gen)
	t.Logf("run err=%v out:\n%s", rerr, rout)
	if rerr == nil && strings.Contains(rout, "LATE BIND OK") && !strings.Contains(rout, "y(") {
		t.Errorf("BUG: Bind(&B{}) after Run returned without a panic and B has no methods in the tagged build")
	}
}

// P3: mygo dev and mygo generate now run generate mode with MYGO_ENV=production,
// so a service bound only under mygo.IsDev() disappears from the TypeScript
// client written during development.
func TestRev338cDevClientLosesDevOnlyService(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dir := bindTestApp(t, `{"name":"Bind Test","identifier":"com.example.bindtest"}`, `package main

import "github.com/egoist/mygo"

type Greeter struct{}

func (Greeter) Hello() string { return "" }

type DevTools struct{}

func (DevTools) Reset() error { return nil }

func main() {
	mygo.Bind(&Greeter{})
	if mygo.IsDev() {
		mygo.Bind(&DevTools{})
	}
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
`)
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "app")
	tk := con.start("build")
	if err := goBuild(context.Background(), c, tk, bin, nil, generatorTags(c.root)...); err != nil {
		t.Fatal(err)
	}
	tk.stop()
	out := filepath.Join(t.TempDir(), "mygo.ts")
	if err := runInGenerateMode(context.Background(), c, bin, out); err != nil {
		t.Fatal(err)
	}
	ts, _ := os.ReadFile(out)
	if !strings.Contains(string(ts), "DevTools") {
		t.Errorf("BUG: the client generated as mygo dev does (generateBindings) has no DevTools, bound under IsDev():\n%s", ts)
	}
}

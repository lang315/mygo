package main

import (
	"runtime"
	"strings"
	"testing"
)

// Each app uses a plugin whose service the run in generate mode does not bind
// (the Use is in a branch the build's run does not take: MYGO_GENERATE_BINDINGS
// is set there, not in the release run). The build must then either name *S in
// the table or keep reflection with a reason; a table without *S and no reason
// is a release app that panics at start, which the build said nothing of.
func rev338dCheck(t *testing.T, body string) {
	t.Helper()
	src := `package main

import (
	"errors"
	"os"

	"github.com/egoist/mygo"
)

var _ = errors.New

type S struct{}

func (*S) Ping() string { return "p" }

type myP struct {
	Name    string
	Service any
	Setup   func() error
}

type W struct{ mygo.Plugin }

func mk() (*S, error) { return &S{}, nil }

type A struct{}

func (*A) X() string { return "" }

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
	gen, reason, berr, bout, rerr, rout := staticBuild(t, runtime.GOOS, map[string]string{"main.go": src})
	if berr != nil {
		t.Fatalf("build: %v\n%s", berr, bout)
	}
	switch {
	case reason != "":
		t.Logf("OK fallback: %s", reason)
	case strings.Contains(gen, "TypeFor[*S]"):
		if rerr != nil {
			t.Errorf("table names *S but run fails: %v\n%s", rerr, firstLine(rout))
		} else {
			t.Logf("OK table names *S")
		}
	case gen == "":
		t.Errorf("no table at all (unexpected, A is bound)")
	default:
		t.Errorf("BUG: silent build (no reason, no *S in table), release run err=%v: %s", rerr, firstLine(rout))
	}
}

func TestRev338dEmbeddedPlugin(t *testing.T) {
	rev338dCheck(t, `		var w W
		w.Name = "late"
		w.Service = &S{}
		mygo.Use(w.Plugin)`)
}

func TestRev338dMultiAssign(t *testing.T) {
	rev338dCheck(t, `		p := mygo.Plugin{Name: "late"}
		var err error
		p.Service, err = mk()
		if err != nil {
			panic(err)
		}
		mygo.Use(p)`)
}

func TestRev338dConversion(t *testing.T) {
	rev338dCheck(t, `		mygo.Use(mygo.Plugin(myP{Name: "late", Service: &S{}}))`)
}

func TestRev338dParenAddr(t *testing.T) {
	rev338dCheck(t, `		p := mygo.Plugin{Name: "late"}
		(&p).Service = &S{}
		mygo.Use(p)`)
}

func TestRev338dIntermediateVar(t *testing.T) {
	rev338dCheck(t, `		svc := &S{}
		p := mygo.Plugin{Name: "late"}
		p.Service = svc
		mygo.Use(p)`)
}

func TestRev338dFuncReturnsPlugin(t *testing.T) {
	rev338dCheck(t, `		mk2 := func() mygo.Plugin { return mygo.Plugin{Name: "late", Service: &S{}} }
		mygo.Use(mk2())`)
}

func TestRev338dNestedField(t *testing.T) {
	rev338dCheck(t, `		var cfg struct{ P mygo.Plugin }
		cfg.P.Name = "late"
		cfg.P.Service = &S{}
		mygo.Use(cfg.P)`)
}

func TestRev338dPluginPtr(t *testing.T) {
	rev338dCheck(t, `		p := &mygo.Plugin{Name: "late"}
		p.Service = &S{}
		mygo.Use(*p)`)
}

func TestRev338dCompoundStructLit(t *testing.T) {
	rev338dCheck(t, `		ps := []mygo.Plugin{{Name: "late"}}
		ps[0].Service = &S{}
		mygo.Use(ps...)`)
}

func TestRev338dFuncValueBind(t *testing.T) {
	rev338dCheck(t, `		reg := mygo.Bind
		reg(&S{})`)
}

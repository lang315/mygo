//go:build !js && !mygo_noinspector

package mygo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type boundTypesPtr struct{}

func (boundTypesPtr) B()  {}
func (*boundTypesPtr) A() {}

type boundTypesValue struct{}

func (boundTypesValue) V(a int) int { return a }

type boundTypesInternal struct{}

func (boundTypesInternal) Hidden() {}

type boundTypeJSONForTest struct {
	Pkg     string   `json:"pkg"`
	Type    string   `json:"type"`
	Pointer bool     `json:"pointer"`
	Methods []string `json:"methods"`
}

func readBoundTypes(t *testing.T, path string) map[string]boundTypeJSONForTest {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var list []boundTypeJSONForTest
	if err := json.Unmarshal(b, &list); err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	byType := map[string]boundTypeJSONForTest{}
	for _, e := range list {
		byType[e.Type] = e
	}
	return byType
}

// Generate mode lists the bound types for mygo build to name: the package, the
// name, whether it is bound as a pointer, and the methods in the order
// reflection lists them. Services of MyGo's own are left out.
func TestGenerateModeWritesTheBoundTypes(t *testing.T) {
	bindForTest(t, "BoundTypesPtr", &boundTypesPtr{})
	bindForTest(t, "BoundTypesValue", boundTypesValue{})
	if err := bind("plugin:boundtypes", &boundTypesInternal{}, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unbind("plugin:boundtypes") })

	path := filepath.Join(t.TempDir(), "bound.json")
	if err := writeBoundTypes(path); err != nil {
		t.Fatal(err)
	}
	got := readBoundTypes(t, path)
	const pkg = "github.com/egoist/mygo"
	if e := got["boundTypesPtr"]; e.Pkg != pkg || !e.Pointer || !slices.Equal(e.Methods, []string{"A", "B"}) {
		t.Errorf("boundTypesPtr: %+v", e)
	}
	if e := got["boundTypesValue"]; e.Pkg != pkg || e.Pointer || !slices.Equal(e.Methods, []string{"V"}) {
		t.Errorf("boundTypesValue: %+v", e)
	}
	if _, ok := got["boundTypesInternal"]; ok {
		t.Error("an internal service was listed")
	}
}

// The path the app takes under MYGO_GENERATE: the hook that writes the
// TypeScript client writes the list when MYGO_GENERATE_BINDINGS names a file,
// and writes no list when it does not.
func TestGenerateModeHookWritesTheListBesideTheClient(t *testing.T) {
	bindForTest(t, "BoundTypesHook", &boundTypesPtr{})
	linkClient()
	dir := t.TempDir()
	client, list := filepath.Join(dir, "mygo.ts"), filepath.Join(dir, "bound.json")

	t.Setenv("MYGO_GENERATE_BINDINGS", "")
	if err := writeClient(client); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(list); err == nil {
		t.Fatal("a list was written without MYGO_GENERATE_BINDINGS")
	}
	t.Setenv("MYGO_GENERATE_BINDINGS", list)
	if err := writeClient(client); err != nil {
		t.Fatal(err)
	}
	if got := readBoundTypes(t, list); got["boundTypesPtr"].Type == "" {
		t.Errorf("the list has no boundTypesPtr: %v", got)
	}
	if _, err := os.Stat(client); err != nil {
		t.Errorf("the TypeScript client was not written: %v", err)
	}
}

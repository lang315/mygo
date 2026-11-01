//go:build !js && !mygo_noinspector

package mygo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
)

// boundTypeJSON is what generate mode says of a bound service so that mygo
// build can name its methods in the code it generates (cmd/mygo/bindgen.go):
// the package and name of its type, whether it was bound as a pointer, and its
// methods in the order reflection lists them.
type boundTypeJSON struct {
	Pkg     string   `json:"pkg"`
	Type    string   `json:"type"`
	Pointer bool     `json:"pointer"`
	Methods []string `json:"methods"`
}

// writeBoundTypes writes the bound services of the app, those of its own and
// not MyGo's, to path as JSON. Generate mode (MYGO_GENERATE) does it when
// MYGO_GENERATE_BINDINGS names the file.
func writeBoundTypes(path string) error {
	list := []boundTypeJSON{}
	ipc.RLock()
	for _, s := range ipc.services {
		if s.internal {
			continue
		}
		t := s.typ
		b := boundTypeJSON{Pointer: t.Kind() == reflect.Pointer}
		if b.Pointer {
			t = t.Elem()
		}
		b.Pkg, b.Type = t.PkgPath(), t.Name()
		for _, m := range s.methods {
			b.Methods = append(b.Methods, m.name)
		}
		list = append(list, b)
	}
	ipc.RUnlock()
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

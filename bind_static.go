//go:build mygo_static && !js

package mygo

import (
	"fmt"
	"reflect"
)

// staticFallback lists the methods of a type that has no table. It is nil in an
// app, which must not use reflection to list them; the tests of the package
// bind many services mygo build has not seen, and set it (bind_static_fallback_test.go).
var staticFallback func(t reflect.Type, v reflect.Value, service string) ([]*method, error)

// listMethods returns the methods of the service v of type t from the table
// mygo build generated (BindMethods), without reflect.Type.Method and
// Value.Method, which make the linker keep every exported method of the
// program.
func listMethods(t reflect.Type, v reflect.Value, service string) ([]*method, error) {
	e, ok := staticMethods.Load(t)
	if !ok {
		if staticFallback != nil {
			return staticFallback(t, v, service)
		}
		return nil, &noTableError{fmt.Sprintf("mygo: %s is bound, but this build has no list of its methods. mygo build lists them by running the app in generate mode, so a service must be bound before App.Run and whatever the conditions of the run (not after Run, nor only under a flag or an environment variable that was not set then). Bind it so, or build without the tag mygo_static to bind by reflection", t)}
	}
	ms := e.([]BoundMethod)
	if len(ms) != t.NumMethod() {
		return nil, fmt.Errorf("mygo: the methods of %s changed since mygo build listed them (%d listed, %d now): build with mygo build again", t, len(ms), t.NumMethod())
	}
	out := make([]*method, 0, len(ms))
	for _, m := range ms {
		fn := reflect.ValueOf(m.Func)
		mi, err := newMethodRecv(service, m.Name, fn, v)
		if err != nil {
			return nil, err
		}
		mi.pc = fn.Pointer()
		if m.Decl != nil {
			mi.pc = reflect.ValueOf(m.Decl).Pointer()
		}
		out = append(out, mi)
	}
	return out, nil
}

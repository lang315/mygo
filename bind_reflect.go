//go:build !mygo_static && !js

package mygo

import "reflect"

// listMethods returns the methods of the service v of type t. Reflection lists
// them, which makes the linker keep every exported method of the program (see
// "Binary size" in the architecture guide); builds by mygo build use the
// tag mygo_static and a table instead (bind_static.go).
func listMethods(t reflect.Type, v reflect.Value, service string) ([]*method, error) {
	var out []*method
	for i := range t.NumMethod() {
		m := t.Method(i)
		mi, err := newMethod(service, m.Name, v.Method(m.Index))
		if err != nil {
			return nil, err
		}
		mi.pc = methodPC(t, m.Name)
		out = append(out, mi)
	}
	return out, nil
}

// methodPC returns the entry of the method's own code rather than the
// wrapper Go generates for value receivers called through a pointer.
func methodPC(t reflect.Type, name string) uintptr {
	if t.Kind() == reflect.Pointer {
		if m, ok := t.Elem().MethodByName(name); ok {
			return m.Func.Pointer()
		}
	}
	if m, ok := t.MethodByName(name); ok {
		return m.Func.Pointer()
	}
	return 0
}

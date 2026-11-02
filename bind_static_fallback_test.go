//go:build mygo_static && !js

package mygo

import "reflect"

// The tests bind hundreds of services that mygo build has no list of. Under
// mygo_static they get the methods by reflection, which the test binary can
// afford; an app cannot, and has no fallback (staticFallback is nil in it). The
// tests of the table itself switch the fallback off.
func init() { staticFallback = listByReflection }

// listByReflection is what a build without the tag does: it lists the
// methods with reflect.Type.Method and Value.Method.
func listByReflection(t reflect.Type, v reflect.Value, service string) ([]*method, error) {
	var out []*method
	for i := range t.NumMethod() {
		m := t.Method(i)
		mi, err := newMethod(service, m.Name, v.Method(m.Index))
		if err != nil {
			return nil, err
		}
		out = append(out, mi)
	}
	return out, nil
}

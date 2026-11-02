// Package bindcheck holds the check that a table of methods given to
// mygo.BindMethods lists what reflection lists. A plugin whose service is
// bound inside the plugin (the types are unexported, so mygo build cannot
// name them) keeps such a table next to the service and tests it with this, so
// a method added to the service and not to the table fails a test and not the
// start of an app built with the tag mygo_static.
package bindcheck

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo"
)

// Table fails t unless table lists every exported method of typ, in the order
// reflection lists them, each as a method expression of its signature.
func Table(t testing.TB, typ reflect.Type, table []mygo.BoundMethod) {
	t.Helper()
	if len(table) != typ.NumMethod() {
		t.Errorf("%s has %d methods, the table lists %d", typ, typ.NumMethod(), len(table))
	}
	for i := range min(len(table), typ.NumMethod()) {
		m := typ.Method(i)
		if table[i].Name != m.Name {
			t.Errorf("%s: method %d is %s, the table has %s", typ, i, m.Name, table[i].Name)
			continue
		}
		if got := reflect.TypeOf(table[i].Func); got != m.Type {
			t.Errorf("%s.%s: the table has a %s, the method is a %s", typ, m.Name, got, m.Type)
		}
	}
}

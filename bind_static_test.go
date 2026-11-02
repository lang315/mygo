//go:build mygo_static && !js

package mygo

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type staticSvc struct{ n int }

func (s staticSvc) Value(a int) int     { return a + s.n }
func (s *staticSvc) Pointer(a int) int  { return a * 2 }
func (s *staticSvc) Fail() (int, error) { return 0, nil }

type staticEmbed struct{ staticSvc }

type staticEmbedPtr struct{ *staticSvc }

// staticStream has a context and a channel, as a streaming method has.
type staticStream struct{}

func (staticStream) Count(ctx context.Context, n int, ch *Channel[int]) error { return nil }
func (staticStream) Sum(prefix string, rest ...int) int                       { return len(rest) }

func init() {
	BindMethods(reflect.TypeFor[staticSvc](), BoundMethod{Name: "Value", Func: staticSvc.Value})
	BindMethods(reflect.TypeFor[*staticSvc](),
		BoundMethod{Name: "Fail", Func: (*staticSvc).Fail},
		BoundMethod{Name: "Pointer", Func: (*staticSvc).Pointer},
		BoundMethod{Name: "Value", Func: (*staticSvc).Value})
	BindMethods(reflect.TypeFor[staticEmbed](), BoundMethod{Name: "Value", Func: staticEmbed.Value})
	BindMethods(reflect.TypeFor[staticEmbedPtr](),
		BoundMethod{Name: "Fail", Func: staticEmbedPtr.Fail},
		BoundMethod{Name: "Pointer", Func: staticEmbedPtr.Pointer},
		BoundMethod{Name: "Value", Func: staticEmbedPtr.Value})
	BindMethods(reflect.TypeFor[staticStream](),
		BoundMethod{Name: "Count", Func: staticStream.Count},
		BoundMethod{Name: "Sum", Func: staticStream.Sum})
}

// invoke calls the method with its arguments, as call does: after the receiver.
func (m *method) invoke(in []reflect.Value) []reflect.Value {
	return m.fn.Call(append([]reflect.Value{m.recv}, in...))
}

// listWithoutTheFallback lists the methods of v from its table only.
func listWithoutTheFallback(t *testing.T, v any, service string) ([]*method, error) {
	t.Helper()
	saved := staticFallback
	staticFallback = nil
	defer func() { staticFallback = saved }()
	return listMethods(reflect.TypeOf(v), reflect.ValueOf(v), service)
}

// TestStaticBindMatchesReflectForEveryReceiverKind: the table lists what
// reflection lists, in its order (sorted by name), for T, *T and the types that
// embed one, and every method is read as reflection reads it.
func TestStaticBindMatchesReflectForEveryReceiverKind(t *testing.T) {
	for _, c := range []struct {
		v    any
		want []string
	}{
		{staticSvc{}, []string{"Value"}},
		{&staticSvc{}, []string{"Fail", "Pointer", "Value"}},
		{staticEmbed{}, []string{"Value"}},
		{staticEmbedPtr{&staticSvc{}}, []string{"Fail", "Pointer", "Value"}},
		{staticStream{}, []string{"Count", "Sum"}},
	} {
		got, err := listWithoutTheFallback(t, c.v, "S")
		if err != nil {
			t.Fatalf("%T: %v", c.v, err)
		}
		var names []string
		for _, m := range got {
			names = append(names, m.name)
		}
		if !slices.Equal(names, c.want) {
			t.Errorf("%T: %v, want %v", c.v, names, c.want)
		}
		// The reflection lister, which the tests keep for the services they bind
		// without a table, lists the same things the same way.
		want, err := listByReflection(reflect.TypeOf(c.v), reflect.ValueOf(c.v), "S")
		if err != nil {
			t.Fatal(err)
		}
		for i, m := range got {
			w := want[i]
			if m.name != w.name || m.ctx != w.ctx || m.variadic != w.variadic || m.streams != w.streams ||
				m.hasErr != w.hasErr || m.result != w.result || !slices.Equal(m.params, w.params) || !slices.Equal(m.chans, w.chans) {
				t.Errorf("%T.%s: read as %+v, reflection reads %+v", c.v, m.name, m, w)
			}
		}
	}
}

// A call goes to the method of the value that was bound.
func TestStaticBindCallsTheBoundValue(t *testing.T) {
	ms, err := listWithoutTheFallback(t, &staticSvc{n: 40}, "S")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if m.name != "Value" {
			continue
		}
		out := m.invoke([]reflect.Value{reflect.ValueOf(2)})
		if got := out[0].Int(); got != 42 {
			t.Errorf("Value(2) on n=40 = %d, want 42", got)
		}
		return
	}
	t.Fatal("no Value method")
}

type staleSvc struct{}

func (staleSvc) A() {}
func (staleSvc) B() {} // not in the table below

func init() { BindMethods(reflect.TypeFor[staleSvc](), BoundMethod{Name: "A", Func: staleSvc.A}) }

func TestStaticBindPanicsWhenTheTableIsStale(t *testing.T) {
	_, err := listWithoutTheFallback(t, staleSvc{}, "Stale")
	if err == nil || !strings.Contains(err.Error(), "mygo build") {
		t.Fatalf("err = %v, want one that says to run mygo build", err)
	}
}

type unlistedSvc struct{}

func (unlistedSvc) A() {}

func TestStaticBindOfAnUnlistedType(t *testing.T) {
	_, err := listWithoutTheFallback(t, unlistedSvc{}, "U")
	if err == nil || !strings.Contains(err.Error(), "mygo build") {
		t.Fatalf("err = %v", err)
	}
}

type lateSvc struct{}

func (lateSvc) Late() string { return "late" }

type neverSvc struct{}

func (neverSvc) Never() {}

// clearPending drops what a test left pending, and its service.
func clearPending(t *testing.T, name string) {
	t.Cleanup(func() {
		pending.Lock()
		pending.list = nil
		pending.Unlock()
		unbind(name)
	})
}

// A build with the tag runs the init function that registers the table after the
// package-level variables of main and the init functions of its imports, where
// Bind may be called. Such a Bind does not fail: the service takes its place, in
// order, and gets its methods when the table arrives.
func TestStaticBindBeforeTheTableWaitsForIt(t *testing.T) {
	saved := staticFallback
	staticFallback = nil
	defer func() { staticFallback = saved }()
	clearPending(t, "Late")
	if err := bind("Late", &lateSvc{}, false); err != nil {
		t.Fatalf("bind before the table: %v", err)
	}
	if lookupMethod("Late.Late") != nil {
		t.Fatal("a method was bound with no table")
	}
	BindMethods(reflect.TypeFor[*lateSvc](), BoundMethod{Name: "Late", Func: (*lateSvc).Late})
	if lookupMethod("Late.Late") == nil {
		t.Fatal("the table did not give the waiting service its method")
	}
	checkPending() // nothing is pending: no panic
}

// A service whose type the table never gets is a failure at App.Run, with the
// cause, never a silent partial app.
func TestStaticBindWithoutATableFailsAtRun(t *testing.T) {
	saved := staticFallback
	staticFallback = nil
	defer func() { staticFallback = saved }()
	clearPending(t, "Never")
	if err := bind("Never", &neverSvc{}, false); err != nil {
		t.Fatal(err)
	}
	defer func() {
		err, _ := recover().(error)
		if err == nil || !strings.Contains(err.Error(), "mygo build") || !strings.Contains(err.Error(), "neverSvc") {
			t.Fatalf("checkPending panicked with %v, want the cause: no list of the methods of *neverSvc, build with mygo build", err)
		}
	}()
	checkPending()
	t.Fatal("checkPending did not panic")
}

func TestStaticBindAsTwiceOnOneType(t *testing.T) {
	bindForTest(t, "StaticOne", &staticSvc{n: 1})
	bindForTest(t, "StaticTwo", &staticSvc{n: 2})
	for _, name := range []string{"StaticOne.Value", "StaticTwo.Value"} {
		if lookupMethod(name) == nil {
			t.Errorf("%s is not bound", name)
		}
	}
	// Each service calls its own value.
	one, two := lookupMethod("StaticOne.Value"), lookupMethod("StaticTwo.Value")
	zero := []reflect.Value{reflect.ValueOf(0)}
	if a, b := one.invoke(zero)[0].Int(), two.invoke(zero)[0].Int(); a != 1 || b != 2 {
		t.Errorf("values = %d, %d, want 1, 2", a, b)
	}
}

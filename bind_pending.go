//go:build mygo_static && !js

package mygo

import (
	"fmt"
	"reflect"
	"sync"
)

// A noTableError says a build with the tag mygo_static has no list of the methods
// of a service (yet).
type noTableError struct{ msg string }

func (e *noTableError) Error() string { return e.msg }

// A pendingService is bound before the table of its methods is.
type pendingService struct {
	s   *service
	v   reflect.Value
	err *noTableError
}

var pending struct {
	sync.Mutex
	list []pendingService
}

// registerPending registers s with no methods; resolvePending gives it them.
func registerPending(s *service, v reflect.Value, err *noTableError) error {
	if e := register(s); e != nil {
		return e
	}
	pending.Lock()
	pending.list = append(pending.list, pendingService{s, v, err})
	pending.Unlock()
	return nil
}

// resolvePending gives the pending services of type t their methods, now that
// the table has them. It panics as Bind does when they are not valid.
func resolvePending(t reflect.Type) {
	pending.Lock()
	var todo []pendingService
	rest := pending.list[:0]
	for _, p := range pending.list {
		if p.s.typ == t {
			todo = append(todo, p)
		} else {
			rest = append(rest, p)
		}
	}
	pending.list = rest
	pending.Unlock()
	for _, p := range todo {
		ms, err := listMethods(t, p.v, p.s.name)
		if err == nil && len(ms) == 0 {
			err = fmt.Errorf("mygo: %s has no exported methods to bind", t)
		}
		if err != nil {
			panic(err)
		}
		ipc.Lock()
		p.s.methods = ms
		for _, m := range ms {
			ipc.methods[p.s.name+"."+m.name] = m
		}
		ipc.Unlock()
	}
}

// checkPending panics, from App.Run, when a service is still without the list of
// its methods: the table of mygo build does not have its type.
func checkPending() {
	pending.Lock()
	defer pending.Unlock()
	if len(pending.list) > 0 {
		panic(pending.list[0].err)
	}
}

// pendBind is what bind does with the error of listMethods: a service the table
// does not have yet waits for it (handled), any other error is bind's to return.
func pendBind(err error, s *service, v reflect.Value) (handled bool, rerr error) {
	nt, ok := err.(*noTableError)
	if !ok {
		return false, nil
	}
	return true, registerPending(s, v, nt)
}

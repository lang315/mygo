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
	// closed is set by App.Run: a service bound after it has no table to wait for.
	closed bool
}

// registerPending registers s with no methods; resolvePending gives it them.
//
// The check of closed and the append are one hold of the lock, so a Bind that
// races App.Run's checkPending is either in the list when it checks or is refused.
// The order of the locks is pending, then ipc; nothing takes them the other way.
func registerPending(s *service, v reflect.Value, err *noTableError) error {
	pending.Lock()
	defer pending.Unlock()
	if pending.closed {
		return err // App.Run has checked: nothing is going to give it methods
	}
	if e := register(s); e != nil {
		return e
	}
	pending.list = append(pending.list, pendingService{s, v, err})
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
	pending.closed = true
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

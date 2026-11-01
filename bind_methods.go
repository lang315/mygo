package mygo

import (
	"reflect"
	"sync"
)

// BoundMethod is a method of a bound service as the code mygo build generates
// names it: Func is a method expression, (*T).Name or T.Name.
type BoundMethod struct {
	Name string
	Func any
}

var staticMethods sync.Map // reflect.Type → []BoundMethod

// BindMethods is for the code mygo build generates, not for apps: it gives
// Bind the methods of t without the reflection that keeps every method of a
// program (built with the tag mygo_static). Methods are in name order, the
// order reflection lists them in.
func BindMethods(t reflect.Type, methods ...BoundMethod) {
	staticMethods.Store(t, methods)
}

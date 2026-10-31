//go:build !js

package mygo

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"sync/atomic"
)

// jsonOptions are used for every value crossing the IPC boundary.
var jsonOptions = json.JoinOptions(
	jsonv1.FormatDurationAsNano(true),
	jsontext.EscapeForJS(true),
)

// A codecFuncs reads and writes the values of the arguments, results and events
// of the IPC with encoding/json/v2, which brings reflection-driven encoding,
// the unicode tables and 500 KB of code. Only an app that binds a service or
// declares an event has values to code, so Bind, BindAs and NewEvent link it
// (linkCodec); the envelope of a call, the windows' own methods and MyGo's own
// events never need it.
type codecFuncs struct {
	unmarshal func(raw []byte, v any) error
	marshal   func(v any) ([]byte, error)
}

var codec atomic.Pointer[codecFuncs]

func linkCodec() {
	codec.CompareAndSwap(nil, &codecFuncs{
		unmarshal: func(raw []byte, v any) error { return json.Unmarshal(raw, v, jsonOptions) },
		marshal:   func(v any) ([]byte, error) { return json.Marshal(v, jsonOptions) },
	})
}

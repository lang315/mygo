//go:build !js

package mygo

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
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

var errNoCodec = errors.New("mygo: no codec is linked: Bind or NewEvent links it before values are coded")

// loadCodec returns the codec, or an error when none is linked. Bind and
// NewEvent link it, so a value of a bound method or an event always finds one;
// the error is for a caller that codes a value without either.
func loadCodec() (*codecFuncs, error) {
	if c := codec.Load(); c != nil {
		return c, nil
	}
	return nil, errNoCodec
}

func linkCodec() {
	codec.CompareAndSwap(nil, &codecFuncs{
		unmarshal: func(raw []byte, v any) error { return json.Unmarshal(raw, v, jsonOptions) },
		marshal:   func(v any) ([]byte, error) { return json.Marshal(v, jsonOptions) },
	})
}

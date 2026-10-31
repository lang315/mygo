//go:build !js

package mygo

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"testing"
)

// The envelope of a call and the messages of a page are read with jsonlite; the
// oracles are the structs encoding/json/v2 read them into before.

type rawArg []byte

func (v *rawArg) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	raw, err := dec.ReadValue()
	*v = rawArg(raw)
	return err
}

type callV2 struct {
	ID int64    `json:"id"`
	K  string   `json:"k"`
	M  string   `json:"m"`
	A  []rawArg `json:"a"`
}

var callDocs = []string{
	`{"t":"call","id":1,"k":"x","m":"A.B","a":[1,"s",{"a":1}]}`,
	`{"id":1,"id":2}`, `{"ID":1}`, `{"a":null}`, `{"a":[]}`, `null`, `{"k":"\ud800"}`,
	`{"a":[{"x":1,"x":2}]}`, `{"m":1}`, `{"id":1.0}`, `{"id":1e2}`, `{"id":-0}`, "{\"k\":\"\xff\"}",
	`{"a":[1,]}`, ` {"id":1} `, `{"id":1}x`, `{"id":5}`, `{"a":[1],"z":{"y":[{}]}}`,
}

func FuzzParseCallIsV2(f *testing.F) {
	for _, d := range callDocs {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		got, gerr := parseCall(b)
		var want callV2
		werr := json.Unmarshal(b, &want)
		if (gerr == nil) != (werr == nil) {
			t.Fatalf("%q: jsonlite err=%v, v2 err=%v", b, gerr, werr)
		}
		if gerr != nil {
			return
		}
		if got.id != want.ID || got.token != want.K || got.name != want.M || len(got.args) != len(want.A) || (got.args == nil) != (want.A == nil) {
			t.Fatalf("%q: jsonlite %+v, v2 %+v", b, got, want)
		}
		for i := range got.args {
			if !bytes.Equal(got.args[i], want.A[i]) {
				t.Fatalf("%q: argument %d is %q, v2 reads %q", b, i, got.args[i], want.A[i])
			}
		}
	})
}

type pageMessageV2 struct {
	T  string  `json:"t"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	C  int64   `json:"c"`
	K  string  `json:"k"`
	N  int64   `json:"n"`
	V  string  `json:"v"`
	W  float64 `json:"w"`
	H  float64 `json:"h"`
	SW float64 `json:"sw"`
	SH float64 `json:"sh"`
	L  string  `json:"l"`
	LT string  `json:"lt"`
	I  string  `json:"i"`
	M  string  `json:"m"`
	S  string  `json:"s"`
	E  bool    `json:"e"`
}

var pageMessageDocs = []string{
	`{"t":"drop","x":1.5,"y":-2e3}`, `{"t":"chan","c":1,"k":"a","n":3}`, `{"e":true,"lt":"x","sw":1}`,
	`{"x":1e400}`, `{"c":1.0}`, `{"T":"x"}`, `{"t":"a","t":"b"}`, `{"e":1}`, `{"x":"1"}`, `{"w":-0}`,
}

func FuzzParsePageMessageIsV2(f *testing.F) {
	for _, d := range pageMessageDocs {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		got, gerr := parsePageMessage(b)
		var want pageMessageV2
		werr := json.Unmarshal(b, &want)
		if (gerr == nil) != (werr == nil) {
			t.Fatalf("%q: jsonlite err=%v, v2 err=%v", b, gerr, werr)
		}
		if gerr == nil && fmt.Sprint(pageMessage(want)) != fmt.Sprint(got) {
			t.Fatalf("%q: jsonlite %+v, v2 %+v", b, got, want)
		}
	})
}

// An app that binds nothing has no codec: a call of a service it has not got,
// and the windows' own methods, are still answered without it.
func TestCallsAreAnsweredWithoutACodec(t *testing.T) {
	saved := codec.Load()
	codec.Store(nil)
	defer codec.Store(saved)
	w := bareWindow(false)
	for _, msg := range []string{
		`{"t":"call","id":1,"k":"tok","m":"Nope.X","a":[1]}`,
		`{"t":"call","id":2,"k":"tok","m":"mygo:window.SetTitle","a":[5]}`,
		`{"t":"call","id":3,"k":"tok","m":"mygo:window.SetTitle","a":[" <>"]}`,
		`{"t":"call","id":4,"k":"tok","m":"mygo:window.IsMaximized","a":[]}`,
		`{"t":"call","id":5,"k":"tok","m":"mygo:window.Minimize","a":[{"x":1}]}`,
	} {
		if rec := runGoldenCall(w, true, msg); len(rec.Replies) != 1 {
			t.Errorf("%s: %d replies, %q", msg, len(rec.Replies), rec.Log)
		}
	}
}

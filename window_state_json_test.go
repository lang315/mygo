package mygo

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// windowStateDocs are seeds: files an older version wrote, and the ways a file
// can differ from them or break.
var windowStateDocs = []string{
	``, `null`, `{}`, `[]`, `1`, `"x"`, `{"main":{"x":10,"y":20,"width":800,"height":600}}`,
	"{\n  \"main\": {\n    \"x\": -5,\n    \"y\": 0,\n    \"width\": 640,\n    \"height\": 480,\n    \"maximized\": true,\n    \"fullScreen\": true\n  }\n}",
	`{"a":{"x":1},"a":{"y":2}}`, `{"a":{"x":1,"x":2}}`, `{"a":{"X":1,"WIDTH":2,"Fullscreen":true,"FULLSCREEN":true}}`,
	`{"a":null}`, `{"a":{"x":null,"maximized":null}}`, `{"a":1}`, `{"a":[]}`, `{"a":{"x":1.5}}`, `{"a":{"x":1e2}}`,
	`{"a":{"x":"1"}}`, `{"a":{"maximized":1}}`, `{"a":{"unknown":{"b":[1,2,null]},"width":3}}`,
	`{"a":{"x":9223372036854775807}}`, `{"a":{"x":9223372036854775808}}`, `{"a":{"x":-9223372036854775808}}`,
	`{"é😀":{"x":1}}`, `{"<&>":{"x":1}}`, "{\"\xff\":{\"x\":1}}", `{"a":{"x":1}} `, `{"a":{"x":1}} x`, `{"a":{"x":1},}`,
	`{"a":{"x":01}}`, `{"a":{"x":-0}}`, `{"a":{"x":0.0}}`, `{"a":{"fullScreen":true,"maximized":false}}`,
	"\xef\xbb\xbf{}", `{"a":{"x":1}`, `{"a":{"x":1}}}`,
}

func sameWindowStates(t *testing.T, b []byte) {
	t.Helper()
	var want map[string]WindowState
	werr := json.Unmarshal(b, &want)
	got, gerr := decodeWindowStates(b)
	if (werr == nil) != (gerr == nil) {
		t.Fatalf("%q: encoding/json error %v, jsonlite error %v", b, werr, gerr)
	}
	if werr == nil && !reflect.DeepEqual(got, want) {
		t.Fatalf("%q: encoding/json reads %+v, jsonlite %+v", b, want, got)
	}
}

func TestWindowStatesAgreeWithJSON(t *testing.T) {
	for _, d := range windowStateDocs {
		sameWindowStates(t, []byte(d))
	}
}

func FuzzWindowStatesAgreeWithJSON(f *testing.F) {
	for _, d := range windowStateDocs {
		f.Add([]byte(d))
	}
	f.Fuzz(sameWindowStates)
}

// What is written is what json.MarshalIndent writes, byte for byte, and reads back.
func sameEncoding(t *testing.T, states map[string]WindowState) {
	t.Helper()
	want, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got := encodeWindowStates(states)
	if !bytes.Equal(got, want) {
		t.Fatalf("%+v: jsonlite writes\n%s\nencoding/json writes\n%s", states, got, want)
	}
	// What it reads back is what encoding/json reads back from the same bytes: a
	// key that is not valid UTF-8 is written as U+FFFD by both, so it is not the
	// original key that comes back.
	var wantBack map[string]WindowState
	if err := json.Unmarshal(want, &wantBack); err != nil {
		t.Fatal(err)
	}
	back, err := decodeWindowStates(got)
	if err != nil || !reflect.DeepEqual(back, wantBack) {
		t.Fatalf("%s reads back as %+v, %v; encoding/json reads %+v", got, back, err, wantBack)
	}
}

func TestWindowStatesAreWrittenAsMarshalIndent(t *testing.T) {
	sameEncoding(t, nil)
	sameEncoding(t, map[string]WindowState{})
	sameEncoding(t, map[string]WindowState{"main": {X: 1, Y: -2, Width: 800, Height: 600}})
	sameEncoding(t, map[string]WindowState{
		"b":             {Maximized: true},
		"a":             {FullScreen: true, Maximized: true, X: 1 << 40},
		"<&> é😀":        {Width: 1},
		"quote\"back\\": {},
	})
}

func FuzzWindowStatesAreWrittenAsMarshalIndent(f *testing.F) {
	f.Add("main", 1, 2, 3, 4, true, false, "other")
	f.Add("<&> ", -1, 1<<40, 0, -1<<40, false, true, "")
	f.Fuzz(func(t *testing.T, k string, x, y, w, h int, max, full bool, k2 string) {
		sameEncoding(t, map[string]WindowState{
			k:  {X: x, Y: y, Width: w, Height: h, Maximized: max, FullScreen: full},
			k2: {X: y, Y: x},
		})
	})
}

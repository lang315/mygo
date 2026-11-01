//go:build !js

package mygo

import (
	"bytes"
	"encoding/json/v2"
	"testing"
)

// MyGo's own events are written by hand; they are what the codec wrote.
func TestEventPayloadsAreV2(t *testing.T) {
	strs := []string{"", "a.txt", "dir/é😀 <&>.txt", "a b ", "q\"\\\n\x00", "\xff", "a\xe2\x80"}
	check := func(what string, got []byte, ok bool, v any) {
		t.Helper()
		want, err := json.Marshal(v, jsonOptions)
		if ok != (err == nil) || ok && !bytes.Equal(got, want) {
			t.Errorf("%s %+v: written %s, %v; encoding/json/v2 writes %s, %v", what, v, got, ok, want, err)
		}
	}
	for _, s := range strs {
		for _, edited := range []bool{false, true} {
			d := DocumentState{Name: s, Path: "/" + s, Edited: edited, ReadOnly: !edited, Conflict: edited, Deleted: !edited}
			got, ok := d.appendJSON(nil)
			check("DocumentState", got, ok, d)
		}
		for _, paths := range [][]string{nil, {}, {s}, {s, "b", s}} {
			e := FileDropEvent{Paths: paths, X: -3, Y: 1 << 40}
			got, ok := e.appendJSON(nil)
			check("FileDropEvent", got, ok, e)
		}
	}
}

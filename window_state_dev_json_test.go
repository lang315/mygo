//go:build !js

package mygo

import (
	"encoding/json"
	"reflect"
	"testing"
)

// mygo dev's file of windows is read and written as encoding/json reads and
// writes a []devWindow, byte for byte.
func TestDevWindowsAgreeWithJSON(t *testing.T) {
	lists := [][]devWindow{
		nil,
		{},
		{{}},
		{
			{Asked: devWindowAsk{X: -5, Y: 7, Width: 800, Height: 600, Center: true, ContentSize: true},
				State: WindowState{X: 10, Y: -20, Width: 640, Height: 480, Maximized: true, PixelX: 20, PixelHeight: 960}},
			{Asked: devWindowAsk{Maximized: true, FullScreen: true}, State: WindowState{FullScreen: true, PixelY: -3, PixelWidth: 1}},
		},
	}
	for _, list := range lists {
		want, err := json.Marshal(list)
		if err != nil {
			t.Fatal(err)
		}
		if got := encodeDevWindows(list); string(got) != string(want) {
			t.Errorf("encodeDevWindows(%+v) = %s, json.Marshal writes %s", list, got, want)
		}
		got, err := decodeDevWindows(want)
		if err != nil {
			t.Fatalf("decodeDevWindows(%s): %v", want, err)
		}
		var back []devWindow
		if err := json.Unmarshal(want, &back); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, back) {
			t.Errorf("decodeDevWindows(%s) = %+v, json.Unmarshal reads %+v", want, got, back)
		}
	}
	// What json.Unmarshal also reads: names of another case, nulls, members
	// of other names.
	for _, doc := range []string{
		`[{"ASKED":{"WIDTH":3,"center":true},"state":{"X":1,"pixelx":2,"zz":[1]},"other":{}}]`,
		`[null,{"asked":null,"state":{"width":null}}]`,
	} {
		got, err := decodeDevWindows([]byte(doc))
		var want []devWindow
		werr := json.Unmarshal([]byte(doc), &want)
		if (err == nil) != (werr == nil) || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: decodeDevWindows = %+v, %v; json.Unmarshal = %+v, %v", doc, got, err, want, werr)
		}
	}
	if _, err := decodeDevWindows([]byte(`[{"asked":`)); err == nil {
		t.Error("a damaged file read without an error")
	}
}

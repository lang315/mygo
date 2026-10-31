package jsonlite

import (
	"bytes"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"reflect"
	"testing"
)

var stringsDocs = []string{
	``, `null`, `[]`, `{}`, `1`, `"a"`, `["a"]`, `["a","b"]`, ` [ "a" , "b" ] `, `["a",]`, `[,]`, `["a" "b"]`,
	`[null]`, `["a",null,"b"]`, `[1]`, `["a",1]`, `[["a"]]`, `[{}]`, `["é","😀","\ud83d"]`,
	"[\"\xff\"]", "[\"a\x00\"]", `["a\x"]`, `["a`, `[`, `["a"]]`, `["a"] x`, "\xef\xbb\xbf[]", `[true]`,
}

// Recent documents are read as the encoding/json/v2 that wrote them reads them: the same
// strings where it reads any. Elsewhere jsonlite does what encoding/json (v1) does,
// whose older versions wrote the file too: it reads a lone surrogate or a byte that is not
// UTF-8 as U+FFFD, which v2 refuses, and refuses what v1 refuses.
func sameStrings(t *testing.T, b []byte) {
	t.Helper()
	got, gerr := DecodeStrings(b)
	var want []string
	if werr := jsonv2.Unmarshal(b, &want); werr == nil {
		if gerr != nil {
			t.Fatalf("%q: encoding/json/v2 reads %q, jsonlite refuses: %v", b, want, gerr)
		}
		if !reflect.DeepEqual(append([]string{}, got...), append([]string{}, want...)) {
			t.Fatalf("%q: encoding/json/v2 reads %q, jsonlite %q", b, want, got)
		}
	}
	var want1 []string
	werr1 := json.Unmarshal(b, &want1)
	if (werr1 == nil) != (gerr == nil) {
		t.Fatalf("%q: encoding/json error %v, jsonlite error %v", b, werr1, gerr)
	}
	if werr1 == nil && !reflect.DeepEqual(append([]string{}, got...), append([]string{}, want1...)) {
		t.Fatalf("%q: encoding/json reads %q, jsonlite %q", b, want1, got)
	}
}

func TestStringsAgreeWithJSON(t *testing.T) {
	for _, d := range stringsDocs {
		sameStrings(t, []byte(d))
	}
}

func FuzzStringsAgreeWithJSON(f *testing.F) {
	for _, d := range stringsDocs {
		f.Add([]byte(d))
	}
	f.Fuzz(sameStrings)
}

func FuzzEncodeStringsIsMarshal(f *testing.F) {
	f.Add("a", "b", true)
	f.Add("<&> é😀\x00\x1f", "", false)
	f.Add("\xff", "x", true)
	f.Fuzz(func(t *testing.T, a, b string, nilList bool) {
		list := []string{a, b}
		if nilList {
			list = nil
		}
		want, werr := jsonv2.Marshal(list)
		got, gerr := EncodeStrings(list)
		if (werr == nil) != (gerr == nil) {
			t.Fatalf("%q: encoding/json/v2 error %v, jsonlite error %v", list, werr, gerr)
		}
		if werr == nil && !bytes.Equal(got, want) {
			t.Fatalf("%q: jsonlite writes %s, encoding/json/v2 %s", list, got, want)
		}
		if gerr == nil {
			if back, err := DecodeStrings(got); err != nil || !reflect.DeepEqual(append([]string{}, back...), append([]string{}, list...)) {
				t.Fatalf("%s reads back as %q, %v", got, back, err)
			}
		}
	})
}

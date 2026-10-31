//go:build !js

package mygo

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
)

// evalResultV2 is how the result of an eval was read with encoding/json/v2.
type evalResultV2 struct {
	OK bool           `json:"ok"`
	V  jsontext.Value `json:"v"`
	E  string         `json:"e"`
}

var evalResultDocs = []string{
	`{"ok":true,"v":1}`, `{"ok":true,"v":{"a":[1,"x",null]},"e":""}`, `{"ok":false,"e":"boom"}`,
	`null`, `{}`, `[]`, `1`, `{"ok":null,"e":null,"v":null}`, `{"v":null}`,
	`{"ok":true,"ok":false}`, `{"e":"a","e":"b"}`, `{"v":1,"v":2}`, `{"x":1,"x":2}`,
	`{"ok":true,"v":{"a":1,"a":2}}`,
	"{\"e\":\"\xff\"}", "{\"e\":\"a\xff\\n\"}", `{"e":"\ud800"}`, `{"e":"\ud800A"}`, "{\"v\":\"\xff\"}", `{"v":"\ud800"}`,
	`{"OK":true}`, `{"ok":1}`, `{"e":1}`, `{"ok":true,}`, `{"ok":true} x`, ` {"ok":true} `, ``, `{`,
}

// The result of an eval is read as encoding/json/v2 reads it: what v2 refuses
// (a name that occurs twice, text that is not UTF-8) is an error, not a value.
func TestParseEvalResultIsV2(t *testing.T) {
	for _, doc := range evalResultDocs {
		checkEvalResult(t, []byte(doc))
	}
}

func FuzzParseEvalResultIsV2(f *testing.F) {
	for _, doc := range evalResultDocs {
		f.Add([]byte(doc))
	}
	f.Fuzz(checkEvalResult)
}

func checkEvalResult(t *testing.T, doc []byte) {
	t.Helper()
	ok, v, msg, err := parseEvalResult(doc)
	var want evalResultV2
	werr := json.Unmarshal(doc, &want)
	if (err == nil) != (werr == nil) {
		t.Fatalf("%q: read with error %v; encoding/json/v2 reads with error %v", doc, err, werr)
	}
	if err != nil {
		return
	}
	// v2 keeps v as written, and a null v as null.
	if ok != want.OK || msg != want.E || !bytes.Equal(v, want.V) {
		t.Fatalf("%q: read %v %q %q; encoding/json/v2 reads %v %q %q", doc, ok, v, msg, want.OK, want.V, want.E)
	}
}

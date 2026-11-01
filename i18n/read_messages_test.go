package i18n

import (
	"encoding/json"
	"maps"
	"testing"
)

// readMessagesV1 is what the catalog's loader did with encoding/json before
// readMessages: the oracle.
func readMessagesV1(data []byte) (map[string]string, bool) {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return nil, false
	}
	out := map[string]string{}
	for key, v := range raw {
		var s string
		if json.Unmarshal(v, &s) != nil {
			continue
		}
		out[key] = s
	}
	return out, true
}

var readMessagesCases = []string{
	``, `null`, `{}`, ` {} `, `[]`, `1`, `"a"`, `{`, `{"a":"b"}`, `{"a":"b",}`,
	`{"a":"b","a":"c"}`, `{"a":"b","a":1}`, `{"a":1,"a":"b"}`, `{"a":null}`,
	`{"a":"b","a":null}`, `{"a":1}`, `{"a":true,"b":"x"}`, `{"a":{"b":"c"},"b":"x"}`,
	`{"a":["x"],"b":"y"}`, `{"@a":"b","c":"d"}`, `{"a":"é😀"}`,
	`{"a":"\ud800"}`, "{\"a\":\"\xff\"}", "{\"\xff\":\"x\"}", `{"a":"b"} x`,
	`{"a":"b"}{}`, `{"a":"\x"}`, `{"a\u0000":"b"}`, `{"A":"x","a":"y"}`, `{"a":-0.0e1,"b":"c"}`,
}

func TestReadMessagesIsV1(t *testing.T) {
	for _, c := range readMessagesCases {
		check(t, []byte(c))
	}
}

func FuzzReadMessagesIsV1(f *testing.F) {
	for _, c := range readMessagesCases {
		f.Add([]byte(c))
	}
	f.Fuzz(check1)
}

func check1(t *testing.T, data []byte) { check(t, data) }

func check(t *testing.T, data []byte) {
	t.Helper()
	got, gok := readMessages(data)
	want, wok := readMessagesV1(data)
	if gok != wok || gok && !maps.Equal(got, want) {
		t.Errorf("%q: read %q, %v; encoding/json reads %q, %v", data, got, gok, want, wok)
	}
}

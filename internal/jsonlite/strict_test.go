package jsonlite

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"testing"
)

func strictSkip(b []byte) error {
	r := NewReader(b)
	r.Strict = true
	if err := r.Skip(); err != nil {
		return err
	}
	return r.End()
}

func TestStrictAgreesWithV2(t *testing.T) {
	docs := append([]string{deep(10000), deep(10001)}, documents...)
	docs = append(docs, `{"a":1,"a":2}`, `{"a":{"b":1,"b":2}}`, `[{"a":1},{"a":2}]`, `{"a":1,"a":2}`, `"\ud83d"`, "\"\xff\"")
	for _, d := range docs {
		if got, want := strictSkip([]byte(d)) == nil, jsontext.Value(d).IsValid(); got != want {
			t.Errorf("%.40q: strict valid %v, encoding/json/v2 valid %v", d, got, want)
		}
	}
}

func FuzzStrictAgreesWithV2(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Add([]byte(`{"a":1,"a":2}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if got, want := strictSkip(b) == nil, jsontext.Value(b).IsValid(); got != want {
			t.Fatalf("%q: strict valid %v, encoding/json/v2 valid %v", b, got, want)
		}
	})
}

func FuzzStrictStringAgreesWithV2(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		r := NewReader(b)
		r.Strict = true
		got, err := r.String()
		if null, _ := NewReader(b).Null(); null {
			// null leaves the value as it is, in v1 and v2; the callers check for it.
			return
		}
		if err == nil {
			err = r.End()
		}
		var want string
		werr := jsonv2.Unmarshal(b, &want)
		if (err == nil) != (werr == nil) || err == nil && got != want {
			t.Fatalf("%q: strict %q, %v; encoding/json/v2 %q, %v", b, got, err, want, werr)
		}
	})
}

func FuzzStrictIntAgreesWithV2(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		r := NewReader(b)
		r.Strict = true
		got, err := r.Int()
		if null, _ := NewReader(b).Null(); null {
			// null leaves the value as it is, in v1 and v2; the callers check for it.
			return
		}
		if err == nil {
			err = r.End()
		}
		var want int
		werr := jsonv2.Unmarshal(b, &want)
		if (err == nil) != (werr == nil) || err == nil && got != want {
			t.Fatalf("%q: strict %d, %v; encoding/json/v2 %d, %v", b, got, err, want, werr)
		}
	})
}

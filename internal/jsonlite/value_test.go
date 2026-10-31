package jsonlite

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"testing"
)

func FuzzStrictInt64AgreesWithV2(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Add([]byte(`9223372036854775807`))
	f.Add([]byte(`-9223372036854775808`))
	f.Add([]byte(`9223372036854775808`))
	f.Fuzz(func(t *testing.T, b []byte) {
		if null, _ := NewReader(b).Null(); null {
			return
		}
		r := NewReader(b)
		r.Strict = true
		got, err := r.Int64()
		if err == nil {
			err = r.End()
		}
		var want int64
		werr := jsonv2.Unmarshal(b, &want)
		if (err == nil) != (werr == nil) || err == nil && got != want {
			t.Fatalf("%q: strict %d, %v; encoding/json/v2 %d, %v", b, got, err, want, werr)
		}
	})
}

func FuzzAppendQuote(f *testing.F) {
	for _, s := range []string{"", "a", "a\"b\\c", "\x00\x1f\x7f", "é😀", "  <>&", "\xff", "a\xe2\x80", "\xf0\x9f\x98", "\xed\xa0\x80"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		want, _ := jsontext.AppendQuote([]byte("x"), s)
		if got := AppendQuote([]byte("x"), s); string(got) != string(want) {
			t.Fatalf("%q: jsonlite %q, jsontext %q", s, got, want)
		}
	})
}

func TestValueIsTheBytesOfTheValue(t *testing.T) {
	r := NewReader([]byte(` [ {"a" : 1} , "x\n" , null ] `))
	r.Strict = true
	var got []string
	err := r.Array(func() error {
		v, err := r.Value()
		got = append(got, string(v))
		return err
	})
	if err != nil || r.End() != nil {
		t.Fatal(err)
	}
	want := []string{`{"a" : 1}`, `"x\n"`, `null`}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("got %q, want %q", got, want)
	}
}

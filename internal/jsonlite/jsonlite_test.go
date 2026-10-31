package jsonlite

import (
	"bytes"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"strings"
	"testing"
	"unicode/utf8"
)

// documents are seeds of the fuzzers: valid and broken, with the escapes, the
// numbers and the white space that tell the readers apart.
var documents = []string{
	``, ` `, `null`, `true`, `false`, `0`, `-0`, `1`, `-1`, `01`, `1.`, `.5`, `1e5`, `1E+5`, `1e`, `-`,
	`"a"`, `"é"`, `"😀"`, `"\ud83d"`, `"\ude00"`, `"\ud83dx"`, `"\ud83dA"`, `"\x"`, `"\u12"`,
	"\"a\x00b\"", "\"a\nb\"", "\"\xff\xfe\"", "\"\xe2\x80\xa8\"", `"\/"`, `"\b\f\n\r\t"`,
	`[]`, `[1,2,3]`, `[1,]`, `[,1]`, `[1 2]`, ` [ 1 , "a" , null ] `, `[[[]]]`, `[`, `]`,
	`{}`, `{"a":1}`, `{"a":1,}`, `{"a" 1}`, `{a:1}`, `{"a":1,"a":2}`, `{"x":1,"y":2,"width":3,"height":4}`,
	`{"A":{"x":1}}`, `{"w":{"x":1.5}}`, `{"w":{"x":1e2}}`, `{"w":{"x":null,"maximized":null}}`, `{"w":null}`,
	`{"w":{"x":9223372036854775807}}`, `{"w":{"x":9223372036854775808}}`, `{"w":{"x":-9223372036854775808}}`,
	`{"w":{"maximized":true,"fullScreen":false,"extra":[1,{"a":"b"}]}}`, `{"w":{"maximized":1}}`, `{"w":1}`,
	"\xef\xbb\xbf{}", "{}\n", "{} x", `{"a":"b"} {}`, "\t\r\n 1 \t\r\n",
	`{"@meta":{"a":[1,2,{"b":null}]},"k":"v","n":null,"x":1,"o":{},"a":[]}`,
}

// deep returns a document nested n levels.
func deep(n int) string { return strings.Repeat("[", n) + strings.Repeat("]", n) }

func skipAll(b []byte) error {
	r := NewReader(b)
	if err := r.Skip(); err != nil {
		return err
	}
	return r.End()
}

func TestSkipAgreesWithValid(t *testing.T) {
	docs := append([]string{deep(10000), deep(10001), deep(9999)}, documents...)
	for _, d := range docs {
		if got, want := skipAll([]byte(d)) == nil, json.Valid([]byte(d)); got != want {
			t.Errorf("%.40q: jsonlite valid %v, encoding/json valid %v", d, got, want)
		}
	}
}

func FuzzSkipAgreesWithValid(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if got, want := skipAll(b) == nil, json.Valid(b); got != want {
			t.Fatalf("%q: jsonlite valid %v, encoding/json valid %v", b, got, want)
		}
	})
}

func readString(b []byte) (string, error) {
	r := NewReader(b)
	s, err := r.String()
	if err != nil {
		return "", err
	}
	return s, r.End()
}

// A string is read as encoding/json reads it: the same text, or both refuse.
func FuzzStringAgreesWithUnmarshal(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		var want string
		werr := json.Unmarshal(b, &want)
		got, gerr := readString(b)
		switch {
		case werr != nil && gerr == nil:
			// null and a string differ only in what encoding/json does with null.
			t.Fatalf("%q: encoding/json refuses (%v), jsonlite reads %q", b, werr, got)
		case werr == nil && gerr != nil:
			if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
				return // null is not a string: the caller asks Null first
			}
			t.Fatalf("%q: encoding/json reads %q, jsonlite refuses (%v)", b, want, gerr)
		case werr == nil && got != want:
			t.Fatalf("%q: encoding/json reads %q, jsonlite %q", b, want, got)
		}
	})
}

func readInt(b []byte) (int, error) {
	r := NewReader(b)
	n, err := r.Int()
	if err != nil {
		return 0, err
	}
	return n, r.End()
}

func FuzzIntAgreesWithUnmarshal(f *testing.F) {
	for _, d := range documents {
		f.Add([]byte(d))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		var want int
		werr := json.Unmarshal(b, &want)
		got, gerr := readInt(b)
		if (werr == nil) != (gerr == nil) {
			if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
				return
			}
			t.Fatalf("%q: encoding/json error %v, jsonlite error %v", b, werr, gerr)
		}
		if werr == nil && got != want {
			t.Fatalf("%q: encoding/json reads %d, jsonlite %d", b, want, got)
		}
	})
}

var stringSamples = []string{
	``, `a`, `<tag> & "quoted" \ slash /`, "tab\tnl\ncr\rbs\bff\f", "\x00\x01\x1f\x7f", "é€😀", "  ", "\xff\xfe", "ab\xc3", "\xed\xa0\x80",
	"line sep", "<script>alert('x')</script>", strings.Repeat("x", 300),
}

// Strings are written as encoding/json's Marshal writes them, and read back.
func TestAppendStringIsMarshal(t *testing.T) {
	for _, s := range stringSamples {
		want, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		got := AppendString(nil, s)
		if !bytes.Equal(got, want) {
			t.Errorf("%q: jsonlite %s, encoding/json %s", s, got, want)
		}
		back, err := readString(got)
		if err != nil || back != string(coerceUTF8([]byte(s))) {
			t.Errorf("%q: read back %q, %v", s, back, err)
		}
	}
}

func FuzzAppendString(f *testing.F) {
	for _, s := range stringSamples {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		want, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if got := AppendString(nil, s); !bytes.Equal(got, want) {
			t.Fatalf("%q: jsonlite %s, encoding/json %s", s, got, want)
		}
	})
}

// The strict writer is encoding/json/v2's: valid strings the same bytes, others refused.
func FuzzAppendStringStrict(f *testing.F) {
	for _, s := range stringSamples {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		want, werr := jsonv2.Marshal(s)
		got, ok := AppendStringStrict(nil, s)
		if ok != (werr == nil) {
			t.Fatalf("%q: jsonlite ok %v, encoding/json/v2 error %v", s, ok, werr)
		}
		if ok && !bytes.Equal(got, want) {
			t.Fatalf("%q: jsonlite %s, encoding/json/v2 %s", s, got, want)
		}
		if ok && !utf8.Valid(got) {
			t.Fatalf("%q: not UTF-8: %s", s, got)
		}
	})
}

// A float64 is written as encoding/json writes it, or refused where it refuses.
func FuzzAppendFloat(f *testing.F) {
	for _, v := range []float64{0, 1, -1.5, 1e-6, 1e-7, 1e20, 1e21, 123456789.125, 5e-324} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, v float64) {
		want, err := json.Marshal(v)
		got, ok := AppendFloat(nil, v)
		if ok != (err == nil) {
			t.Fatalf("%v: jsonlite ok %v, encoding/json error %v", v, ok, err)
		}
		if ok && !bytes.Equal(got, want) {
			t.Fatalf("%v: jsonlite %s, encoding/json %s", v, got, want)
		}
	})
}

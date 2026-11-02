package cdp

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

// The Windows backend used encoding/json for the parameters and the replies of the
// DevTools protocol, which brought its reflection into every app. These tests hold
// the replacements to what encoding/json did, as the code they replaced was
// written: the oracles below are that code.

var hardStrings = []string{
	"", "a", `"`, `\`, "\x00", "\x1f", "\b\f\n\r\t", "<script>&</script>", "  ", "‧‪",
	"\xff", "a\xffb", "\xe2\x80", "\xf0\x9f\x98\x80", "😀 ünï", "퟿\U0010ffff", strings.Repeat("x\"\\\n", 300),
	"(async () => {\nreturn 1\n})()", "document.execCommand(\"bold\")", "\u007f", "\u0085",
}

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("not JSON: %v\n%q", err, s)
	}
	return v
}

func marshal(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return decode(t, string(b))
}

func TestParamsDecodeAsEncodingJSONMarshalled(t *testing.T) {
	for _, s := range hardStrings {
		if got, want := decode(t, InsertText(s)), marshal(t, map[string]string{"text": s}); !reflect.DeepEqual(got, want) {
			t.Errorf("InsertText(%q): %v, want %v", s, got, want)
		}
		for _, g := range []bool{false, true} {
			if got, want := decode(t, Evaluate(s, g)), marshal(t, map[string]any{"expression": s, "userGesture": g}); !reflect.DeepEqual(got, want) {
				t.Errorf("Evaluate(%q, %v): %v, want %v", s, g, got, want)
			}
		}
		if got, want := decode(t, EvaluateAsync(s)), marshal(t, map[string]any{"expression": s, "awaitPromise": true, "returnByValue": true}); !reflect.DeepEqual(got, want) {
			t.Errorf("EvaluateAsync(%q): %v, want %v", s, got, want)
		}
	}
}

func TestPrintToPDFParams(t *testing.T) {
	for _, n := range []float64{0, 1, -1, 0.5, 8.5, 11, 0.4, 1e-7, 1e21, 123456789.125, math.SmallestNonzeroFloat64, math.MaxFloat64} {
		for _, b := range []bool{false, true} {
			got, err := PrintToPDF(b, !b, n, n+1, n+2, n+3, n+4, n+5)
			if err != nil {
				t.Fatal(err)
			}
			want := marshal(t, map[string]any{
				"landscape": b, "printBackground": !b, "paperWidth": n, "paperHeight": n + 1, "marginTop": n + 2,
				"marginRight": n + 3, "marginBottom": n + 4, "marginLeft": n + 5, "preferCSSPageSize": false,
			})
			if g := decode(t, got); !reflect.DeepEqual(g, want) {
				t.Errorf("PrintToPDF(%v): %v, want %v", n, g, want)
			}
		}
	}
	// A number JSON has no text for was an error that the caller ignored, and the
	// request went with no parameters; it is an error now.
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := PrintToPDF(false, false, bad, 1, 1, 1, 1, 1); err == nil {
			t.Errorf("PrintToPDF(%v) has no error", bad)
		}
	}
}

// oldValue is the code of CallAsyncFunction that read the reply of Runtime.evaluate.
func oldValue(result string) (string, error) {
	var out struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception *struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		return "", err
	}
	if d := out.ExceptionDetails; d != nil {
		msg := d.Text
		if d.Exception != nil && d.Exception.Description != "" {
			msg = d.Exception.Description
		}
		return "", errors.New(msg)
	}
	var s string
	_ = json.Unmarshal(out.Result.Value, &s)
	return s, nil
}

// oldData is the code of PrintToPDF that read the reply of Page.printToPDF.
func oldData(res string) (string, error) {
	var r struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		return "", err
	}
	return r.Data, nil
}

// oldTitle is the code of the title handler.
func oldTitle(result string) (string, error) {
	var title string
	err := json.Unmarshal([]byte(result), &title)
	return title, err
}

type reader struct {
	name     string
	old, new func(string) (string, error)
}

var readers = []reader{
	{"Value", oldValue, Value},
	{"Data", oldData, Data},
	{"Title", oldTitle, Title},
}

// same fails if the old code and the new one differ in the value, in whether there
// is an error, and, for an exception, in the message.
func same(t *testing.T, r reader, in string) {
	t.Helper()
	ov, oe := r.old(in)
	nv, ne := r.new(in)
	if (oe == nil) != (ne == nil) {
		t.Errorf("%s(%q): error %v, was %v", r.name, in, ne, oe)
		return
	}
	if oe != nil {
		// The message of an exception is the page's own; that of a broken document
		// is the package's.
		var syn *json.SyntaxError
		var typ *json.UnmarshalTypeError
		if !errors.As(oe, &syn) && !errors.As(oe, &typ) && oe.Error() != ne.Error() {
			t.Errorf("%s(%q): message %q, was %q", r.name, in, ne, oe)
		}
		return
	}
	if nv != ov {
		t.Errorf("%s(%q): %q, was %q", r.name, in, nv, ov)
	}
}

var replies = []string{
	``, ` `, `null`, `true`, `1`, `"s"`, `[]`, `{}`, `{"x":1}`, `{`, `{"result":`, `{"result":{}}`, `{"result":{"value":`,
	`{"result":{"type":"string","value":"hello"}}`,
	`{"result":{"value":""}}`,
	`{"result":{"value":null}}`,
	`{"result":{"value":1}}`,
	`{"result":{"value":12345678901234567890}}`,
	`{"result":{"value":true}}`,
	`{"result":{"value":{"a":["b",{"c":null}]}}}`,
	`{"result":{"value":["x"]}}`,
	`{"result":{"value":"é😀\ud83d \ude00 \n\t\"\\\/"}}`,
	`{"result":{"value":"a b\u0000c"}}`,
	`{"result":{"value":"` + "\xff\xfe" + `"}}`,
	`{"result":null}`,
	`{"result":{"value":"v"},"result":{"value":"last"}}`,
	`{"result":{"value":"a","value":"b"}}`,
	`{"result":{"value":"x"},"other":[1,2,{"k":"v"}],"id":7}`,
	`  {"result" : { "value" : "sp" } }  `,
	`{"result":{"value":"x"}} trailing`,
	`{"result":{"value":"x"}}{}`,
	`{"exceptionDetails":null,"result":{"value":"ok"}}`,
	`{"exceptionDetails":{"text":"Uncaught"},"result":{}}`,
	`{"exceptionDetails":{"text":"Uncaught","exception":{"description":"Error: boom\n    at <anonymous>:1:1"}}}`,
	`{"exceptionDetails":{"text":"Uncaught","exception":{"description":""}}}`,
	`{"exceptionDetails":{"text":"Uncaught","exception":null}}`,
	`{"exceptionDetails":{"exception":{"description":"d"}}}`,
	`{"exceptionDetails":{}}`,
	`{"exceptionDetails":{"text":1}}`,
	`{"exceptionDetails":[]}`,
	`{"exceptionDetails":{"text":"\ud83d"}}`,
	`{"data":"QUJD"}`, `{"data":""}`, `{"data":null}`, `{"data":1}`, `{"data":"a","data":"b"}`, `{"data":"A"}`,
	`""`, `"a title"`, `"é😀"`, `"a\"b\\c\/d"`, `"unterminated`, `"bad\q"`, `"\ud800"`, `"x" y`,
	`"` + "\xff" + `"`, `"` + "\n" + `"`,
}

func TestReadersMatchEncodingJSON(t *testing.T) {
	for _, r := range readers {
		for _, in := range replies {
			same(t, r, in)
		}
	}
}

// A short walk with a fixed seed over documents made of the parts of the replies
// above: cut, joined and flipped.
func TestReadersMatchEncodingJSONOnMutations(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	const cuts = `{}[]":,\ntrufals0-1.e+ `
	for range 4000 {
		in := []byte(replies[rng.IntN(len(replies))])
		for range rng.IntN(4) {
			if len(in) == 0 {
				break
			}
			switch i := rng.IntN(len(in)); rng.IntN(4) {
			case 0:
				in = in[:i]
			case 1:
				in = append(in[:i:i], in[i+1:]...)
			case 2:
				in[i] = cuts[rng.IntN(len(cuts))]
			case 3:
				in = append(in[:i:i], append([]byte(replies[rng.IntN(len(replies))]), in[i:]...)...)
			}
		}
		for _, r := range readers {
			same(t, r, string(in))
		}
	}
}

func FuzzReaders(f *testing.F) {
	for _, in := range replies {
		f.Add(in)
	}
	f.Fuzz(func(t *testing.T, in string) {
		for _, r := range readers {
			same(t, r, in)
		}
	})
}

func TestStringsAndJSStringAreEncodingJSON(t *testing.T) {
	for _, s := range hardStrings {
		b, _ := json.Marshal(s)
		if got := JSString(s); got != string(b) {
			t.Errorf("JSString(%q) = %s, json.Marshal %s", s, got, b)
		}
	}
	for _, l := range [][]string{nil, {}, {"app"}, hardStrings} {
		b, _ := json.Marshal(l)
		if got := Strings(l); got != string(b) {
			t.Errorf("Strings(%q) = %s, json.Marshal %s", l, got, b)
		}
	}
}

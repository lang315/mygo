package bridge

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"math"
	"testing"
)

var floats = []float64{0, -0.0, 1, -1, 1.5, 0.1, 100, 1e-6, 1e-7, 1.5e-9, 123456789.125, 1e20, 1e21, 1.5e21, 2.5e-300, math.MaxFloat64, math.SmallestNonzeroFloat64}

var strs = []string{"", "en", "zh-Hans", "ltr", "rtl", "9.0.1", "a\"b\\c", "<script>&</script>", "tab\tnl\n  é😀", "\xff", "x\x00y"}

// The JSON written here is the JSON encoding/json writes for the same types: the
// scripts and the events of every window say it.
func TestJSONIsMarshal(t *testing.T) {
	same := func(what string, got []byte, v any) {
		t.Helper()
		want, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: written\n%s\nencoding/json writes\n%s", what, got, want)
		}
	}
	for _, s := range strs {
		cfg := Config{Platform: s, WindowID: -7, Version: s, Secret: s, Locale: s, Dir: s}
		same("Config", cfg.JSON(), cfg)
	}
	for _, f := range floats {
		same("TitleBar", TitleBar{Height: f, Left: -f, Right: f / 3}.JSON(), TitleBar{Height: f, Left: -f, Right: f / 3})
		fit := FitContent{MinWidth: f, MinHeight: -f, MaxWidth: f / 2, MaxHeight: 0}
		same("FitContent", fit.AppendJSON(nil), fit)
		cfg := Config{Platform: "darwin", WindowID: 3, Version: "1", Secret: "s", Locale: "en", Dir: "ltr", Fit: &fit}
		same("Config with fit", cfg.JSON(), cfg)
	}
}

// The payload of the locale event is what the IPC wrote with encoding/json/v2,
// with its options: an event that v2 cannot write is not sent.
func TestLocaleJSONIsV2(t *testing.T) {
	opts := jsontext.EscapeForJS(true)
	for _, s := range append(strs, "a b ", "<&>") {
		for _, l := range []Locale{{Locale: s, Dir: "rtl"}, {Locale: "en", Dir: s}} {
			want, werr := jsonv2.Marshal(l, opts)
			got, ok := l.JSON()
			if ok != (werr == nil) || ok && !bytes.Equal(got, want) {
				t.Errorf("%q: written %s, %v; encoding/json/v2 writes %s, %v", s, got, ok, want, werr)
			}
		}
	}
}

// A number JSON has no text for is written as 0, not as nothing.
func TestJSONOfANumberThatIsNone(t *testing.T) {
	got := string(TitleBar{Height: math.NaN(), Left: math.Inf(1), Right: 1}.JSON())
	if got != `{"height":0,"left":0,"right":1}` {
		t.Errorf("written %s", got)
	}
}

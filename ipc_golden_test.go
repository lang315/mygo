//go:build !js

package mygo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdflag "flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The golden recordings of the page IPC: what the Go side says to a page for
// each message the page can post, recorded from the encoding/json (v2) code
// before the IPC stopped depending on it. They pin the old behavior: byte for
// byte the replies of calls (every parameter and result type, errors, panics,
// malformed envelopes, calls of pages that may not call), what the log says of
// a call that is dropped, the windows' own methods (mygo:window), and the effect
// of the messages that are not calls. Re-record with -update-golden only to
// change the IPC on purpose.
var updateGolden = stdflag.Bool("update-golden", false, "rewrite testdata/ipc_golden.json")

type goldenPoint struct {
	X, Y int
	Tag  string `json:"tag,omitempty"`
	Skip string `json:"-"`
}

type goldenService struct{}

func (goldenService) Str(s string) string                        { return s }
func (goldenService) Int(n int) int                              { return n }
func (goldenService) I8(n int8) int8                             { return n }
func (goldenService) U64(n uint64) uint64                        { return n }
func (goldenService) F64(f float64) float64                      { return f }
func (goldenService) Bool(b bool) bool                           { return b }
func (goldenService) Bytes(b []byte) []byte                      { return b }
func (goldenService) Strs(s []string) []string                   { return s }
func (goldenService) Map(m map[string]int) map[string]int        { return m }
func (goldenService) Any(v any) any                              { return v }
func (goldenService) Point(p goldenPoint) goldenPoint            { return p }
func (goldenService) PointPtr(p *goldenPoint) *goldenPoint       { return p }
func (goldenService) Dur(d time.Duration) time.Duration          { return d }
func (goldenService) Time(t time.Time) time.Time                 { return t }
func (goldenService) Optional(a string, b *int) string           { return fmt.Sprintf("%s:%v", a, b == nil) }
func (goldenService) Variadic(prefix string, rest ...int) string { return fmt.Sprint(prefix, rest) }
func (goldenService) Ctx(ctx context.Context, n int) int         { return CallerWindow(ctx).ID() + n }
func (goldenService) Fail(fail bool) (string, error) {
	if fail {
		return "", errors.New("failed <&> \u2028 \"quoted\"")
	}
	return "fine", nil
}
func (goldenService) Panic()                 { panic("boom") }
func (goldenService) Nothing()               {}
func (goldenService) Nil() []string          { return nil }
func (goldenService) NilMap() map[string]int { return nil }
func (goldenService) HTML() string           { return "<a href=\"x\">&'\u2028\u2029 é😀\x00\x1f</a>" }
func (goldenService) Invalid() string        { return "bad\xffutf8" }
func (goldenService) NaN() float64           { return nan() }
func (goldenService) Big() int64             { return 1 << 62 }
func (goldenService) Float() float64         { return 1e21 }
func (goldenService) Small() float64         { return 1e-7 }
func (goldenService) Nested() any {
	// A struct, not a map: encoding/json/v2 writes the members of a map in the
	// order of the map, which varies from run to run.
	return struct {
		A float64 `json:"a"`
		B []any   `json:"b"`
	}{1.5, []any{1, "two", nil, true, map[string]any{}}}
}
func (goldenService) Duration() time.Duration { return 1500 * time.Millisecond }

func nan() float64 {
	var z float64
	return z / z
}

type goldenRecord struct {
	Replies []string `json:"replies,omitempty"`
	Log     string   `json:"log,omitempty"`
	Note    string   `json:"note,omitempty"`
}

// goldenCases are the messages of calls: the name of the case, and the message
// the page posts after the secret. Each is run on a window that is trusted.
func goldenCases() [][2]string {
	var cases [][2]string
	add := func(name, msg string) { cases = append(cases, [2]string{name, msg}) }
	call := func(name, method, args string) {
		add(name, fmt.Sprintf(`{"t":"call","id":1,"k":"tok","m":%q,"a":%s}`, method, args))
	}
	// Parameters.
	call("str", "G.Str", `["hello"]`)
	call("str escapes", "G.Str", `["a\"b\\c\n\t\u00e9\ud83d\ude00\u2028<&>"]`)
	call("str lone surrogate", "G.Str", `["\ud800x"]`)
	call("str number", "G.Str", `[1]`)
	call("str null", "G.Str", `[null]`)
	call("str missing", "G.Str", `[]`)
	call("str extra", "G.Str", `["a","b"]`)
	call("int", "G.Int", `[42]`)
	call("int negative", "G.Int", `[-7]`)
	call("int float", "G.Int", `[1.5]`)
	call("int exponent", "G.Int", `[1e2]`)
	call("int string", "G.Int", `["5"]`)
	call("int huge", "G.Int", `[9223372036854775808]`)
	call("int8 overflow", "G.I8", `[200]`)
	call("uint64 max", "G.U64", `[18446744073709551615]`)
	call("uint64 negative", "G.U64", `[-1]`)
	call("float", "G.F64", `[1.25]`)
	call("float exponent", "G.F64", `[1e21]`)
	call("float tiny", "G.F64", `[1e-7]`)
	call("float negative zero", "G.F64", `[-0]`)
	call("bool", "G.Bool", `[true]`)
	call("bool number", "G.Bool", `[1]`)
	call("bytes", "G.Bytes", `["aGVsbG8="]`)
	call("bytes bad", "G.Bytes", `["!!"]`)
	call("bytes array", "G.Bytes", `[[1,2,3]]`)
	call("strings", "G.Strs", `[["a","b"]]`)
	call("strings null element", "G.Strs", `[["a",null]]`)
	call("strings null", "G.Strs", `[null]`)
	call("map", "G.Map", `[{"a":1}]`)
	call("map duplicate key", "G.Map", `[{"a":1,"a":2}]`)
	call("map bad value", "G.Map", `[{"a":"x"}]`)
	call("any object", "G.Any", `[{"z":[1,2,{"q":null}]}]`)
	call("any number", "G.Any", `[12345678901234567890]`)
	call("any null", "G.Any", `[null]`)
	call("point", "G.Point", `[{"X":1,"Y":2,"tag":"t"}]`)
	call("point lowercase", "G.Point", `[{"x":1,"y":2}]`)
	call("point unknown field", "G.Point", `[{"X":1,"extra":true}]`)
	call("point skip field", "G.Point", `[{"X":1,"Skip":"s"}]`)
	call("point duplicate field", "G.Point", `[{"X":1,"X":2}]`)
	call("point pointer", "G.PointPtr", `[{"X":3}]`)
	call("point pointer null", "G.PointPtr", `[null]`)
	call("duration", "G.Dur", `[1500000000]`)
	call("duration string", "G.Dur", `["1.5s"]`)
	call("time", "G.Time", `["2026-10-31T12:00:00Z"]`)
	call("time bad", "G.Time", `["yesterday"]`)
	call("optional", "G.Optional", `["a"]`)
	call("optional set", "G.Optional", `["a",5]`)
	call("optional null", "G.Optional", `["a",null]`)
	call("variadic", "G.Variadic", `["p",1,2,3]`)
	call("variadic none", "G.Variadic", `["p"]`)
	call("variadic bad", "G.Variadic", `["p",1,"x"]`)
	call("context", "G.Ctx", `[3]`)
	// Results and errors.
	call("fail", "G.Fail", `[true]`)
	call("fail not", "G.Fail", `[false]`)
	call("panic", "G.Panic", `[]`)
	call("nothing", "G.Nothing", `[]`)
	call("nil slice", "G.Nil", `[]`)
	call("nil map", "G.NilMap", `[]`)
	call("html result", "G.HTML", `[]`)
	call("invalid utf8 result", "G.Invalid", `[]`)
	call("nan result", "G.NaN", `[]`)
	call("big result", "G.Big", `[]`)
	call("float result", "G.Float", `[]`)
	call("small result", "G.Small", `[]`)
	call("nested result", "G.Nested", `[]`)
	call("duration result", "G.Duration", `[]`)
	call("missing method", "G.Missing", `[]`)
	call("missing service", "Nope.Str", `["x"]`)
	call("empty method", "", `[]`)
	call("method with html", "G.<&>", `[]`)
	// The envelope.
	add("envelope reordered", `{"t":"call","m":"G.Str","a":["x"],"id":2,"k":"tok"}`)
	add("envelope unknown field", `{"t":"call","id":3,"z":[1,{"a":null}],"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope duplicate id", `{"t":"call","id":3,"id":4,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope duplicate m", `{"t":"call","id":3,"k":"tok","m":"G.Str","m":"G.Int","a":["x"]}`)
	add("envelope uppercase names", `{"t":"call","ID":3,"K":"tok","M":"G.Str","A":["x"]}`)
	add("envelope id string", `{"t":"call","id":"5","k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id float", `{"t":"call","id":1.5,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id exponent", `{"t":"call","id":1e2,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id negative", `{"t":"call","id":-1,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id huge", `{"t":"call","id":9223372036854775807,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id too huge", `{"t":"call","id":9223372036854775808,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id missing", `{"t":"call","k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope id null", `{"t":"call","id":null,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope k number", `{"t":"call","id":1,"k":5,"m":"G.Str","a":["x"]}`)
	add("envelope k escapes", `{"t":"call","id":1,"k":"t\"o\\k\u2028<&>é","m":"G.Str","a":["x"]}`)
	add("envelope k invalid utf8", "{\"t\":\"call\",\"id\":1,\"k\":\"t\xffk\",\"m\":\"G.Str\",\"a\":[\"x\"]}")
	add("envelope k missing", `{"t":"call","id":1,"m":"G.Str","a":["x"]}`)
	add("envelope m missing", `{"t":"call","id":1,"k":"tok","a":["x"]}`)
	add("envelope m number", `{"t":"call","id":1,"k":"tok","m":5,"a":["x"]}`)
	add("envelope a missing", `{"t":"call","id":1,"k":"tok","m":"G.Str"}`)
	add("envelope a null", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":null}`)
	add("envelope a object", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":{"x":1}}`)
	add("envelope a string", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":"x"}`)
	add("envelope a whitespace", "{\"t\":\"call\",\"id\":1,\"k\":\"tok\",\"m\":\"G.Str\",\"a\":[ \"x\" \n]}")
	add("envelope spaces around", `{"t":"call", "id" : 1 , "k":"tok","m":"G.Str","a":["x"] }`)
	add("envelope trailing garbage", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["x"]} x`)
	add("envelope trailing object", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["x"]}{}`)
	add("envelope trailing space", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["x"]}  `)
	add("envelope truncated", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["x"`)
	add("envelope comma", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["x"],}`)
	add("envelope bare key", `{"t":"call",id:1,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope single quotes", `{"t":"call","id":1,"k":'tok',"m":"G.Str","a":["x"]}`)
	add("envelope nested arg", `{"t":"call","id":1,"k":"tok","m":"G.Any","a":[[[[[[[[{"a":[1,2,[3]]}]]]]]]]]}`)
	add("envelope depth", `{"t":"call","id":1,"k":"tok","m":"G.Any","a":[`+strings.Repeat("[", 1000)+strings.Repeat("]", 1000)+`]}`)
	add("envelope lone surrogate m", `{"t":"call","id":1,"k":"tok","m":"G.\ud800","a":[]}`)
	add("envelope invalid utf8 arg", "{\"t\":\"call\",\"id\":1,\"k\":\"tok\",\"m\":\"G.Str\",\"a\":[\"a\xffb\"]}")
	add("envelope control char", "{\"t\":\"call\",\"id\":1,\"k\":\"tok\",\"m\":\"G.Str\",\"a\":[\"a\nb\"]}")
	add("envelope number leading zero", `{"t":"call","id":01,"k":"tok","m":"G.Str","a":["x"]}`)
	add("envelope big payload", `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["`+strings.Repeat("x", 5000)+`"]}`)
	return cases
}

// bareWindow is a window with no native window and a page context: what
// handleCall needs to run a call and queue the reply.
func bareWindow(untrusted bool) *Window {
	w := &Window{id: 7, untrusted: untrusted}
	w.pageCtx = context.WithValue(context.Background(), callerKey{}, w)
	return w
}

// goldenLog keeps what the log says of a call that does not depend on the
// version of Go or of the machine: that a call was dropped, or a method
// panicked, without the text of the decoder's error or the stack.
func goldenLog(s string) string {
	if i := strings.Index(s, "\ngoroutine "); i >= 0 {
		s = s[:i+1]
	}
	if head, _, ok := strings.Cut(s, "malformed call from window "); ok {
		return head + "malformed call from window N: <error>\n"
	}
	return s
}

// decoderText matches the text of the decoder's or the encoder's own error in a
// reply, which varies with the version of Go ("cannot unmarshal" became "unable
// to unmarshal"): the golden pins the words of mygo, the index of the argument,
// and that the call failed, not the words of encoding/json.
var decoderText = regexp.MustCompile(`("e":"(?:invalid argument \d+|cannot encode result): )(?:[^"\\]|\\.)*"`)

func goldenReply(s string) string {
	return decoderText.ReplaceAllString(s, `${1}<decoder error>"`)
}

func runGoldenCall(w *Window, trusted bool, msg string) goldenRecord {
	var logs bytes.Buffer
	flags := log.Flags()
	log.SetFlags(0)
	log.SetOutput(&logs)
	defer func() { log.SetOutput(os.Stderr); log.SetFlags(flags) }()
	w.outMu.Lock()
	w.outbox, w.flushing = nil, true // nothing is sent: the replies are read from the outbox
	w.outMu.Unlock()
	handleCall(w, w.pageContext(), msg, trusted)
	w.outMu.Lock()
	msgs := w.outbox
	w.outbox, w.flushing = nil, false
	w.outMu.Unlock()
	rec := goldenRecord{Log: goldenLog(logs.String())}
	for _, m := range msgs {
		rec.Replies = append(rec.Replies, goldenReply(string(m.appendTo(nil))))
	}
	return rec
}

func TestIPCGolden(t *testing.T) {
	bindForTest(t, "G", goldenService{})
	got := map[string]goldenRecord{}

	// The calls of a page that may call, on a window that has no native window.
	trusted := bareWindow(false)
	for _, c := range goldenCases() {
		got["call/"+c[0]] = runGoldenCall(trusted, true, c[1])
	}
	// Pages that may not.
	untrustedWin := bareWindow(true)
	got["untrusted/call"] = runGoldenCall(untrustedWin, false, `{"t":"call","id":1,"k":"tok","m":"G.Str","a":["x"]}`)
	got["foreign origin/call"] = runGoldenCall(bareWindow(false), false, `{"t":"call","id":2,"k":"tok","m":"G.Str","a":["x"]}`)
	got["foreign origin/malformed"] = runGoldenCall(bareWindow(false), false, `{"t":"call","id":2,`)

	// The windows' own methods and the messages that are no calls, on a window.
	w, fw := readyWindow(t, WindowOptions{})
	builtin := func(name, method, args string) {
		rec := runGoldenCall(w, true, fmt.Sprintf(`{"t":"call","id":1,"k":"tok","m":%q,"a":%s}`, method, args))
		rec.Note = fmt.Sprintf("title %q maximized %v fullscreen %v", w.Title(), w.IsMaximized(), w.IsFullScreen())
		got["window/"+name] = rec
	}
	builtin("is maximized", "mygo:window.IsMaximized", `[]`)
	builtin("maximize", "mygo:window.Maximize", `[]`)
	builtin("is maximized after", "mygo:window.IsMaximized", `[]`)
	builtin("unmaximize", "mygo:window.Unmaximize", `[]`)
	builtin("toggle maximize", "mygo:window.ToggleMaximize", `[]`)
	builtin("toggle maximize back", "mygo:window.ToggleMaximize", `[]`)
	builtin("toggle full screen", "mygo:window.ToggleFullScreen", `[]`)
	builtin("toggle full screen back", "mygo:window.ToggleFullScreen", `[]`)
	builtin("set title", "mygo:window.SetTitle", `["Set <by> page"]`)
	builtin("set title number", "mygo:window.SetTitle", `[5]`)
	builtin("set title missing", "mygo:window.SetTitle", `[]`)
	builtin("set title extra", "mygo:window.SetTitle", `["two","args"]`)
	builtin("unknown", "mygo:window.Fly", `[]`)

	state := func(name, msg string) {
		page(fw, msg)
		got["message/"+name] = goldenRecord{Note: fmt.Sprintf("domReady %v themeColor %v %v title %q", w.domReady, w.hasThemeColor, w.themeColor, w.Title())}
	}
	state("theme color", `{"t":"theme-color","v":"#ff8000"}`)
	state("theme color named", `{"t":"theme-color","v":"red"}`)
	state("theme color bad", `{"t":"theme-color","v":"nonsense"}`)
	state("theme color empty", `{"t":"theme-color","v":""}`)
	state("theme color number", `{"t":"theme-color","v":5}`)
	state("theme color duplicate", `{"t":"theme-color","v":"#00ff00","v":"#0000ff"}`)
	state("theme color uppercase key", `{"t":"theme-color","V":"#123456"}`)
	state("unknown type", `{"t":"nothing"}`)
	state("no type", `{"v":"#112233"}`)
	state("type number", `{"t":5}`)
	state("not an object", `[1,2]`)
	state("not json", `dom-ready`)
	state("empty object", `{}`)
	state("trailing garbage", `{"t":"theme-color","v":"#010203"} x`)
	state("field of another type", `{"t":"theme-color","v":"#040506","x":"a"}`)
	state("unknown field", `{"t":"theme-color","v":"#070809","zz":[1,{"a":null}]}`)
	state("invalid utf8", "{\"t\":\"theme-color\",\"v\":\"\xff\"}")
	state("lone surrogate", `{"t":"theme-color","v":"\ud800"}`)
	state("dom-ready", `{"t":"dom-ready"}`)
	state("drag untrusted check", `{"t":"drag"}`)
	state("escape", `{"t":"escape"}`)
	state("context menu", `{"t":"context-menu","l":"https://x","lt":"link","i":"img","m":"media","s":"selection","e":true,"x":1.5,"y":2}`)
	state("chan-ack unknown", `{"t":"chan-ack","c":99,"k":"nope","n":1}`)
	state("content-size", `{"t":"content-size","w":100,"h":50,"sw":0,"sh":0}`)
	state("content-size strings", `{"t":"content-size","w":"x"}`)
	state("wrong secret", "wrong"+`{"t":"dom-ready"}`)

	path := filepath.Join("testdata", "ipc_golden.json")
	data, err := json.MarshalIndent(got, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no recording (go test -run TestIPCGolden -update-golden): %v", err)
	}
	if bytes.Equal(data, want) {
		return
	}
	var wantMap map[string]goldenRecord
	if err := json.Unmarshal(want, &wantMap); err != nil {
		t.Fatal(err)
	}
	for name, rec := range got {
		if w, ok := wantMap[name]; !ok {
			t.Errorf("%s: not recorded: %+v", name, rec)
		} else if fmt.Sprintf("%+v", w) != fmt.Sprintf("%+v", rec) {
			t.Errorf("%s:\n got  %+v\n want %+v", name, rec, w)
		}
	}
	for name := range wantMap {
		if _, ok := got[name]; !ok {
			t.Errorf("%s: recorded, and not run", name)
		}
	}
}

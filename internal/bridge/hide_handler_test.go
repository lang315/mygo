package bridge

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestHideHandlerScript runs the script in a JavaScript engine against windows
// as the engines make them: it takes the handler of MyGo away from a subframe
// and leaves the top frame, and the other things on the same objects, alone.
func TestHideHandlerScript(t *testing.T) {
	engine, err := exec.LookPath("node")
	if err != nil {
		if engine, err = exec.LookPath("bun"); err != nil {
			t.Skip("no node or bun to run JavaScript with")
		}
	}
	src, _ := json.Marshal(HideHandlerScript)
	driver := `
const vm = require("node:vm");
const script = ` + string(src) + `;
function run(make, top) {
  const win = make();
  win.window = win;
  win.top = top ? win : {};
  vm.createContext(win);
  vm.runInContext(script, win);
  return win;
}
const handler = () => ({ postMessage() {} });
const out = {};

// A subframe of WebKit: the handler goes, others stay.
let w = run(() => ({ webkit: { messageHandlers: { mygo: handler(), other: handler() } } }), false);
out.webkitSubframe = ["mygo" in w.webkit.messageHandlers && w.webkit.messageHandlers.mygo !== undefined, !!w.webkit.messageHandlers.other];

// The handler is on the prototype, which delete cannot reach: the frame loses webkit.
const proto = { mygo: handler() };
w = run(() => ({ webkit: { messageHandlers: Object.create(proto) } }), false);
out.webkitPrototype = [w.webkit === undefined || !w.webkit.messageHandlers || !w.webkit.messageHandlers.mygo];

// The top frame keeps its handler.
w = run(() => ({ webkit: { messageHandlers: { mygo: handler() } }, chrome: { webview: handler() } }), true);
out.top = [!!w.webkit.messageHandlers.mygo, !!w.chrome.webview];

// A subframe of WebView2: webview goes, the rest of chrome stays.
w = run(() => ({ chrome: { webview: handler(), runtime: {} } }), false);
out.webview = [w.chrome.webview === undefined, !!w.chrome.runtime];

// Neither object: nothing throws.
w = run(() => ({}), false);
out.none = [true];

// A frame whose top cannot be read still gets the handler taken away or leaves.
w = run(() => ({ webkit: { messageHandlers: { mygo: handler() } } }), false);
out.again = [!(w.webkit && w.webkit.messageHandlers && w.webkit.messageHandlers.mygo)];

console.log(JSON.stringify(out));
`
	out, err := exec.Command(engine, "-e", driver).CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", engine, err, out)
	}
	var got map[string][]bool
	line := strings.TrimSpace(string(out))
	if err := json.Unmarshal([]byte(line[strings.LastIndex(line, "\n")+1:]), &got); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	want := map[string][]bool{
		"webkitSubframe":  {false, true},
		"webkitPrototype": {true},
		"top":             {true, true},
		"webview":         {true, true},
		"none":            {true},
		"again":           {true},
	}
	for k, w := range want {
		if g := got[k]; len(g) != len(w) || (len(w) > 0 && !equalBools(g, w)) {
			t.Errorf("%s: %v, want %v", k, g, w)
		}
	}
}

func equalBools(a, b []bool) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

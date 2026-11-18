//go:build !js

package mygo

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/wait"
)

// What a page asked for, and what the app tells its pages, goes to a page
// that is still the window's and that the app trusts; a call the bridge
// cannot read, or whose result cannot be encoded, is still answered.

type auditService struct{}

type panicky struct{}

func (panicky) MarshalJSON() ([]byte, error) { panic("encoding failed") }

var (
	auditGate    = make(chan struct{})
	auditStarted = make(chan struct{}, 4)
	auditOnce    sync.Once
	auditEvent   = NewEvent[string]("audit-secret")
	panickyEvent = NewEvent[panicky]("audit-panicky")
)

func (auditService) Echo(s string) string { return s }
func (auditService) Slow() string {
	gate := auditGate
	auditStarted <- struct{}{}
	<-gate
	return "reply of the first page"
}
func (auditService) Panicky() panicky { return panicky{} }
func (auditService) Proto() map[string]any {
	return map[string]any{"__proto__": map[string]any{"a": 1}}
}

func auditWindow(t *testing.T) (*Window, *fake.Window) {
	t.Helper()
	auditOnce.Do(func() { BindAs("AuditTest", auditService{}) })
	return readyWindow(t, WindowOptions{})
}

// scriptsWith returns the scripts evaluated in fw that contain s.
func scriptsWith(fw *fake.Window, s string) []string {
	var out []string
	for _, js := range fw.Scripts() {
		if strings.Contains(js, s) {
			out = append(out, js)
		}
	}
	return out
}

func TestReplyOfALeftPageIsDropped(t *testing.T) {
	_, fw := auditWindow(t)
	gate := auditGate
	auditGate = make(chan struct{})
	defer func() { auditGate = gate }()
	release := auditGate
	page(fw, `{"t":"call","id":1,"k":"tok","m":"AuditTest.Slow","a":[]}`)
	<-auditStarted
	commit(fw, "https://evil.example/")
	before := len(fw.Scripts())
	close(release)
	// A call of the new document, answered after the first: once its reply
	// is in, the first one has been sent or dropped.
	s := newTestSecret()
	hello(fw, s)
	onMain(func() { fw.H.Message(s + `{"t":"dom-ready"}`) })
	onMain(func() {
		fw.H.Message(s + `{"t":"call","id":2,"k":"tok2","m":"AuditTest.Echo","a":["second"]}`)
	})
	wait.Until(t, "the reply of the new document", func() bool { return len(scriptsWith(fw, `"tok2"`)) > 0 })
	for _, js := range fw.Scripts()[before:] {
		if strings.Contains(js, "reply of the first page") {
			t.Fatalf("a script evaluated in the new document carries the first page's reply: %.200s", js)
		}
	}
}

func TestEncodeResultPanicBecomesAnError(t *testing.T) {
	_, fw := auditWindow(t)
	reply := call(t, fw, 1, "AuditTest.Panicky")
	if reply["ok"] != false {
		t.Fatalf("a result whose encoding panics was answered %v, want an error", reply)
	}
	if e, _ := reply["e"].(string); !strings.Contains(e, "encoding failed") {
		t.Errorf("the error is %q, want it to say the panic", e)
	}
}

func TestEncodeEventPanicBecomesAnError(t *testing.T) {
	w, _ := auditWindow(t)
	for name, emit := range map[string]func() error{
		"Emit":      func() error { return panickyEvent.Emit(w, panicky{}) },
		"EmitPage":  func() error { return panickyEvent.EmitPage(w.Page(), panicky{}) },
		"Broadcast": func() error { return panickyEvent.Broadcast(panicky{}) },
	} {
		var err error
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("%s panicked: %v", name, p)
				}
			}()
			err = emit()
		}()
		if err == nil || !strings.Contains(err.Error(), "encoding failed") {
			t.Errorf("%s returned %v, want an error saying the panic", name, err)
		}
	}
}

func TestCallWithALoneSurrogateIsAnswered(t *testing.T) {
	_, fw := auditWindow(t)
	page(fw, `{"t":"call","id":7,"k":"tok","m":"AuditTest.Echo","a":["\ud83d"]}`)
	reply := received(t, fw, func(m map[string]any) bool { return m["t"] == "reply" && m["id"] == float64(7) })
	// The half of a pair reads as U+FFFD, as nothing else can stand for it
	// in a Go string, and the call is made (upstream #1941).
	if reply["ok"] != true || reply["v"] != "�" {
		t.Fatalf("a call with a lone surrogate was answered %v, want U+FFFD back", reply)
	}
}

func TestAppEventsSkipAnUntrustedPage(t *testing.T) {
	w, fw := auditWindow(t)
	commit(fw, "https://evil.example/")
	before := len(fw.Scripts())
	// Held until the DOM is ready, then released to the page.
	if err := auditEvent.Emit(w, "emitted while loading"); err != nil {
		t.Fatal(err)
	}
	s := newTestSecret()
	hello(fw, s)
	onMain(func() { fw.H.Message(s + `{"t":"dom-ready"}`) })
	// Sent right away.
	if err := auditEvent.Emit(w, "emitted when ready"); err != nil {
		t.Fatal(err)
	}
	if err := auditEvent.Broadcast("broadcast when ready"); err != nil {
		t.Fatal(err)
	}
	// A reply, queued after the events, marks the point by which they have
	// been flushed.
	onMain(func() {
		fw.H.Message(s + `{"t":"call","id":3,"k":"tok3","m":"AuditTest.Echo","a":["marker"]}`)
	})
	wait.Until(t, "the reply after the events", func() bool { return len(scriptsWith(fw, `"tok3"`)) > 0 })
	for _, p := range []string{"emitted while loading", "emitted when ready", "broadcast when ready"} {
		if js := scriptsWith(fw, p); len(js) > 0 {
			t.Errorf("an untrusted page got the app event %q: %.200s", p, js[0])
		}
	}
	locale := false
	for _, js := range fw.Scripts()[before:] {
		locale = locale || strings.Contains(js, `"mygo:locale"`)
	}
	if !locale {
		t.Error("the untrusted page did not get MyGo's own mygo:locale event")
	}
}

// pageValues parses the argument of a __mygo.receive script as the page
// does: a JSON string literal goes through JSON.parse, which makes a
// "__proto__" member an own key, and an array literal is a JavaScript
// literal, in which "__proto__" sets the object's prototype instead.
func pageValues(t *testing.T, js string) []map[string]any {
	t.Helper()
	arg, ok := strings.CutPrefix(js, "__mygo.receive(")
	if !ok {
		t.Fatalf("not a receive script: %.100s", js)
	}
	arg = strings.TrimSuffix(arg, ")")
	var msgs []map[string]any
	if strings.HasPrefix(arg, `"`) {
		var s string
		if err := json.Unmarshal([]byte(arg), &s); err != nil {
			t.Fatalf("bad string literal: %v", err)
		}
		if err := json.Unmarshal([]byte(s), &msgs); err != nil {
			t.Fatalf("bad JSON in the string: %v", err)
		}
		return msgs
	}
	if err := json.Unmarshal([]byte(arg), &msgs); err != nil {
		t.Fatalf("bad script %.100s: %v", js, err)
	}
	var literal func(v any)
	literal = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			delete(v, "__proto__")
			for _, e := range v {
				literal(e)
			}
		case []any:
			for _, e := range v {
				literal(e)
			}
		}
	}
	for _, m := range msgs {
		literal(m)
	}
	return msgs
}

func TestProtoKeyReachesThePage(t *testing.T) {
	_, fw := auditWindow(t)
	page(fw, `{"t":"call","id":9,"k":"tokp","m":"AuditTest.Proto","a":[]}`)
	// The token may be escaped, in a string literal.
	wait.Until(t, "the reply", func() bool { return len(scriptsWith(fw, `tokp`)) > 0 })
	for _, m := range pageValues(t, scriptsWith(fw, `tokp`)[0]) {
		if m["k"] != "tokp" {
			continue
		}
		v, _ := m["v"].(map[string]any)
		if _, own := v["__proto__"]; !own {
			t.Fatalf("the reply's own keys lack __proto__: %v", m)
		}
		return
	}
	t.Fatal("no reply with the call's token")
}

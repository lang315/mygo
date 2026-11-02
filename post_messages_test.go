//go:build !js

package mygo

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/wait"
)

// Backends that take the bridge's messages as web messages (WebView2) get a
// flush as one JSON array, without a script around it, and only the replies
// and channel values of the page that is the window's.

type postService struct{}

var (
	postGate    = make(chan struct{})
	postStarted = make(chan struct{}, 4)
	postOnce    sync.Once
)

func (postService) Echo(s string) string { return s }
func (postService) Fail() error          { return errors.New("it failed") }
func (postService) Slow() string {
	postStarted <- struct{}{}
	<-postGate
	return "secret of the first page"
}
func (postService) Count(n int, ch *Channel[int]) error {
	for i := range n {
		if err := ch.Send(i); err != nil {
			return err
		}
	}
	return nil
}
func (postService) Early(ch *Channel[string]) error {
	err := ch.Send("secret value queued early")
	postStarted <- struct{}{}
	return err
}

// postingWindow returns a ready window that takes web messages.
func postingWindow(t *testing.T) (*Window, *fake.Window) {
	t.Helper()
	postOnce.Do(func() { BindAs("PostTest", postService{}) })
	fb.Posting = true
	t.Cleanup(func() { fb.Posting = false })
	return readyWindow(t, WindowOptions{})
}

// postCall posts a call of method with args, as the bridge of the document with
// the secret of fw would.
func postCall(fw *fake.Window, id int, method, args string) {
	msg := `{"t":"call","id":` + strconv.Itoa(id) + `,"k":"tok","m":"PostTest.` + method + `","a":` + args + `}`
	s := secret(fw)
	onMain(func() { fw.H.Message(s + msg) })
}

// posted returns the messages posted so far, in order, once at least n have.
func posted(t *testing.T, fw *fake.Window, n int) []map[string]json.RawMessage {
	t.Helper()
	var out []map[string]json.RawMessage
	wait.Until(t, strconv.Itoa(n)+" messages posted", func() bool {
		out = nil
		for _, p := range fw.Posts() {
			var batch []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(p), &batch); err != nil {
				t.Fatalf("a post is not a JSON array of messages: %v\n%.200s", err, p)
			}
			out = append(out, batch...)
		}
		return len(out) >= n
	})
	return out
}

func TestPostedMessagesAreOneJSONArray(t *testing.T) {
	_, fw := postingWindow(t)
	for _, s := range fw.Scripts() {
		if strings.Contains(s, "__mygo.receive") {
			t.Fatalf("a flush ran a script: %.100s", s)
		}
	}
	big := strings.Repeat("0123456789abcdef", 1<<16+3) // above a MiB
	const odd = "😀 a b \"quoted\" \\ \x00 é \U0010ffff"
	oddJSON, _ := json.Marshal(odd)
	postCall(fw, 1, "Echo", `["`+big+`"]`)
	postCall(fw, 2, "Echo", `[`+string(oddJSON)+`]`)
	postCall(fw, 3, "Fail", `[]`)
	byID := map[string]map[string]json.RawMessage{}
	wait.Until(t, "the three replies", func() bool {
		for _, m := range posted(t, fw, 1) {
			if string(m["t"]) == `"reply"` {
				byID[string(m["id"])] = m
			}
		}
		return len(byID) == 3
	})
	var v string
	json.Unmarshal(byID["1"]["v"], &v)
	if v != big {
		t.Errorf("a large reply came back as %d bytes, want %d: %.120s", len(v), len(big), byID["1"]["e"])
	}
	json.Unmarshal(byID["2"]["v"], &v)
	if v != odd {
		t.Errorf("a reply with Unicode came back as %q, want %q", v, odd)
	}
	var e string
	json.Unmarshal(byID["3"]["e"], &e)
	if string(byID["3"]["ok"]) != "false" || e != "it failed" {
		t.Errorf("the error reply is %s %q", byID["3"]["ok"], e)
	}
}

func TestPostedChannelValuesKeepTheirOrder(t *testing.T) {
	_, fw := postingWindow(t)
	postCall(fw, 1, "Count", `[2000, 1]`)
	got := posted(t, fw, 2001)
	next := 0
	for _, m := range got {
		if string(m["t"]) != `"chan"` || m["p"] == nil {
			continue
		}
		var n int
		json.Unmarshal(m["p"], &n)
		if n != next {
			t.Fatalf("value %d came where %d was expected", n, next)
		}
		next++
	}
	if next != 2000 {
		t.Fatalf("%d values came, want 2000", next)
	}
}

// What a call of a page asked for is for that page: after a navigation its
// reply, and the values of its channel, are not posted, for the listeners of
// the page that came do not belong to the one that asked.
func TestPostedRepliesDoNotReachAnotherDocument(t *testing.T) {
	// finish is what both cases end with: the new document says its hello and
	// makes a call, answered after what came before it, so when its reply is
	// posted what came before has been posted or dropped.
	finish := func(t *testing.T, fw *fake.Window, marker string) {
		s := newTestSecret()
		hello(fw, s)
		onMain(func() { fw.H.Message(s + `{"t":"dom-ready"}`) })
		onMain(func() {
			fw.H.Message(s + `{"t":"call","id":2,"k":"tok2","m":"PostTest.Echo","a":["second page"]}`)
		})
		wait.Until(t, "the reply of the new document", func() bool {
			for _, m := range posted(t, fw, 1) {
				if string(m["id"]) == "2" {
					return true
				}
			}
			return false
		})
		for _, m := range posted(t, fw, 1) {
			if strings.Contains(string(m["v"]), marker) || strings.Contains(string(m["p"]), marker) {
				t.Fatalf("the new document got a message of the first: %v", m)
			}
		}
	}
	t.Run("reply", func(t *testing.T) {
		_, fw := postingWindow(t)
		gate := postGate
		postGate = make(chan struct{})
		defer func() { postGate = gate }()
		release := postGate
		postCall(fw, 1, "Slow", `[]`)
		<-postStarted
		// The page navigates to another document; the first call answers after.
		commit(fw, "https://other.example/")
		close(release)
		finish(t, fw, "secret of the first page")
	})
	// A value queued before the navigation and flushed after it is dropped by
	// its page, not by the closed channel: the main thread is held from the
	// call until the commit, so the flush waits behind both.
	t.Run("queued value", func(t *testing.T) {
		_, fw := postingWindow(t)
		s := secret(fw)
		onMain(func() {
			fw.H.Message(s + `{"t":"call","id":1,"k":"tok","m":"PostTest.Early","a":[1]}`)
			<-postStarted // the value is queued, its flush is behind this task
			fw.H.NavigationCommitted("https://other.example/")
		})
		finish(t, fw, "secret value queued early")
	})
}

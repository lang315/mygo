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
func (postService) Stream(ch *Channel[string]) error {
	postStarted <- struct{}{}
	<-postGate
	return ch.Send("secret value of the first page")
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
	for _, tc := range []struct{ name, method, args, marker string }{
		{"reply", "Slow", `[]`, "secret of the first page"},
		{"channel", "Stream", `[1]`, "secret value of the first page"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, fw := postingWindow(t)
			gate := postGate
			postGate = make(chan struct{})
			defer func() { postGate = gate }()
			release := postGate
			postCall(fw, 1, tc.method, tc.args)
			<-postStarted
			// The page navigates to another document, which says its hello.
			commit(fw, "https://other.example/")
			s := newTestSecret()
			hello(fw, s)
			onMain(func() { fw.H.Message(s + `{"t":"dom-ready"}`) })
			close(release) // the first call answers now
			// A call of the new document, answered after: when its reply is
			// posted, what came before it has been posted or dropped.
			onMain(func() {
				fw.H.Message(s + `{"t":"call","id":2,"k":"tok2","m":"PostTest.Echo","a":["second page"]}`)
			})
			got := posted(t, fw, 1)
			for _, m := range got {
				if strings.Contains(string(m["v"]), tc.marker) || strings.Contains(string(m["p"]), tc.marker) {
					t.Fatalf("the new document got a message of the first: %s", m["v"])
				}
			}
			wait.Until(t, "the reply of the new document", func() bool {
				for _, m := range posted(t, fw, 1) {
					if string(m["id"]) == "2" {
						return true
					}
				}
				return false
			})
			for _, m := range posted(t, fw, 1) {
				if strings.Contains(string(m["v"]), tc.marker) || strings.Contains(string(m["p"]), tc.marker) {
					t.Fatalf("the new document got a message of the first: %s", m)
				}
			}
		})
	}
}

// While a navigation of the main frame runs, the messages wait: the document
// it replaces may be gone when they are posted, and the next one's scripts
// hear what they were. A navigation that ends without a commit (canceled, or
// within the document) posts them to the page that stayed; one that commits
// drops those the old page asked.
func TestPostedMessagesWaitForANavigation(t *testing.T) {
	for _, commitIt := range []bool{false, true} {
		name := "canceled"
		if commitIt {
			name = "committed"
		}
		t.Run(name, func(t *testing.T) {
			_, fw := postingWindow(t)
			gate := postGate
			postGate = make(chan struct{})
			defer func() { postGate = gate }()
			release := postGate
			postCall(fw, 1, "Slow", `[]`)
			<-postStarted
			before := len(fw.Posts()) // the window's own events, at dom-ready
			onMain(func() { fw.H.NavigationStarted("https://other.example/") })
			close(release)
			// Another message after, for the page that is current once it settles.
			postCall(fw, 2, "Echo", `["after"]`)
			// Both were enqueued; nothing is posted while the navigation runs.
			wait.Until(t, "the replies enqueued", func() bool {
				var n int
				onMain(func() {
					fw.H.Message("") // reaches the main loop: the flushes ran
					n = 1
				})
				return n == 1
			})
			if n := len(fw.Posts()) - before; n != 0 {
				t.Fatalf("%d posts while a navigation ran", n)
			}
			var s string
			if commitIt {
				commit(fw, "https://other.example/")
				s = newTestSecret()
				hello(fw, s)
			} else {
				s = secret(fw)
				onMain(func() { fw.H.(interface{ NavigationSettled() }).NavigationSettled() })
			}
			if !commitIt {
				got := posted(t, fw, 2)
				var ids []string
				for _, m := range got {
					if string(m["t"]) == `"reply"` {
						ids = append(ids, string(m["id"]))
					}
				}
				wait.Until(t, "both replies posted", func() bool {
					ids = ids[:0]
					for _, m := range posted(t, fw, 1) {
						if string(m["t"]) == `"reply"` {
							ids = append(ids, string(m["id"]))
						}
					}
					return len(ids) == 2
				})
				if ids[0] != "1" || ids[1] != "2" {
					t.Fatalf("the replies came in the order %v after the navigation was canceled", ids)
				}
				return
			}
			// Committed: the call of the first page is not posted; a call of the new one is.
			onMain(func() { fw.H.Message(s + `{"t":"call","id":3,"k":"tok3","m":"PostTest.Echo","a":["new"]}`) })
			wait.Until(t, "the reply of the new page", func() bool {
				for _, m := range posted(t, fw, 1) {
					if string(m["id"]) == "3" {
						return true
					}
				}
				return false
			})
			for _, m := range posted(t, fw, 1) {
				if id := string(m["id"]); string(m["t"]) == `"reply"` && id != "3" {
					t.Fatalf("a reply of the page that went was posted: %s", id)
				}
			}
		})
	}
}

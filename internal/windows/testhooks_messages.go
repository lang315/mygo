//go:build windows && (amd64 || arm64)

package windows

import (
	"strings"
	"sync"
	"unsafe"
)

// messageHook and commitHook are set by tests (TestWatchMessages) and nil
// otherwise: the first hears each web message with the URL of the document
// it says it came from and that of the document the window shows, the
// second each commit of the main frame.
var (
	messageHook func(source, current string)
	commitHook  func()
)

// messageSource returns the URL of the document that sent a web message,
// as the window names its documents (an app's scheme for its host).
func (w *window) messageSource(args uintptr) string {
	var p uintptr
	if failed(comCall(args, msgGetSource, uintptr(unsafe.Pointer(&p)))) {
		return ""
	}
	return w.appURL(takeWstr(p))
}

// documentURL is a URL without its fragment, which a navigation inside a
// document changes and its identity does not.
func documentURL(u string) string {
	u, _, _ = strings.Cut(u, "#")
	return u
}

var messageStats struct {
	sync.Mutex
	heard, other, commits int
	others                []string
}

// TestWatchMessages starts counting the web messages the windows hear, those
// whose source is not the document shown, and the commits of main frames.
func TestWatchMessages() {
	messageStats.Lock()
	defer messageStats.Unlock()
	messageStats.heard, messageStats.other, messageStats.commits, messageStats.others = 0, 0, 0, nil
	messageHook = func(source, current string) {
		messageStats.Lock()
		defer messageStats.Unlock()
		messageStats.heard++
		if documentURL(source) != documentURL(current) {
			messageStats.other++
			if len(messageStats.others) < 5 {
				messageStats.others = append(messageStats.others, source+" while at "+current)
			}
		}
	}
	commitHook = func() {
		messageStats.Lock()
		defer messageStats.Unlock()
		messageStats.commits++
	}
}

// TestMessageStats returns what TestWatchMessages counted: the messages
// heard, how many of them came from a document other than the one shown (and
// the first few, as "source while at current"), and the main frame's commits.
func TestMessageStats() (heard, other int, others []string, commits int) {
	messageStats.Lock()
	defer messageStats.Unlock()
	return messageStats.heard, messageStats.other, append([]string(nil), messageStats.others...), messageStats.commits
}

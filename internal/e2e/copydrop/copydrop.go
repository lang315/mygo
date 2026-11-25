// Package copydrop decides what internal/e2e's
// TestContentWindowWebViewEditMenu does with a Copy whose write never came
// on Windows. WebView2 (157) now and then loses the browser's write of a
// page's copy without opening the clipboard: the page's copy event ran, no
// program held the clipboard, and the write was never tried (win-copy-flake,
// measured at 9 in 200 Copies in CI). That shape, and only that one, may be
// skipped, once a test process, so a WebView2 that drops every write still
// fails. Its own package, as internal/e2e's tests run only in a desktop
// session: these run anywhere.
package copydrop

import (
	"encoding/json"
	"time"
)

// MaxGap is the longest gap between the sampler's tries for its "never
// busy" to prove no holder: Chromium's write retries OpenClipboard for 20
// ms or more, so a hold that lost it would span a try.
const MaxGap = 15 * time.Millisecond

// Window is how long after Copy the sampler must have tried the clipboard.
const Window = 300 * time.Millisecond

// Facts is what the test recorded about a Copy whose write never came.
type Facts struct {
	PageCopied  bool          // the page's copy event ran with the selection, not prevented
	Moved       bool          // the clipboard's sequence moved after Copy
	Sampled     time.Duration // how long after Copy the sampler tried the clipboard
	MaxGap      time.Duration // the longest gap between its tries
	Busy        bool          // a try found the clipboard held
	DropsBefore int           // drops already skipped in this test process
}

// Skip reports whether the Copy is WebView2's dropped write, which the test
// may skip; any other shape fails it.
func Skip(f Facts) bool {
	return f.PageCopied && !f.Moved && f.Sampled >= Window && f.MaxGap <= MaxGap && !f.Busy && f.DropsBefore == 0
}

// PageCopied reports whether the page's state, as the test reads it (JSON
// with copies and copied), says one copy event ran with selection, not
// prevented.
func PageCopied(page, selection string) bool {
	var p struct {
		Copies int `json:"copies"`
		Copied []struct {
			Selection string `json:"selection"`
			Prevented bool   `json:"prevented"`
		} `json:"copied"`
	}
	if json.Unmarshal([]byte(page), &p) != nil || p.Copies != 1 || len(p.Copied) != 1 {
		return false
	}
	return p.Copied[0].Selection == selection && !p.Copied[0].Prevented
}

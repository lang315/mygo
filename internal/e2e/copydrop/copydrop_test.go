package copydrop

import (
	"testing"
	"time"
)

func TestSkip(t *testing.T) {
	// A failure of CI's run 45598945847, as the test recorded it.
	drop := Facts{PageCopied: true, Sampled: 301 * time.Millisecond, MaxGap: 2 * time.Millisecond}
	if !Skip(drop) {
		t.Fatalf("the dropped write %+v fails, want a skip", drop)
	}
	for name, change := range map[string]func(*Facts){
		"the page did not copy":          func(f *Facts) { f.PageCopied = false },
		"the clipboard changed":          func(f *Facts) { f.Moved = true },
		"sampled less than the window":   func(f *Facts) { f.Sampled = Window - time.Millisecond },
		"not sampled":                    func(f *Facts) { f.Sampled = 0 },
		"a gap a holder could hide in":   func(f *Facts) { f.MaxGap = MaxGap + time.Millisecond },
		"a program held the clipboard":   func(f *Facts) { f.Busy = true },
		"a second drop in the process":   func(f *Facts) { f.DropsBefore = 1 },
		"a third drop in the process":    func(f *Facts) { f.DropsBefore = 2 },
		"the window exactly, gap at max": nil,
	} {
		f := drop
		if change == nil {
			f.Sampled, f.MaxGap = Window, MaxGap
			if !Skip(f) {
				t.Errorf("%s: %+v fails, want a skip", name, f)
			}
			continue
		}
		change(&f)
		if Skip(f) {
			t.Errorf("%s: %+v skips, want a failure", name, f)
		}
	}
}

func TestPageCopied(t *testing.T) {
	for page, want := range map[string]bool{
		// A failure of CI's run 45570299397, as the test read the page.
		`{"commands":["selectAll: true","copy: true"],"copies":1,"copied":[{"selection":"hello","focused":true,"active":"e","prevented":false,"at":1795595296088}],"selection":"hello","focused":true}`: true,
		`{"copies":1,"copied":[{"selection":"hello","prevented":true}]}`:      false,
		`{"copies":1,"copied":[{"selection":"","prevented":false}]}`:          false,
		`{"copies":0,"copied":[]}`:                                            false,
		`{"copies":2,"copied":[{"selection":"hello"},{"selection":"hello"}]}`: false,
		``:         false,
		`not json`: false,
	} {
		if got := PageCopied(page, "hello"); got != want {
			t.Errorf("PageCopied(%s) = %v, want %v", page, got, want)
		}
	}
}

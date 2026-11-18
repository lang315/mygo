package editor

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

// An edit of one line while the find bar is open keeps the array of the
// matches: with 100,000 of them it allocates well under the 1.6 MB a new
// array takes.
func TestFindEditDoesNotAllocateTheMatches(t *testing.T) {
	const n = 100000
	ct := newEditorTest(t, "", "|"+strings.Repeat("cat x\n\n", n))
	ct.st.Find("cat")
	ct.tt.Frame()
	ed := &ct.st.ed
	at := 7*(n/2) + 4
	edit := func(i int) {
		text := "y"
		if i%2 == 1 {
			text = "x"
		}
		ed.apply([]Edit{{Start: at, End: at + 1, Text: text}}, editOther, nil)
		ed.find.search(ed)
	}
	// The first edit makes the spare array, and the second uses it.
	edit(0)
	edit(1)
	const edits = 4
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < edits; i++ {
		edit(i)
	}
	runtime.ReadMemStats(&after)
	if got := (after.TotalAlloc - before.TotalAlloc) / edits; got > 200<<10 {
		t.Errorf("an edit allocated %d bytes with %d matches, want under 200 KB", got, n)
	}
	if len(ed.find.matches) != n {
		t.Fatalf("%d matches, want %d", len(ed.find.matches), n)
	}
}

// Edits in a row at the first, a middle and the last line, each searched
// again on the array the one before left, leave the matches a search of
// the whole text finds.
func TestFindEditsInARowKeepTheMatchesRight(t *testing.T) {
	const n = 2000
	ct := newEditorTest(t, "", "|"+strings.Repeat("cat x\n\n", n))
	ct.st.Find("cat")
	ct.tt.Frame()
	ed := &ct.st.ed
	mid := 7 * (n / 2)
	last := 7 * (n - 1)
	for i, e := range []Edit{
		{Start: 4, End: 5, Text: "cat"},                // the first line gets a second match
		{Start: mid + 2, End: mid + 5, Text: "dog"},    // a middle line loses its match
		{Start: last + 7, End: last + 7, Text: " cat"}, // the last line gets another
	} {
		ed.apply([]Edit{e}, editOther, nil)
		ct.tt.Frame()
		got := slices.Clone(ed.find.matches)
		fresh := newEditorTest(t, "", "|"+ct.st.Buffer.String())
		fresh.st.Find("cat")
		fresh.tt.Frame()
		if want := fresh.st.ed.find.matches; !slices.Equal(got, want) || len(want) == 0 {
			t.Fatalf("edit %d: %d matches, a full search finds %d", i, len(got), len(want))
		}
	}
}

package editor

import (
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkFindEditOneLine measures one edit of a line beside a match while
// the find bar is open: the edit moves the matches, and searchLines finds
// the line's again.
func BenchmarkFindEditOneLine(b *testing.B) {
	for _, n := range []int{10000, 100000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			st := &State{Buffer: NewBuffer(strings.Repeat("cat x\n\n", n))}
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, st).Fill().AutoFocus()
			}, 600, 400)
			tt.Frame()
			st.Find("cat")
			tt.Frame()
			ed := &st.ed
			at := 7*(n/2) + 4
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				text := "y"
				if i%2 == 1 {
					text = "x"
				}
				ed.apply([]Edit{{Start: at, End: at + 1, Text: text}}, editOther, nil)
				ed.find.search(ed)
			}
			b.StopTimer()
			if len(ed.find.matches) != n {
				b.Fatalf("%d matches, want %d", len(ed.find.matches), n)
			}
		})
	}
}

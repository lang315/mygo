package text

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/text/width"
)

// TestEastAsianWide: eastAsianWide is x/text/width's East Asian Wide for every rune.
func TestEastAsianWide(t *testing.T) {
	var table strings.Builder
	start, bad := rune(-1), 0
	for r := rune(0); r <= 0x10FFFF; r++ {
		w := width.LookupRune(r).Kind() == width.EastAsianWide
		if w && start < 0 {
			start = r
		}
		if !w && start >= 0 {
			fmt.Fprintf(&table, "\t{%#x, %#x},\n", start, r-1)
			start = -1
		}
		if eastAsianWide(r) != w {
			if bad++; bad <= 5 {
				t.Errorf("eastAsianWide(%#x) = %v, x/text/width says %v", r, !w, w)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d runes differ; the table of ranges now is:\n%s", bad, table.String())
	}
}

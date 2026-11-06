package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func indexWidth(tb *DataGridViewWidget[acct, int]) float32 {
	return tb.columns[0].Width
}

// shownWhole fails unless the number text, which the top row shows, is not cut:
// a cut text reports the room it has, less than the same text alone.
func shownWhole(t *testing.T, tt *ui.Tester, text string, column float32) {
	t.Helper()
	got, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %s in %q", text, tt.Texts()[:min(8, len(tt.Texts()))])
	}
	if whole := find(t, tester(t, HBox(0).Add(Label(text))), text); got.W < whole.W {
		t.Fatalf("%s is cut to %v wide, it needs %v (the # column is %v wide)", text, got.W, whole.W, column)
	}
}

// 4 is the widest digit of the system font on macOS, 9 is not: a column
// measured for the 9s cut the 8s of 88888.
func TestIndexColumnShowsTheLargestNumberWhole(t *testing.T) {
	for _, grid := range []bool{false, true} {
		tb := testTable(manyAccts(88_888)...).IndexColumnBySource().Sort("#", true).SetGridLines(grid)
		tt := tester(t, tb)
		shownWhole(t, tt, "88888", indexWidth(tb))
	}
	tb := testTable(manyAccts(99_999)...).IndexColumnBySource().Sort("#", true).SetGridLines(true)
	shownWhole(t, tester(t, tb), "99999", indexWidth(tb))
}

func TestIndexColumnOfASmallTableKeeps56(t *testing.T) {
	for _, tb := range []*DataGridViewWidget[acct, int]{testTableOf3().IndexColumn(), testTableOf3().IndexColumnBySource()} {
		tester(t, tb)
		if w := indexWidth(tb); w != 56 {
			t.Fatalf("the # column of 3 rows is %v wide", w)
		}
	}
}

func TestIndexColumnFollowsTheDigitsOfTheRows(t *testing.T) {
	tb := testTable(manyAccts(9999)...).IndexColumnBySource()
	tt := ui.NewTester(content(tb), 1280, 800)
	tt.Frame()
	narrow := indexWidth(tb)
	tb.AddRows([]acct{{id: 9999, name: "x"}}) // 10000 rows
	tt.Frame()
	wide := indexWidth(tb)
	if wide <= narrow {
		t.Fatalf("the # column stayed %v wide across 9999 -> 10000 rows (now %v)", narrow, wide)
	}
	tb.DeleteRows(9999)
	tt.Frame()
	if got := indexWidth(tb); got != narrow {
		t.Fatalf("after deleting back to 9999 rows the # column is %v, want %v", got, narrow)
	}
}

// The numbers of IndexColumnBySource are places in the source, so a search
// that shows few rows leaves them wide; those of IndexColumn are positions
// among the rows shown, so it narrows.
func TestIndexColumnCountsTheRowsOfItsKind(t *testing.T) {
	bySource := testTable(manyAccts(100_000)...).IndexColumnBySource().Search("Name", "acct00000")
	tester(t, bySource)
	plain := testTable(manyAccts(100_000)...).IndexColumn().Search("Name", "acct00000")
	tester(t, plain)
	if w := indexWidth(bySource); w <= 56 {
		t.Fatalf("by source, a search narrowed the # column to %v", w)
	}
	if w := indexWidth(plain); w != 56 {
		t.Fatalf("a search of 10 rows left the # column %v wide", w)
	}
}

func TestIndexColumnFollowsTheFontSize(t *testing.T) {
	tb := testTable(manyAccts(99_999)...).IndexColumnBySource()
	tt := ui.NewTester(content(tb), 1280, 800)
	tt.Frame()
	normal := indexWidth(tb)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	if big := indexWidth(tb); big <= normal {
		t.Fatalf("the # column stayed %v wide when the text scale went to 1.5 (now %v)", normal, big)
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	tt.Frame()
	if back := indexWidth(tb); back != normal {
		t.Fatalf("the # column is %v after the text scale went back to 1, want %v", back, normal)
	}
}

func TestIndexColumnWideningKeepsTheWidthsOfTheOthers(t *testing.T) {
	tb := testTable(manyAccts(9999)...).IndexColumnBySource().SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": 300}})
	tt := ui.NewTester(content(tb), 1280, 800)
	tt.Frame()
	gap := func() float32 { return find(t, tt, "Score").X - find(t, tt, "Name").X }
	before, narrow := gap(), indexWidth(tb)
	tb.AddRows([]acct{{id: 9999, name: "x"}})
	tt.Frame()
	if indexWidth(tb) <= narrow {
		t.Fatalf("the # column did not widen (%v)", narrow)
	}
	if after := gap(); after != before {
		t.Fatalf("Name was %v wide before the # column widened, %v after", before, after)
	}
}

package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func indexWidth(tb *DataGridViewWidget[acct, int]) float32 {
	return tb.columns[0].Width
}

func TestIndexColumnShowsTheLargestNumberWhole(t *testing.T) {
	tb := testTable(manyAccts(100_000)...).IndexColumnBySource().Sort("#", true)
	tt := tester(t, tb)
	text, ok := tt.Find("100000")
	if !ok {
		t.Fatalf("no 100000 in %q", tt.Texts()[:min(8, len(tt.Texts()))])
	}
	// A cut number reports the room it has, less than the same text alone.
	whole := find(t, tester(t, HBox(0).Add(Label("100000"))), "100000")
	if text.W < whole.W {
		t.Fatalf("100000 is cut to %v wide, it needs %v (the # column is %v wide)", text.W, whole.W, indexWidth(tb))
	}
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

func TestIndexColumnWideningKeepsTheWidthsOfTheOthers(t *testing.T) {
	tb := testTable(manyAccts(9999)...).IndexColumnBySource().SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": 300}})
	tt := ui.NewTester(content(tb), 1280, 800)
	tt.Frame()
	gap := func() float32 { return find(t, tt, "Score").X - find(t, tt, "Name").X }
	before := gap()
	tb.AddRows([]acct{{id: 9999, name: "x"}})
	tt.Frame()
	if after := gap(); after != before {
		t.Fatalf("Name was %v wide before the # column widened, %v after", before, after)
	}
}

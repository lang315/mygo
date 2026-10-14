package widget

import (
	"cmp"
	"image/color"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

type acct struct {
	id    int
	name  string
	score int
}

// testTable is a table of accounts by id: Name sorts and searches as
// text, Score sorts as a number.
func testTable(rows ...acct) *TableWidget[acct, int] {
	return Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name },
			Width: 200, Sortable: true, Filterable: true},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) },
			Compare: func(a, b *acct) int { return cmp.Compare(a.score, b.score) },
			Width:   100, Align: End, Sortable: true},
	).SetRows(rows).Expand()
}

// inOrder fails unless names show from the top down in that order.
func inOrder(t *testing.T, tt *ui.Tester, names ...string) {
	t.Helper()
	for i := 1; i < len(names); i++ {
		if a, b := find(t, tt, names[i-1]), find(t, tt, names[i]); b.Y <= a.Y {
			t.Fatalf("%q (y %v) not above %q (y %v)", names[i-1], a.Y, names[i], b.Y)
		}
	}
}

func TestTableShowsRows(t *testing.T) {
	tt := tester(t, testTable(acct{1, "ann", 30}, acct{2, "bob", 10}))
	for _, s := range []string{"Name", "Score", "ann", "bob", "30", "10"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestTableRowsAppendAndReplace(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{1, "amy", 2}) // the last of a key stays
	if tb.Len() != 1 {
		t.Fatalf("Len %d after SetRows with one key twice", tb.Len())
	}
	tb.AddRow(acct{2, "bob", 3}).AddRow(acct{1, "ada", 4})
	tt := tester(t, tb)
	if tb.Len() != 2 || tt.HasText("amy") || !tt.HasText("ada") {
		t.Fatalf("Len %d, texts %q", tb.Len(), tt.Texts())
	}
	inOrder(t, tt, "ada", "bob") // appended after; replaced in place
	tb.UpdateRow(2, func(a *acct) { a.name = "ben" })
	tt.Frame()
	if !tt.HasText("ben") {
		t.Fatalf("UpdateRow not shown: %q", tt.Texts())
	}
	if r, ok := tb.Row(2); !ok || r.name != "ben" {
		t.Fatalf("Row(2) = %+v, %v", r, ok)
	}
	tb.DeleteRows(1, 99)
	tt.Frame()
	if tb.Len() != 1 || tt.HasText("ada") {
		t.Fatalf("after DeleteRows: Len %d, texts %q", tb.Len(), tt.Texts())
	}
	if _, ok := tb.Row(1); ok {
		t.Fatal("Row(1) after DeleteRows")
	}
	tb.Clear()
	tt.Frame()
	if tb.Len() != 0 || tt.HasText("ben") {
		t.Fatalf("after Clear: Len %d, texts %q", tb.Len(), tt.Texts())
	}
}

func TestTableSortsByHeader(t *testing.T) {
	// As text, the scores would sort 10, 100, 9.
	tt := tester(t, testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100}))
	click(t, tt, "Name")
	inOrder(t, tt, "ann", "bob", "cat")
	click(t, tt, "Name")
	inOrder(t, tt, "cat", "bob", "ann")
	click(t, tt, "Score")
	inOrder(t, tt, "ann", "cat", "bob")
}

func TestTableClearSortShowsRowsAsGiven(t *testing.T) {
	tb := testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100})
	tt := tester(t, tb)
	click(t, tt, "Name")
	inOrder(t, tt, "ann", "bob", "cat")
	tb.ClearSort()
	tt.Frame()
	inOrder(t, tt, "cat", "ann", "bob")
}

func TestTableIndexColumnShowsShownPositions(t *testing.T) {
	// Scores with no 1 or 2 in them: Find matches whole texts, the first.
	// A name finds its row, which ui labels with the first column's text.
	tt := tester(t, testTable(acct{1, "cat", 30}, acct{2, "ann", 40}).IndexColumn())
	click(t, tt, "Name")
	for _, p := range [][2]string{{"1", "ann"}, {"2", "cat"}} {
		n, row := find(t, tt, p[0]), find(t, tt, p[1])
		if y := n.Y + n.H/2; y < row.Y || y > row.Y+row.H {
			t.Fatalf("%s at %v, not in the row of %s at %v", p[0], n, p[1], row)
		}
	}
}

func TestTableCellsShowColorAndImage(t *testing.T) {
	png := testPNG(t, color.White)
	bmp := decodeImage(png)
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name },
			Color: func(a *acct) ui.Color { return ui.RGB(200, 0, 0) },
			Image: func(a *acct) *ui.Bitmap { return bmp }},
	).SetRows([]acct{{1, "ann", 0}}).Expand()
	tt := tester(t, tb)
	if !tt.HasText("ann") {
		t.Fatalf("no ann in %q", tt.Texts())
	}
}

func TestTableStaleRowHasZeroKey(t *testing.T) {
	tb := testTable(acct{1, "a", 0}, acct{2, "b", 0}, acct{3, "c", 0})
	tester(t, tb)
	tb.DeleteRows(1, 2, 3) // the view still shows three rows until it syncs
	if k := tb.keyAt(2); k != 0 {
		t.Fatalf("keyAt(2) of a deleted row = %v", k)
	}
	if s := tb.labelAt(2); s != "" {
		t.Fatalf("labelAt(2) of a deleted row = %q", s)
	}
}

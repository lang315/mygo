package widget

import "testing"

// IntColumn sorts as a number, not as text: 9 comes before 10.
func TestIntColumnSortsAsANumberAndAlignsToTheEnd(t *testing.T) {
	col := IntColumn("Score", 100, func(a *acct) int { return a.score })
	if !col.Sortable || col.Align != End || col.Title != "Score" || col.Width != 100 {
		t.Fatalf("column %+v", col)
	}
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200},
		col,
	).SetRows([]acct{{1, "ann", 10}, {2, "bob", 9}, {3, "cat", 100}}).Expand()
	tt := tester(t, tb)
	click(t, tt, "Score")
	inOrder(t, tt, "bob", "ann", "cat") // 9, 10, 100; as text: 10, 100, 9
	click(t, tt, "Score")
	inOrder(t, tt, "cat", "ann", "bob")
	if !tt.HasText("100") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

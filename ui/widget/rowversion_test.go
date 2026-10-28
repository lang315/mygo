package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// versionsOf returns RowVersion of rows 0 to n-1 of a built control's list.
func versionsOf(s *ui.ListState, n int) []uint64 {
	v := make([]uint64, n)
	for i := range v {
		v[i] = s.RowVersion(i)
	}
	return v
}

// changed returns which of two versions differ.
func changed(before, after []uint64) (rows []int) {
	for i := range before {
		if before[i] != after[i] {
			rows = append(rows, i)
		}
	}
	return rows
}

// wantChanged fails unless exactly the rows want changed their version.
func wantChanged(t *testing.T, what string, before, after []uint64, want ...int) {
	t.Helper()
	if got := changed(before, after); !slices.Equal(got, want) {
		t.Errorf("%s: the rows %v changed their version, want %v", what, got, want)
	}
}

// Every way to change what a row shows changes the versions of the rows it
// changes, and of no other: a table keeps the others, built as they were.
func TestRowVersionRowSource(t *testing.T) {
	src := NewRowSource(acctKey)
	src.SetRows([]acct{{1, "ann", 1}, {2, "bob", 2}, {3, "cat", 3}})
	v1, v2, v3 := src.Version(1), src.Version(2), src.Version(3)
	if v1 == 0 || v1 == v2 || v2 == v3 {
		t.Fatalf("versions %d %d %d: each row has its own", v1, v2, v3)
	}
	src.UpdateRow(2, func(a *acct) { a.name = "zed" })
	if src.Version(1) != v1 || src.Version(3) != v3 || src.Version(2) == v2 {
		t.Errorf("UpdateRow(2): versions %d %d %d, were %d %d %d", src.Version(1), src.Version(2), src.Version(3), v1, v2, v3)
	}
	v2 = src.Version(2)
	src.AddRows([]acct{{3, "cat!", 3}, {4, "dan", 4}})
	if src.Version(1) != v1 || src.Version(2) != v2 || src.Version(3) == v3 || src.Version(4) == 0 {
		t.Errorf("AddRows (3 again, 4 new): versions %d %d %d %d", src.Version(1), src.Version(2), src.Version(3), src.Version(4))
	}
	v3 = src.Version(3)
	src.DeleteRows(1)
	if src.Version(1) != 0 || src.Version(2) != v2 {
		t.Errorf("DeleteRows(1): versions %d %d, want 0 and %d", src.Version(1), src.Version(2), v2)
	}
	src.SetRows([]acct{{2, "bob", 2}, {3, "cat", 3}})
	if src.Version(2) == v2 || src.Version(3) == v3 {
		t.Errorf("SetRows: the rows kept their versions %d %d", src.Version(2), src.Version(3))
	}
}

func TestRowVersionDataGridView(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3})
	tt := tester(t, tb)
	before := versionsOf(&tb.list, 3)
	if slices.Contains(before, 0) {
		t.Fatalf("versions %v: a row of data has one", before)
	}
	tb.UpdateRow(2, func(a *acct) { a.name = "zed" })
	tt.Frame()
	after := versionsOf(&tb.list, 3)
	wantChanged(t, "UpdateRow(2)", before, after, 1)
	before = after
	tb.AddRow(acct{3, "cat!", 3})
	tt.Frame()
	after = versionsOf(&tb.list, 3)
	wantChanged(t, "AddRow of an existing key", before, after, 2)
	before = after
	tb.IndexColumn()
	tt.Frame()
	after = versionsOf(&tb.list, 3)
	wantChanged(t, "IndexColumn", before, after, 0, 1, 2)
}

func TestRowVersionListBox(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	before := versionsOf(&lb.list, 4)
	lb.UpdateItem("b", func(i *litem) { i.name = "Blueberry" })
	tt.Frame()
	after := versionsOf(&lb.list, 4)
	wantChanged(t, "UpdateItem(b)", before, after, 1)
	before = after
	lb.AddItems([]litem{{"c", "Cherry!"}})
	tt.Frame()
	after = versionsOf(&lb.list, 4)
	wantChanged(t, "AddItems of an existing key", before, after, 2)
	before = after
	lb.SetImage(func(*litem) *ui.Bitmap { return nil })
	tt.Frame()
	after = versionsOf(&lb.list, 4)
	wantChanged(t, "SetImage", before, after, 0, 1, 2, 3)
	before = after
	lb.SetItems(fruit)
	tt.Frame()
	after = versionsOf(&lb.list, 4)
	wantChanged(t, "SetItems", before, after, 0, 1, 2, 3)
}

// A check box shows whether the item is checked: checking one changes its row.
func TestRowVersionCheckedListBox(t *testing.T) {
	lb := newChecked().SetItems(fruit)
	tt := tester(t, lb)
	before := versionsOf(&lb.list, 4)
	lb.SetChecked([]string{"c"}, true)
	tt.Frame()
	after := versionsOf(&lb.list, 4)
	wantChanged(t, "SetChecked(c)", before, after, 2)
}

func TestRowVersionListView(t *testing.T) {
	lv := newView()
	tt := tester(t, lv)
	before := versionsOf(&lv.Details().list, 4)
	lv.UpdateRow("c", func(v *vitem) { v.name = "Cherries" })
	tt.Frame()
	after := versionsOf(&lv.Details().list, 4)
	wantChanged(t, "UpdateRow(c)", before, after, 2)
	before = after
	lv.SetImage(func(*vitem) *ui.Bitmap { return nil })
	tt.Frame()
	after = versionsOf(&lv.Details().list, 4)
	wantChanged(t, "SetImage", before, after, 0, 1, 2, 3)
}

func TestRowVersionTreeView(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	// A, A1, A2 and B show, and B1 once B is open.
	tv.Expand("A", "B")
	tt.Frame()
	rows := tv.state.Rows()
	version := func() []uint64 {
		v := make([]uint64, rows)
		for i := range v {
			v[i] = tv.state.RowVersion(tv.state.Item(i))
		}
		return v
	}
	before := version()
	tv.UpdateNode("A1", func(n *tnode) { n.name = "Alpha uno" })
	tt.Frame()
	after := version()
	if got := changed(before, after); len(got) != 1 || tv.state.Item(got[0]).k != "A1" {
		t.Errorf("UpdateNode(A1): the rows %v changed their version", got)
	}
	before = after
	tv.SetImage(func(*tnode) *ui.Bitmap { return nil })
	tt.Frame()
	after = version()
	if got := changed(before, after); len(got) != rows {
		t.Errorf("SetImage: %d of %d rows changed their version", len(got), rows)
	}
}

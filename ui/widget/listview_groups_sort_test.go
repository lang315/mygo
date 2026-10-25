package widget

import (
	"slices"
	"testing"
)

// The rows of a group keep the order of the table: a sort in Details, with no
// row changed and no group, makes the groups again.
func TestListViewGroupsFollowTheSortOfTheTable(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	lv.SetView(ViewList)
	tt.Frame()
	tt.Frame()
	if yOf(t, tt, "Cherry") >= yOf(t, tt, "Date") {
		t.Fatalf("setup: Cherry is not above Date in the group drupe")
	}
	groupsInOrder := func(when string) {
		t.Helper()
		if !(yOf(t, tt, "berry (1)") < yOf(t, tt, "drupe (2)") && yOf(t, tt, "drupe (2)") < yOf(t, tt, "pome (1)")) {
			t.Errorf("%s, the groups are not berry, drupe, pome, by title", when)
		}
	}
	groupsInOrder("before the sort")
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	click(t, tt, "Name") // ascending, as it is
	click(t, tt, "Name") // descending
	lv.SetView(ViewList)
	tt.Frame()
	tt.Frame()
	if yOf(t, tt, "Date") >= yOf(t, tt, "Cherry") {
		t.Fatalf("after a sort by Name, descending, Date is not above Cherry in the group drupe")
	}
	// The groups do not move with the sort: the app gave no order, so they go
	// by title, not by the row each shows first.
	groupsInOrder("after the sort")
}

// The groups the app names come first; the others follow by title, not in the
// order their rows show.
func TestListViewGroupsNamedFirstThenByTitle(t *testing.T) {
	lv := groupedView(ViewList).SetGroupOrder("pome")
	tt := tester(t, lv)
	want := []ListViewGroup{{"pome", 1}, {"berry", 1}, {"drupe", 2}}
	if got := lv.Groups(); !slices.Equal(got, want) {
		t.Errorf("Groups() = %v, want %v", got, want)
	}
	lv.SetGroupOrder("drupe")
	tt.Frame()
	tt.Frame()
	want = []ListViewGroup{{"drupe", 2}, {"berry", 1}, {"pome", 1}}
	if got := lv.Groups(); !slices.Equal(got, want) {
		t.Errorf("Groups() = %v, want %v", got, want)
	}
}

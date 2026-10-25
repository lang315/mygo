package widget

import "testing"

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
	// The groups come in the order their rows first show.
	if yOf(t, tt, "drupe (2)") >= yOf(t, tt, "berry (1)") {
		t.Errorf("after the sort, the group drupe (Date first) is not above berry")
	}
}

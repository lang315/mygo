package widget

import "testing"

// Groups() is empty as soon as SetGroupBy(nil) was called, also while Details
// shows, which makes no groups: the icon views still read the old ones until
// they build again, so they are not dropped by the setter.
func TestListViewGroupsIsEmptyAfterSetGroupByNilInDetails(t *testing.T) {
	lv := groupedView(ViewLargeIcon)
	tt := tester(t, lv)
	if got := lv.Groups(); len(got) != 3 {
		t.Fatalf("setup: Groups() = %v, want the three groups", got)
	}
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	lv.SetGroupBy(nil)
	tt.Frame()
	tt.Frame()
	if got := lv.Groups(); len(got) != 0 {
		t.Fatalf("Groups() = %v after SetGroupBy(nil) in Details, want none", got)
	}
	// Back to the icons, with no groups, nothing reads the old ones.
	lv.SetView(ViewLargeIcon)
	tt.Frame()
	tt.Frame()
	if got := lv.Groups(); len(got) != 0 || shows(tt, "pome (1)") {
		t.Fatalf("after back to the icons: Groups() = %v, headers shown: %v", got, shows(tt, "pome (1)"))
	}
}

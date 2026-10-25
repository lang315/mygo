package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// kindOf is the group of a fruit: its kind.
func kindOf(v *vitem) string { return v.kind }

// iconModes are the views that show groups.
var iconModes = allModes[1:]

// groupedView makes a list view of the fruits, grouped by kind, in view m.
func groupedView(m ViewMode) *ListViewWidget[vitem, string] {
	return newView().SetView(m).SetGroupBy(kindOf)
}

func yOf(t *testing.T, tt *ui.Tester, text string) float32 {
	t.Helper()
	return find(t, tt, text).Y
}

// The icon views show a header over the rows of each group, in the order the
// groups first appear in the table's, each row under its own.
func TestListViewGroupsShowAHeaderOverEachGroup(t *testing.T) {
	for _, m := range iconModes {
		t.Run(m.name, func(t *testing.T) {
			tt := tester(t, groupedView(m.mode))
			order := []string{"pome (1)", "Apple", "berry (1)", "Banana", "drupe (2)", "Cherry"}
			for i := 1; i < len(order); i++ {
				if yOf(t, tt, order[i-1]) >= yOf(t, tt, order[i]) {
					t.Errorf("%q is not above %q", order[i-1], order[i])
				}
			}
			if yOf(t, tt, "drupe (2)") >= yOf(t, tt, "Date") {
				t.Errorf("the header drupe (2) is not above Date")
			}
		})
	}
}

// SetGroupOrder says the order of the groups; a group it does not name comes
// after those it does, in the order the rows show it; a group without rows is
// not shown.
func TestListViewGroupsFollowTheOrderTheAppGives(t *testing.T) {
	lv := groupedView(ViewList).SetGroupOrder("drupe", "citrus", "pome")
	tt := tester(t, lv)
	order := []string{"drupe (2)", "Cherry", "Date", "pome (1)", "Apple", "berry (1)", "Banana"}
	for i := 1; i < len(order); i++ {
		if yOf(t, tt, order[i-1]) >= yOf(t, tt, order[i]) {
			t.Errorf("%q is not above %q", order[i-1], order[i])
		}
	}
	if shows(tt, "citrus (0)") {
		t.Error("a group without rows is shown")
	}
	want := []ListViewGroup{{"drupe", 2}, {"pome", 1}, {"berry", 1}}
	if got := lv.Groups(); !slices.Equal(got, want) {
		t.Errorf("Groups() = %v, want %v", got, want)
	}
}

// SetGroupBy(nil) shows the rows as a list view without groups does.
func TestListViewSetGroupByNilIsTheViewWithoutGroups(t *testing.T) {
	for _, m := range iconModes {
		t.Run(m.name, func(t *testing.T) {
			plain := tester(t, newView().SetView(m.mode))
			lv := groupedView(m.mode)
			tt := tester(t, lv)
			if !shows(tt, "pome (1)") {
				t.Fatal("the headers are not there to switch off")
			}
			lv.SetGroupBy(nil)
			tt.Frame()
			tt.Frame()
			for _, text := range []string{"Apple", "Banana", "Cherry", "Date"} {
				if got, want := yOf(t, tt, text), yOf(t, plain, text); got != want {
					t.Errorf("%s is at y %v, without groups at %v", text, got, want)
				}
			}
			if shows(tt, "pome (1)") || len(lv.Groups()) != 0 {
				t.Errorf("headers are still shown: %q, groups %v", tt.Texts(), lv.Groups())
			}
		})
	}
}

// Details does not show groups yet.
func TestListViewDetailsShowsNoGroups(t *testing.T) {
	tt := tester(t, groupedView(ViewDetails))
	if shows(tt, "pome (1)") || !shows(tt, "Apple") {
		t.Errorf("Details shows %q", tt.Texts())
	}
}

// The keys pass the headers by, and Shift and Cmd+A leave them out.
func TestListViewGroupsKeysPassTheHeaders(t *testing.T) {
	for _, m := range []ViewMode{ViewList, ViewSmallIcon, ViewLargeIcon} {
		t.Run(m.String(), func(t *testing.T) {
			lv := groupedView(m)
			tt := tester(t, lv)
			click(t, tt, "Apple")
			if got := lv.Selected(); !slices.Equal(got, []string{"a"}) {
				t.Fatalf("click on Apple chose %v", got)
			}
			tt.Key(0, ui.KeyDown)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
				t.Fatalf("Down from Apple chose %v, want Banana, passing the header berry", got)
			}
			tt.Key(0, ui.KeyUp)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"a"}) {
				t.Fatalf("Up from Banana chose %v, want Apple", got)
			}
			tt.Key(0, ui.KeyEnd)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"d"}) {
				t.Fatalf("End chose %v, want Date", got)
			}
			tt.Key(0, ui.KeyHome)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"a"}) {
				t.Fatalf("Home chose %v, want Apple, not a header", got)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Frame()
			if got := lv.Selected(); len(got) != 4 {
				t.Fatalf("Cmd+A chose %v, want the four rows and no header", got)
			}
		})
	}
}

// A click on a header chooses nothing, and a Shift-click over headers chooses
// the rows between and not them.
func TestListViewGroupsHeadersAreNotChosen(t *testing.T) {
	lv := groupedView(ViewList)
	tt := tester(t, lv)
	click(t, tt, "Banana")
	click(t, tt, "drupe (2)")
	if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("a click on a header changed the choice to %v", got)
	}
	click(t, tt, "Apple")
	if err := tt.ClickWith(ui.Shift, "Date"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lv.Selected(); len(got) != 4 {
		t.Fatalf("Shift-click from Apple to Date chose %v, want the four rows", got)
	}
}

// Assistive technology reads a header as "Title (n)", an item of the list that
// is not chosen.
func TestListViewGroupsHeadersAreReadByAssistiveTechnology(t *testing.T) {
	lv := groupedView(ViewSmallIcon)
	tt := tester(t, lv)
	click(t, tt, "Cherry")
	var header, row *ui.TestNode
	nodes := tt.Nodes()
	for i := range nodes {
		if nodes[i].Role != ui.RoleListItem {
			continue // the text inside it has the same label
		}
		switch nodes[i].Label {
		case "drupe (2)":
			header = &nodes[i]
		case "Cherry":
			row = &nodes[i]
		}
	}
	if header == nil || header.Role != ui.RoleListItem || header.Selected || header.Checked {
		t.Fatalf("the header node is %+v, want an item of the list, not chosen", header)
	}
	if row == nil || !row.Checked {
		t.Fatalf("the Cherry node is %+v, want it chosen", row)
	}
}

// What is chosen is the same in every view, kept by key, with groups too.
func TestListViewGroupsKeepTheChoiceWhenTheViewSwitches(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	click(t, tt, "Cherry")
	if err := tt.ClickWith(ui.Shift, "Date"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"c", "d"}) {
		t.Fatalf("in Details: %v", got)
	}
	lv.SetView(ViewLargeIcon)
	tt.Frame()
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"c", "d"}) {
		t.Fatalf("after Details to LargeIcon: %v", got)
	}
	if err := tt.ClickWith(ui.Cmd, "Apple"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"a", "c", "d"}) {
		t.Fatalf("after a Cmd-click on Apple: %v", got)
	}
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"a", "c", "d"}) {
		t.Fatalf("after LargeIcon back to Details: %v", got)
	}
	// The cursor and the pivot cross as well: Down in the icon view goes on
	// from the row the keys were on in Details.
	lv.SetView(ViewList)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("Home in List: %v", got)
	}
}

// A row that moves to another group, one added, one deleted and a search that
// hides rows change the headers; nothing else makes the groups again.
func TestListViewGroupsAreMadeAgainOnlyWhenSomethingChanged(t *testing.T) {
	lv := groupedView(ViewLargeIcon)
	tt := tester(t, lv)
	frames := func() {
		for range 4 {
			tt.Frame()
		}
	}
	frames()
	built := lv.groupBuilds
	if built < 1 {
		t.Fatal("the groups were not made")
	}
	frames()
	if lv.groupBuilds != built {
		t.Fatalf("frames at rest made the groups again: %d, was %d", lv.groupBuilds, built)
	}
	step := func(what string, change func()) {
		t.Helper()
		before := lv.groupBuilds
		change()
		frames()
		if lv.groupBuilds == before {
			t.Fatalf("%s did not make the groups again", what)
		}
		before = lv.groupBuilds
		frames()
		if lv.groupBuilds != before {
			t.Fatalf("frames after %s made them again", what)
		}
	}
	step("adding a row", func() { lv.AddRow(vitem{"e", "Elderberry", "berry"}) })
	if !shows(tt, "berry (2)") {
		t.Errorf("after adding a berry: %q", tt.Texts())
	}
	step("moving a row to another group", func() { lv.UpdateRow("d", func(v *vitem) { v.kind = "pome" }) })
	if !shows(tt, "pome (2)") || !shows(tt, "drupe (1)") {
		t.Errorf("after Date moved to pome: %q", tt.Texts())
	}
	step("deleting a row", func() { lv.DeleteRows("a") })
	if !shows(tt, "pome (1)") {
		t.Errorf("after deleting Apple: %q", tt.Texts())
	}
	step("changing the order", func() { lv.SetGroupOrder("drupe", "berry") })
	step("changing the grouping", func() { lv.SetGroupBy(func(v *vitem) string { return v.name[:1] }) })
	step("a search", func() { lv.Details().Search("Name", "an") })
	if shows(tt, "C (1)") || !shows(tt, "B (1)") {
		t.Errorf("after a search for \"an\": %q", tt.Texts())
	}
}

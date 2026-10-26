package widget

import (
	"fmt"
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

// The icon views show a header over the rows of each group, by title when the
// app gives no order, each row under its own.
func TestListViewGroupsShowAHeaderOverEachGroup(t *testing.T) {
	for _, m := range iconModes {
		t.Run(m.name, func(t *testing.T) {
			tt := tester(t, groupedView(m.mode))
			order := []string{"berry (1)", "Banana", "drupe (2)", "Cherry", "pome (1)", "Apple"}
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
// after those it does, by title; a group without rows is not shown.
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

// The keys pass the headers by, and Shift and Cmd+A leave them out.
func TestListViewGroupsKeysPassTheHeaders(t *testing.T) {
	for _, m := range []ViewMode{ViewList, ViewSmallIcon, ViewLargeIcon} {
		t.Run(m.String(), func(t *testing.T) {
			lv := groupedView(m)
			tt := tester(t, lv)
			// The rows show as Banana, Cherry, Date, Apple, under berry, drupe
			// and pome.
			click(t, tt, "Banana")
			if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
				t.Fatalf("click on Banana chose %v", got)
			}
			tt.Key(0, ui.KeyDown)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"c"}) {
				t.Fatalf("Down from Banana chose %v, want Cherry, passing the header drupe", got)
			}
			tt.Key(0, ui.KeyUp)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
				t.Fatalf("Up from Cherry chose %v, want Banana", got)
			}
			tt.Key(0, ui.KeyEnd)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"a"}) {
				t.Fatalf("End chose %v, want Apple", got)
			}
			tt.Key(0, ui.KeyHome)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
				t.Fatalf("Home chose %v, want Banana, not a header", got)
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
	if err := tt.ClickWith(ui.Shift, "Banana"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lv.Selected(); len(got) != 4 {
		t.Fatalf("Shift-click from Apple to Banana chose %v, want the four rows", got)
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
	// Selected is in the order shown, grouped in every view: berry, drupe, pome.
	if got := lv.Selected(); !slices.Equal(got, []string{"c", "d", "a"}) {
		t.Fatalf("after a Cmd-click on Apple: %v", got)
	}
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"c", "d", "a"}) {
		t.Fatalf("after LargeIcon back to Details: %v", got)
	}
	// The cursor and the pivot cross as well: Down in the icon view goes on
	// from the row the keys were on in Details.
	lv.SetView(ViewList)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Home in List: %v, want Banana, the first row of the first group", got)
	}
}

// The icon views tell the grid where the headers are when they move, and the
// grid does not ask whether each of the lines is one: a few thousand rows move
// from group to group, and ui is asked about the lines in view and a few more.
func TestListViewGroupsTellTheGridWhereTheHeadersAre(t *testing.T) {
	rows := make([]vitem, 4000)
	for i := range rows {
		rows[i] = vitem{fmt.Sprint("k", i), fmt.Sprint("Row ", i), fmt.Sprint("g", i%8)}
	}
	lv := groupedView(ViewList).SetRows(rows)
	tt := tester(t, lv)
	tt.Frame()
	lv.t.asks = 0
	// A row moves to the next group: the headers move, the lines stay.
	lv.UpdateRow("k0", func(v *vitem) { v.kind = "g1" })
	tt.Frame()
	tt.Frame()
	if lv.t.asks > len(rows)/10 {
		t.Fatalf("ui asked whether a line is a header %d times for %d rows", lv.t.asks, len(rows))
	}
	if !shows(tt, "g0 (499)") || shows(tt, "g0 (500)") {
		t.Fatalf("after k0 moved to g1: %q", tt.Texts())
	}
}

// A header moves when a row changes group and the number of lines stays: the
// icon views, told where the headers are, lay out the new places.
func TestListViewGroupsHeaderMovesWhenTheLinesStay(t *testing.T) {
	for _, m := range iconModes {
		t.Run(m.name, func(t *testing.T) {
			lv := groupedView(m.mode)
			tt := tester(t, lv)
			// berry, Banana, drupe, Cherry, Date, pome, Apple; Cherry becomes a berry.
			lv.UpdateRow("c", func(v *vitem) { v.kind = "berry" })
			tt.Frame()
			tt.Frame()
			if m.mode == ViewList {
				above(t, tt, "berry (2)", "Banana", "Cherry", "drupe (1)", "Date", "pome (1)", "Apple")
			} else {
				// Several columns: the header of drupe starts a row under Cherry's.
				if yOf(t, tt, "drupe (1)") <= yOf(t, tt, "Cherry") || yOf(t, tt, "Date") <= yOf(t, tt, "drupe (1)") {
					t.Errorf("drupe (1) is at y=%v, Cherry at y=%v, Date at y=%v", yOf(t, tt, "drupe (1)"), yOf(t, tt, "Cherry"), yOf(t, tt, "Date"))
				}
			}
		})
	}
}

// The app's function is asked once for each row changed in place, for all of
// them only when the rows are placed again (rows added or deleted, a new order or
// grouping); a frame at rest and a search ask it nothing. (A table places the
// rows again, whatever changed, when more than an eighth of them did: hence
// forty rows.)
func TestListViewGroupsAskTheFunctionOnlyForWhatChanged(t *testing.T) {
	kinds := []string{"pome", "berry", "drupe", "nut"}
	rows := make([]vitem, 40)
	for i := range rows {
		rows[i] = vitem{fmt.Sprint("k", i), fmt.Sprint("Row ", i), kinds[i%4]}
	}
	calls := 0
	lv := newView().SetRows(rows).SetView(ViewLargeIcon).SetGroupBy(func(v *vitem) string { calls++; return v.kind })
	tt := tester(t, lv)
	frames := func() {
		for range 4 {
			tt.Frame()
		}
	}
	frames()
	if calls != len(rows) {
		t.Fatalf("the function was asked %d times for %d rows", calls, len(rows))
	}
	step := func(what string, want int, change func()) {
		t.Helper()
		calls = 0
		change()
		frames()
		if calls != want {
			t.Fatalf("%s asked the function %d times, want %d", what, calls, want)
		}
		calls = 0
		frames()
		if calls != 0 {
			t.Fatalf("frames after %s asked the function %d times", what, calls)
		}
	}
	has := func(group string, n int) bool {
		for _, g := range lv.Groups() {
			if g.Title == group {
				return g.Count == n
			}
		}
		return false
	}
	// A row added gives the table a new slice, so it places all of them.
	step("adding a row", 41, func() { lv.AddRow(vitem{"new", "Row new", "berry"}) })
	if !has("berry", 11) {
		t.Errorf("after adding a berry: %v", lv.Groups())
	}
	step("moving a row to another group", 1, func() { lv.UpdateRow("k0", func(v *vitem) { v.kind = "nut" }) })
	if !has("pome", 9) || !has("nut", 11) {
		t.Errorf("after k0 moved to nut: %v", lv.Groups())
	}
	step("deleting a row", 40, func() { lv.DeleteRows("k1") })
	if !has("berry", 10) {
		t.Errorf("after deleting k1: %v", lv.Groups())
	}
	step("changing the order", 40, func() { lv.SetGroupOrder("drupe", "berry") })
	step("changing the grouping", 40, func() {
		lv.SetGroupBy(func(v *vitem) string { calls++; return v.kind[:1] })
	})
	step("a search", 0, func() { lv.Details().Search("Name", "Row 3") })
	if len(lv.Groups()) == 0 {
		t.Errorf("after a search for \"Row 3\": no group")
	}
}

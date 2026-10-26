package widget

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// above checks that the texts show in the order given, from the top.
func above(t *testing.T, tt *ui.Tester, order ...string) {
	t.Helper()
	for i := 1; i < len(order); i++ {
		if yOf(t, tt, order[i-1]) >= yOf(t, tt, order[i]) {
			t.Errorf("%q is not above %q", order[i-1], order[i])
		}
	}
}

// Details shows a header over the rows of each group, over the whole width,
// the groups in the order the icon views show them.
func TestListViewDetailsShowsAHeaderOverEachGroup(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	above(t, tt, "berry (1)", "Banana", "drupe (2)", "Cherry", "Date", "pome (1)", "Apple")
	want := []ListViewGroup{{"berry", 1}, {"drupe", 2}, {"pome", 1}}
	if got := lv.Groups(); !slices.Equal(got, want) {
		t.Errorf("Groups() = %v, want %v", got, want)
	}
}

// Without SetGroupBy, or after SetGroupBy(nil), Details shows the rows as it
// always did.
func TestListViewDetailsWithoutGroupsShowsNoHeader(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	lv.SetGroupBy(nil)
	tt.Frame()
	tt.Frame()
	if shows(tt, "pome (1)") {
		t.Errorf("Details shows a header after SetGroupBy(nil): %q", tt.Texts())
	}
	above(t, tt, "Apple", "Banana", "Cherry", "Date")
	if got := lv.Groups(); len(got) != 0 {
		t.Errorf("Groups() = %v", got)
	}
}

// The keys pass the headers by, and Shift and Cmd+A leave them out.
func TestListViewDetailsKeysPassTheHeaders(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	click(t, tt, "Banana")
	if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("click on Banana chose %v", got)
	}
	steps := []struct {
		mods ui.Modifiers
		key  ui.Key
		want []string
		what string
	}{
		{0, ui.KeyDown, []string{"c"}, "Down from Banana: Cherry, passing the header drupe"},
		{0, ui.KeyUp, []string{"b"}, "Up from Cherry: Banana"},
		{0, ui.KeyEnd, []string{"a"}, "End: Apple"},
		{0, ui.KeyHome, []string{"b"}, "Home: Banana, not a header"},
		{ui.Cmd, ui.KeyA, []string{"b", "c", "d", "a"}, "Cmd+A: the four rows, no header"},
	}
	for _, s := range steps {
		tt.Key(s.mods, s.key)
		tt.Frame()
		if got := lv.Selected(); !slices.Equal(got, s.want) {
			t.Fatalf("%s: chose %v, want %v", s.what, got, s.want)
		}
	}
}

// A click on a header chooses nothing, and a Shift-click over headers chooses
// the rows between and not them.
func TestListViewDetailsHeadersAreNotChosen(t *testing.T) {
	lv := groupedView(ViewDetails)
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

// Sorting by a column sorts the rows inside the groups, which stay where they
// are.
func TestListViewDetailsGroupsDoNotMoveWithTheSort(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	click(t, tt, "Name") // ascending
	click(t, tt, "Name") // descending
	above(t, tt, "berry (1)", "Banana", "drupe (2)", "Date", "Cherry", "pome (1)", "Apple")
}

// A group whose rows a search hides shows no header, and the count is of the
// rows shown.
func TestListViewDetailsSearchHidesEmptyGroups(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	lv.Details().Search("Name", "an")
	tt.Frame()
	tt.Frame()
	if !shows(tt, "berry (1)") || !shows(tt, "Banana") || shows(tt, "drupe (2)") || shows(tt, "pome (1)") {
		t.Fatalf("after a search for \"an\": %q", tt.Texts())
	}
	want := []ListViewGroup{{"berry", 1}}
	if got := lv.Groups(); !slices.Equal(got, want) {
		t.Errorf("Groups() = %v, want %v", got, want)
	}
	lv.Details().Search("Name", "")
	tt.Frame()
	tt.Frame()
	if !shows(tt, "drupe (2)") || !shows(tt, "pome (1)") {
		t.Fatalf("after the search is cleared: %q", tt.Texts())
	}
}

// A row that changes group in place moves to it, and the counts follow.
func TestListViewDetailsFollowsARowChangedInPlace(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	lv.UpdateRow("a", func(v *vitem) { v.kind = "berry" })
	tt.Frame()
	tt.Frame()
	if !shows(tt, "berry (2)") || shows(tt, "pome (1)") || shows(tt, "berry (1)") {
		t.Fatalf("after Apple became a berry: %q", tt.Texts())
	}
	above(t, tt, "berry (2)", "Apple", "Banana", "drupe (2)")
}

// Copy, the context menu and Enter work on the rows, not on the places of the
// table, which count the headers.
func TestListViewDetailsCopyMenuAndActivateActOnTheRows(t *testing.T) {
	activated := make(chan string, 4)
	menuFor := make(chan []string, 4)
	lv := groupedView(ViewDetails).
		CopyFunc(func(ks []string) string { return strings.Join(ks, ",") }).
		ContextMenuFunc(func(ks []string) []MenuItem {
			menuFor <- ks
			return []MenuItem{{Text: "Open"}}
		}).
		OnActivate(func(k string) { activated <- k })
	tt := tester(t, lv)
	click(t, tt, "Banana")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "b" {
		t.Fatalf("Copy of Banana copied %q, want \"b\"", got)
	}
	if err := tt.ClickWith(ui.Cmd, "Date"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Command("copy")
	if got := tt.Clipboard(); got != "b,d" {
		t.Fatalf("Copy of Banana and Date copied %q, want \"b,d\"", got)
	}
	click(t, tt, "Cherry")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	select {
	case k := <-activated:
		if k != "c" {
			t.Fatalf("Enter on Cherry activated %q", k)
		}
	case <-time.After(time.Minute):
		t.Fatal("Enter did not activate")
	}
	if err := tt.RightClick("Apple"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	<-posted()
	select {
	case ks := <-menuFor:
		if strings.Join(ks, ",") != "a" {
			t.Fatalf("the context menu of Apple was built for %v, want [a]", ks)
		}
	case <-time.After(time.Minute):
		t.Fatal("the context menu was not built")
	}
}

// The index column numbers the rows, not the headers.
func TestListViewDetailsIndexColumnSkipsTheHeaders(t *testing.T) {
	lv := groupedView(ViewDetails)
	lv.Details().IndexColumn()
	tt := tester(t, lv)
	// Banana is the first row, under the header berry: it is 1, not 2.
	if !shows(tt, "berry (1)") {
		t.Fatalf("setup: no header in %q", tt.Texts())
	}
	for _, r := range []struct{ name, number string }{{"Banana", "1"}, {"Cherry", "2"}, {"Date", "3"}, {"Apple", "4"}} {
		row, number := find(t, tt, r.name), find(t, tt, r.number)
		if number.Y < row.Y || number.Y >= row.Y+row.H {
			t.Errorf("%s is numbered %s at y=%v, off its row at y=%v..%v", r.name, r.number, number.Y, row.Y, row.Y+row.H)
		}
	}
	if shows(tt, "5") || shows(tt, "6") || shows(tt, "7") {
		t.Errorf("the headers are numbered: %q", tt.Texts())
	}
}

// With check boxes, the boxes check the rows, and the header's check-all
// checks them all and no header.
func TestListViewDetailsCheckAllSkipsTheHeaders(t *testing.T) {
	lv := groupedView(ViewDetails).CheckBoxes()
	tt := tester(t, lv)
	if err := tt.Click("Check Banana"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lv.Checked(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("a click on the box of Banana checked %v", got)
	}
	if err := tt.Click("Check all"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lv.Checked(); len(got) != 4 {
		t.Fatalf("Check all checked %v, want the four rows", got)
	}
}

// With groups, "Check all" and then "Check all" again report each row flipped,
// and no header, as they do without groups.
func TestListViewDetailsCheckAllFiresOnCheckForEachRow(t *testing.T) {
	lv := groupedView(ViewDetails).CheckBoxes()
	ev := watched(lv)
	tt := tester(t, lv)
	click(t, tt, "Check all")
	got := ev.drain()
	slices.Sort(got)
	if want := []string{"check a true", "check b true", "check c true", "check d true"}; !slices.Equal(got, want) {
		t.Fatalf("Check all: events %v, want %v", got, want)
	}
	click(t, tt, "Check all")
	got = ev.drain()
	slices.Sort(got)
	if want := []string{"check a false", "check b false", "check c false", "check d false"}; !slices.Equal(got, want) {
		t.Fatalf("Check all again: events %v, want %v", got, want)
	}
	if n := len(lv.Checked()); n != 0 {
		t.Fatalf("Checked() has %d keys after unchecking all", n)
	}
}

// Assistive technology reads a header as "Title (n)", which is not chosen,
// and the rows around it as rows.
func TestListViewDetailsHeadersAreReadByAssistiveTechnology(t *testing.T) {
	lv := groupedView(ViewDetails)
	tt := tester(t, lv)
	click(t, tt, "Cherry")
	var header, row *ui.TestNode
	nodes := tt.Nodes()
	for i := range nodes {
		switch {
		case nodes[i].Label == "drupe (2)" && nodes[i].Role == ui.RoleRow:
			header = &nodes[i]
		case nodes[i].Label == "Cherry" && row == nil && nodes[i].Role == ui.RoleRow:
			row = &nodes[i]
		}
	}
	if header == nil || header.Selected || header.Checked {
		t.Fatalf("the header node is %+v, want one named \"drupe (2)\", not chosen", header)
	}
	if row == nil || !row.Checked {
		t.Fatalf("the Cherry node is %+v, want it chosen", row)
	}
}

// The groups come in the order the app gives, in Details too, the others
// after them by title.
func TestListViewDetailsFollowsTheOrderTheAppGives(t *testing.T) {
	lv := groupedView(ViewDetails).SetGroupOrder("pome", "drupe")
	tt := tester(t, lv)
	above(t, tt, "pome (1)", "Apple", "drupe (2)", "Cherry", "Date", "berry (1)", "Banana")
	want := []ListViewGroup{{"pome", 1}, {"drupe", 2}, {"berry", 1}}
	if got := lv.Groups(); !slices.Equal(got, want) {
		t.Errorf("Groups() = %v, want %v", got, want)
	}
}

// With few rows chosen among many, the table hands ui their lines, which count
// the headers, not their positions, without going through every row.
func TestListViewDetailsChosenRowsAreLines(t *testing.T) {
	rows := make([]vitem, 40)
	for i := range rows {
		rows[i] = vitem{fmt.Sprint("k", i), fmt.Sprintf("Item %02d", i), fmt.Sprint("kind ", i%4)}
	}
	lv := groupedView(ViewDetails).SetRows(rows)
	tt := tester(t, lv)
	lv.SetSelected("k5", "k22")
	tt.Frame()
	tt.Frame()
	got, ok := lv.Details().chosenRows()
	if !ok || len(got) != 2 {
		t.Fatalf("chosenRows() = %v, %v, want two lines", got, ok)
	}
	for _, line := range got {
		if k := lv.Details().keyAt(line); k != "k5" && k != "k22" {
			t.Errorf("line %d is the row %v, not one chosen", line, k)
		}
	}
	// k5 is the second row of the second group of ten: 10 rows + 1 header + 1
	// header + 1 = line 13; k22 the sixth of the third: 20 rows + 3 headers + 5 =
	// line 28. Positions, which count no header, would be 5 and 22.
	if !slices.Equal(got, []int{13, 28}) {
		t.Errorf("chosenRows() = %v, want the lines [13 28]", got)
	}
}

// The cursor crosses from an icon view to Details and back as a row, not as a
// number: the icon views count the headers apart from the table.
func TestListViewDetailsKeepsTheCursorAcrossViews(t *testing.T) {
	lv := groupedView(ViewLargeIcon)
	tt := tester(t, lv)
	click(t, tt, "Date")
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"c"}) {
		t.Fatalf("Up in Details from Date, chosen in the icons: %v, want Cherry", got)
	}
	lv.SetView(ViewLargeIcon)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Up in the icons from Cherry, chosen in Details: %v, want Banana", got)
	}
}

// ui asks for the keys and labels of the frame the user saw, whatever a
// goroutine changed since: a row that moved to another group, and the headers
// with it, does not make a line mean another row.
func TestListViewDetailsLinesStayThoseOfTheFrameSeen(t *testing.T) {
	lv := groupedView(ViewDetails)
	tester(t, lv) // the lines: berry, Banana, drupe, Cherry, Date, pome, Apple
	lv.UpdateRow("a", func(v *vitem) { v.kind = "berry" })
	lv.Selected() // brings the table up to date outside a frame: the groups move on
	t2 := lv.Details()
	if got := t2.keyAt(4); got != "d" {
		t.Errorf("the key of line 4 is %v, want d (Date) as the frame showed it", got)
	}
	if got := t2.labelAt(5); got != "pome (1)" {
		t.Errorf("the label of line 5 is %q, want the header pome (1) as the frame showed it", got)
	}
	if got := t2.keyAt(2); got != (groupKey{"drupe"}) {
		t.Errorf("the key of line 2 is %v, want the header drupe", got)
	}
}

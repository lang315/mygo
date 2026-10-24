package widget

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// vitem is an item of the test list views.
type vitem struct{ id, name, kind string }

var fruits = []vitem{
	{"a", "Apple", "pome"}, {"b", "Banana", "berry"}, {"c", "Cherry", "drupe"}, {"d", "Date", "drupe"},
}

func newView() *ListViewWidget[vitem, string] {
	return ListView(func(v *vitem) string { return v.id },
		Column[vitem]{Title: "Name", Text: func(v *vitem) string { return v.name }, Width: 200, Sortable: true, Filterable: true},
		Column[vitem]{Title: "Kind", Text: func(v *vitem) string { return v.kind }, Width: 100}).
		SetRows(fruits)
}

var allModes = []struct {
	name string
	mode ViewMode
}{{"Details", ViewDetails}, {"LargeIcon", ViewLargeIcon}, {"SmallIcon", ViewSmallIcon}, {"List", ViewList}}

// seen collects what a list view reported, as short strings.
type seen chan string

func (e seen) drain() (got []string) {
	<-posted()
	for {
		select {
		case s := <-e:
			got = append(got, s)
		default:
			return
		}
	}
}

func watched(lv *ListViewWidget[vitem, string]) seen {
	ev := make(seen, 64)
	lv.OnSelect(func(ks []string) { ev <- fmt.Sprint("select ", ks) }).
		OnCheck(func(k string, on bool) { ev <- fmt.Sprint("check ", k, " ", on) }).
		OnActivate(func(k string) { ev <- "activate " + k })
	return ev
}

func TestListViewDefaultsToDetailsAndShowsTheColumnsThere(t *testing.T) {
	lv := newView()
	if lv.View() != ViewDetails {
		t.Fatalf("the zero view is %v, want Details", lv.View())
	}
	tt := tester(t, lv)
	for _, s := range []string{"Name", "Kind", "Apple", "pome", "Date"} {
		if !shows(tt, s) {
			t.Fatalf("Details shows %q, no %q", tt.Texts(), s)
		}
	}
}

func TestListViewEveryModeShowsTheItemsAndTheOthersDoNotShowColumns(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			tt := tester(t, lv)
			for _, s := range []string{"Apple", "Banana", "Cherry", "Date"} {
				if !shows(tt, s) {
					t.Fatalf("%s shows %q, no %q", m.name, tt.Texts(), s)
				}
			}
			if details := m.mode == ViewDetails; shows(tt, "Kind") != details || shows(tt, "pome") != details {
				t.Fatalf("%s shows %q: the columns belong to ViewDetails only", m.name, tt.Texts())
			}
			if lv.View() != m.mode {
				t.Fatalf("View %v, want %v", lv.View(), m.mode)
			}
			a, b := find(t, tt, "Apple"), find(t, tt, "Banana")
			if !(a.Y < b.Y || a.X < b.X) {
				t.Fatalf("Apple at %v is not before Banana at %v", a, b)
			}
		})
	}
}

func TestListViewClickChoosesByKeyInEveryMode(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			ev := watched(lv)
			tt := tester(t, lv)
			click(t, tt, "Banana")
			if got := lv.Selected(); !slices.Equal(got, []string{"b"}) {
				t.Fatalf("Selected %v", got)
			}
			if got := ev.drain(); !slices.Equal(got, []string{"select [b]"}) {
				t.Fatalf("seen %v, want one OnSelect for Banana", got)
			}
		})
	}
}

func TestListViewCmdAndShiftChooseSeveralInEveryMode(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			tt := tester(t, lv)
			click(t, tt, "Apple")
			if err := tt.ClickWith(ui.Cmd, "Cherry"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"a", "c"}) {
				t.Fatalf("Cmd-click: Selected %v, want Apple and Cherry", got)
			}
			if lv.CountSelected() != 2 {
				t.Fatalf("CountSelected %d", lv.CountSelected())
			}
			click(t, tt, "Banana")
			if err := tt.ClickWith(ui.Shift, "Date"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"b", "c", "d"}) {
				t.Fatalf("Shift-click: Selected %v, want Banana to Date", got)
			}
		})
	}
}

func TestListViewCheckBoxesCheckTheItemsChosenWithSpaceInEveryMode(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().CheckBoxes().SetView(m.mode)
			ev := watched(lv)
			tt := tester(t, lv)
			click(t, tt, "Cherry")
			ev.drain()
			tt.Key(0, ui.KeySpace)
			tt.Frame()
			if got := ev.drain(); !slices.Equal(got, []string{"check c true"}) {
				t.Fatalf("Space: seen %v", got)
			}
			if got := lv.Checked(); !slices.Equal(got, []string{"c"}) || !lv.IsChecked("c") {
				t.Fatalf("Checked %v", got)
			}
			if len(lv.Selected()) != 1 {
				t.Fatalf("checking changed the choice: %v", lv.Selected())
			}
			tt.Key(0, ui.KeySpace)
			tt.Frame()
			if got := ev.drain(); !slices.Equal(got, []string{"check c false"}) {
				t.Fatalf("Space again: seen %v", got)
			}
		})
	}
}

func TestListViewSettersFireNoEvents(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().CheckBoxes().SetView(m.mode)
			ev := watched(lv)
			tt := tester(t, lv)
			lv.SetSelected("b", "c")
			lv.SetChecked([]string{"a", "d"}, true)
			lv.UpdateRow("b", func(v *vitem) { v.name = "Blueberry" })
			lv.SetView(ViewLargeIcon).SetView(m.mode)
			tt.Frame()
			tt.Frame()
			if got := ev.drain(); len(got) != 0 {
				t.Fatalf("a program's change fired %v", got)
			}
			if got := lv.Selected(); !slices.Equal(got, []string{"b", "c"}) {
				t.Fatalf("Selected %v", got)
			}
			if got := lv.Checked(); !slices.Equal(got, []string{"a", "d"}) {
				t.Fatalf("Checked %v", got)
			}
		})
	}
}

// The choice and the checks are the list view's, by key: switching the view
// at run time keeps them, and says nothing.
func TestListViewSwitchingTheViewKeepsTheChoiceAndTheChecksByKeyAndFiresNothing(t *testing.T) {
	lv := newView().CheckBoxes()
	ev := watched(lv)
	tt := tester(t, lv)
	click(t, tt, "Apple")
	if err := tt.ClickWith(ui.Cmd, "Cherry"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeySpace) // checks both
	tt.Frame()
	ev.drain()
	want := lv.Selected()
	wantChecked := lv.Checked()
	if !slices.Equal(want, []string{"a", "c"}) || !slices.Equal(wantChecked, []string{"a", "c"}) {
		t.Fatalf("set up: Selected %v, Checked %v", want, wantChecked)
	}
	for _, m := range []ViewMode{ViewLargeIcon, ViewSmallIcon, ViewList, ViewDetails, ViewList, ViewLargeIcon} {
		lv.SetView(m)
		tt.Frame()
		tt.Frame()
		if got := lv.Selected(); !slices.Equal(got, want) {
			t.Fatalf("in %v Selected %v, want %v", m, got, want)
		}
		if got := lv.Checked(); !slices.Equal(got, wantChecked) {
			t.Fatalf("in %v Checked %v, want %v", m, got, wantChecked)
		}
	}
	if got := ev.drain(); len(got) != 0 {
		t.Fatalf("switching fired %v", got)
	}
	// And the new view takes the user's next change from there.
	click(t, tt, "Date")
	if got := lv.Selected(); !slices.Equal(got, []string{"d"}) {
		t.Fatalf("after switching, a click chose %v", got)
	}
	if got := ev.drain(); !slices.Equal(got, []string{"select [d]"}) {
		t.Fatalf("seen %v", got)
	}
}

// Sort and filter belong to Details() and drive every view: the order of the
// icons follows the table's, and a change keeps the choice and the checks by
// key, with the cursor on the item chosen.
func TestListViewSortAndFilterInDetailsDriveTheIconsAndKeepTheChoiceByKey(t *testing.T) {
	lv := newView().CheckBoxes()
	tt := tester(t, lv)
	click(t, tt, "Name") // sorts by name
	click(t, tt, "Name") // descending
	lv.SetView(ViewLargeIcon)
	tt.Frame()
	if d, a := find(t, tt, "Date"), find(t, tt, "Apple"); !(d.Y < a.Y || d.X < a.X) {
		t.Fatalf("Date at %v is not before Apple at %v: the icons follow the table's descending sort", d, a)
	}
	click(t, tt, "Cherry")
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	lv.Details().ClearSort() // ascending order: Apple Banana Cherry Date
	tt.Frame()
	tt.Frame()
	if a, d := find(t, tt, "Apple"), find(t, tt, "Date"); !(a.Y < d.Y || a.X < d.X) {
		t.Fatalf("Apple at %v is not before Date at %v after ClearSort", a, d)
	}
	if got := lv.Selected(); !slices.Equal(got, []string{"c"}) || !slices.Equal(lv.Checked(), []string{"c"}) {
		t.Fatalf("after the sort changed: Selected %v, Checked %v", got, lv.Checked())
	}
	tt.Key(0, ui.KeyRight) // the cursor followed Cherry: next is Date
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"d"}) {
		t.Fatalf("Right after the sort changed chose %v, want Date, the item after Cherry", got)
	}
	// A filter: Banana goes, and Cherry moves up a place.
	click(t, tt, "Cherry")
	lv.Details().Search("Name", "e") // Apple, Cherry, Date
	tt.Frame()
	tt.Frame()
	if shows(tt, "Banana") {
		t.Fatalf("the filter left Banana: %q", tt.Texts())
	}
	if got := lv.Selected(); !slices.Equal(got, []string{"c"}) || !lv.IsChecked("c") {
		t.Fatalf("after the filter: Selected %v, IsChecked(c) %v", got, lv.IsChecked("c"))
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"d"}) {
		t.Fatalf("Right after the filter chose %v, want Date", got)
	}
}

// The keys of an icon view move from the row a setter chose.
func TestListViewSetSelectedIsWhereTheKeysMoveFromInTheIconViews(t *testing.T) {
	lv := newView().SetView(ViewLargeIcon)
	tt := tester(t, lv)
	click(t, tt, "Date")
	lv.SetSelected("b")
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if got := lv.Selected(); !slices.Equal(got, []string{"c"}) {
		t.Fatalf("Right after SetSelected(b) chose %v, want Cherry", got)
	}
}

// The table is not built in an icon view, but its changes redraw the window
// that shows the list view.
func TestListViewRowChangesRedrawTheWindowInTheIconViews(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			_, w := showFrame(t, lv)
			redrawn := recordInvalidations(t)
			lv.AddRow(vitem{"e", "Elderberry", "berry"})
			wantRedrawn(t, redrawn(), w)
			lv.SetChecked([]string{"a"}, true)
			wantRedrawn(t, redrawn(), w)
			lv.SetSelected("b")
			wantRedrawn(t, redrawn(), w)
		})
	}
}

func TestListViewKeyboardInTheIconViews(t *testing.T) {
	for _, m := range []ViewMode{ViewLargeIcon, ViewSmallIcon, ViewList} {
		t.Run(fmt.Sprint(m), func(t *testing.T) {
			lv := newView().SetView(m)
			tt := tester(t, lv)
			click(t, tt, "Banana")
			for _, c := range []struct {
				key  ui.Key
				want string
			}{{ui.KeyRight, "c"}, {ui.KeyLeft, "b"}, {ui.KeyEnd, "d"}, {ui.KeyHome, "a"}} {
				if m == ViewList && (c.key == ui.KeyRight || c.key == ui.KeyLeft) {
					c.key = map[ui.Key]ui.Key{ui.KeyRight: ui.KeyDown, ui.KeyLeft: ui.KeyUp}[c.key]
				}
				tt.Key(0, c.key)
				tt.Frame()
				if got := lv.Selected(); !slices.Equal(got, []string{c.want}) {
					t.Fatalf("%v: Selected %v, want %q", c.key, got, c.want)
				}
			}
			tt.Key(0, ui.KeyD)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"d"}) {
				t.Fatalf("typing d: Selected %v, want Date", got)
			}
		})
	}
}

func TestListViewEnterActivatesInEveryMode(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			ev := watched(lv)
			tt := tester(t, lv)
			click(t, tt, "Cherry")
			ev.drain()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if got := ev.drain(); !slices.Equal(got, []string{"activate c"}) {
				t.Fatalf("Enter: seen %v", got)
			}
		})
	}
}

func TestListViewCopyFuncServesTheIconViewsToo(t *testing.T) {
	lv := newView().CopyFunc(func(ks []string) string { return fmt.Sprint(ks) }).SetView(ViewLargeIcon)
	tt := tester(t, lv)
	click(t, tt, "Banana")
	if err := tt.ClickWith(ui.Cmd, "Date"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != "[b d]" {
		t.Fatalf("Cmd+C copied %q", got)
	}
}

func TestListViewRemovedRowsLeaveTheChoiceAndTheChecksInEveryView(t *testing.T) {
	lv := newView().CheckBoxes().SetView(ViewLargeIcon)
	ev := watched(lv)
	tt := tester(t, lv)
	click(t, tt, "Banana")
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	ev.drain()
	lv.DeleteRows("b")
	tt.Frame()
	tt.Frame()
	if shows(tt, "Banana") || len(lv.Selected()) != 0 || len(lv.Checked()) != 0 || lv.CountSelected() != 0 {
		t.Fatalf("shows %q, Selected %v, Checked %v", tt.Texts(), lv.Selected(), lv.Checked())
	}
	lv.AddRows([]vitem{{"b", "Banana", "berry"}})
	lv.SetView(ViewDetails)
	tt.Frame()
	if lv.IsChecked("b") || len(lv.Selected()) != 0 {
		t.Fatalf("a row added again came back chosen %v or checked %v", lv.Selected(), lv.IsChecked("b"))
	}
	if got := ev.drain(); len(got) != 0 {
		t.Fatalf("removing rows fired %v", got)
	}
}

func TestListViewDetailsIsTheTableUnderneath(t *testing.T) {
	lv := newView()
	if lv.Details() == nil {
		t.Fatal("Details() is nil")
	}
	lv.Details().Search("Name", "an")
	tt := tester(t, lv)
	if !shows(tt, "Banana") || shows(tt, "Apple") {
		t.Fatalf("a Details() search shows %q", tt.Texts())
	}
	lv.SetView(ViewSmallIcon)
	tt.Frame()
	if !shows(tt, "Banana") || shows(tt, "Apple") {
		t.Fatalf("the search does not drive the icons: %q", tt.Texts())
	}
}

func TestListViewImageShowsInTheIconViews(t *testing.T) {
	asked := map[string]bool{}
	lv := newView().SetImage(func(v *vitem) *ui.Bitmap { asked[v.id] = true; return nil })
	for _, m := range allModes {
		lv.SetView(m.mode)
		tester(t, lv)
	}
	if !asked["a"] || !asked["d"] {
		t.Fatalf("SetImage's function was asked for %v", asked)
	}
}

// clickCostOf returns the objects and bytes a click on an item allocates in a
// list view of n rows shown as mode.
func clickCostOf(t *testing.T, mode ViewMode, n int) (allocs float64, bytes int64) {
	t.Helper()
	rows := make([]vitem, n)
	for i := range rows {
		rows[i] = vitem{fmt.Sprint("k", i), fmt.Sprint("Item ", i), "kind"}
	}
	lv := newView().SetRows(rows).SetView(mode)
	tt := tester(t, lv)
	click(t, tt, "Item 1")
	pick := func(name string) func() {
		return func() {
			if err := tt.Click(name); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
		}
	}
	a, b := pick("Item 2"), pick("Item 3")
	a()
	b()
	var k int
	allocs = testing.AllocsPerRun(20, func() {
		if k++; k%2 == 0 {
			a()
		} else {
			b()
		}
	})
	res := testing.Benchmark(func(bb *testing.B) {
		for i := 0; i < bb.N; i++ {
			if i%2 == 0 {
				a()
			} else {
				b()
			}
		}
	})
	return allocs, res.AllocedBytesPerOp()
}

// A click costs what the items in view cost, not the rows: in ViewDetails and in
// ViewLargeIcon, 200,000 rows against 2,000.
func TestListViewClickAllocationsDoNotGrowWithTheRows(t *testing.T) {
	for _, m := range []ViewMode{ViewDetails, ViewLargeIcon} {
		t.Run(fmt.Sprint(m), func(t *testing.T) {
			smallAllocs, smallBytes := clickCostOf(t, m, 2000)
			bigAllocs, bigBytes := clickCostOf(t, m, 200_000)
			t.Logf("%v: a click on 2000 rows: %.0f allocs, %d B/op; on 200000: %.0f allocs, %d B/op", m, smallAllocs, smallBytes, bigAllocs, bigBytes)
			if bigAllocs > smallAllocs*1.1+10 {
				t.Errorf("a click allocated %.0f objects on 200000 rows, %.0f on 2000: want the same", bigAllocs, smallAllocs)
			}
			if bigBytes > smallBytes*11/10+1<<10 {
				t.Errorf("a click allocated %d bytes on 200000 rows, %d on 2000: want the same", bigBytes, smallBytes)
			}
			if bigBytes > 512<<10 {
				t.Errorf("a click allocated %d bytes: want far less than a clone of the rows (MB)", bigBytes)
			}
		})
	}
}

package widget

import (
	"fmt"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

// An empty table that follows its end follows the rows that come.
func TestTableFollowEndOfAnEmptyTable(t *testing.T) {
	tb := testTable().FollowEnd(true)
	tt := tester(t, tb)
	tb.AddRows(followRows(1, 200))
	tt.Frame()
	if !inView(tt, "r0200") {
		t.Fatalf("empty then 200 rows: r0200 not in view")
	}
}

// Rows trickling in one by one, from a table that fits, are followed.
func TestTableFollowEndRowsOneByOne(t *testing.T) {
	tb := testTable(followRows(1, 2)...).FollowEnd(true)
	tt := tester(t, tb)
	for i := 3; i <= 80; i++ {
		tb.AddRows(followRows(i, 1))
		tt.Frame()
		if s := fmt.Sprintf("r%04d", i); !inView(tt, s) {
			t.Fatalf("row %d not followed", i)
		}
	}
}

func TestTableScrollToEndBeforeTheFirstFrame(t *testing.T) {
	tb := testTable(followRows(1, 200)...).ScrollToEnd()
	tt := tester(t, tb)
	tt.Frame()
	if !inView(tt, "r0200") {
		t.Fatalf("ScrollToEnd before the first frame: r0200 not in view")
	}
}

// The user at the end, the app turns FollowEnd on and rows come before the
// next frame.
func TestTableFollowEndOnAtTheEndWithRowsInTheSameFrame(t *testing.T) {
	tb := testTable(followRows(1, 200)...)
	tt := tester(t, tb)
	tt.Scroll(300, 300, 0, 100000)
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0200") {
		t.Fatal("not at the end")
	}
	tb.FollowEnd(true)
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatalf("at the end, FollowEnd(true) with rows in the same frame does not follow")
	}
}

// FollowEnd(true) on a table already shown at its top: does it go to the end?
func TestFollowEndOnATableShownAtItsTop(t *testing.T) {
	tb := testTable(followRows(1, 200)...)
	tt := tester(t, tb)
	tb.FollowEnd(true)
	tt.Frame()
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if !inView(tt, "r0001") || inView(tt, "r0250") {
		t.Fatalf("FollowEnd(true) on a table shown at its top moved it: r0001 %v, r0250 %v", inView(tt, "r0001"), inView(tt, "r0250"))
	}
	tb.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatal("ScrollToEnd after FollowEnd(true) did not show the end")
	}
	tb.AddRows(followRows(251, 50))
	tt.Frame()
	if !inView(tt, "r0300") {
		t.Fatal("the table does not follow after ScrollToEnd")
	}
}

func TestTableFollowEndKeepsTheSelection(t *testing.T) {
	tb := testTable(followRows(1, 200)...).FollowEnd(true)
	tt := tester(t, tb)
	click(t, tt, "r0195")
	if got := tb.Selected(); !slices.Equal(got, []int{195}) {
		t.Fatalf("selected %v", got)
	}
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if got := tb.Selected(); !slices.Equal(got, []int{195}) {
		t.Fatalf("selected %v after rows came, want [195]", got)
	}
	if !inView(tt, "r0250") {
		t.Fatal("a row chosen near the end stops following")
	}
}

// Sorted descending by Name, the end is r0001: rows that sort above stay out.
func TestTableFollowEndSortedDescending(t *testing.T) {
	tb := testTable(followRows(1, 200)...).FollowEnd(true)
	tt := tester(t, tb)
	click(t, tt, "Name")
	click(t, tt, "Name")
	tt.Frame()
	if !inView(tt, "r0001") {
		t.Fatalf("sorted descending while following: end r0001 not in view")
	}
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if !inView(tt, "r0001") {
		t.Fatalf("rows sorted above the end moved the table away from it")
	}
}

func TestTableFollowEndFiltered(t *testing.T) {
	tb := testTable(followRows(1, 300)...).FollowEnd(true)
	tt := tester(t, tb)
	tb.Search("Name", "r01")
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0199") {
		t.Fatalf("filtered while following: r0199 not in view")
	}
	tb.AddRows([]acct{{1001, "r0199x", 1}})
	tt.Frame()
	if !inView(tt, "r0199x") {
		t.Fatalf("a row that matches the filter is not followed")
	}
}

func TestTableFollowEndHomeAndEnd(t *testing.T) {
	tb := testTable(followRows(1, 200)...).FollowEnd(true)
	tt := tester(t, tb)
	click(t, tt, "r0195")
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0001") {
		t.Fatal("Home did not go to the top")
	}
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if !inView(tt, "r0001") {
		t.Fatal("after Home, rows came and the table moved")
	}
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatal("End did not go to the end")
	}
	tb.AddRows(followRows(251, 50))
	tt.Frame()
	if !inView(tt, "r0300") {
		t.Fatal("after End, the table does not follow again")
	}
}

// ScrollToEnd through Details() while another view shows.
func TestFollowListViewScrollToEndWaitsForTheDetailsView(t *testing.T) {
	lv := ListView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).Expand()
	lv.SetRows(followRows(1, 200))
	tt := tester(t, lv)
	tt.Frame()
	lv.SetView(ViewList)
	tt.Frame()
	lv.Details().ScrollToEnd()
	tt.Frame()
	tt.Frame()
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0200") {
		t.Fatalf("ScrollToEnd called while the List view showed did not wait for the Details view: %q", tt.Texts())
	}
}

// Following in Details, switched to List, rows added, back to Details.
func TestListViewFollowEndAcrossViews(t *testing.T) {
	lv := ListView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).Expand()
	lv.Details().FollowEnd(true)
	lv.SetRows(followRows(1, 200))
	tt := tester(t, lv)
	lv.SetView(ViewList)
	tt.Frame()
	lv.AddRows(followRows(201, 50))
	tt.Frame()
	lv.SetView(ViewDetails)
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatalf("back in Details, following, r0250 not in view")
	}
}

// FollowEnd and ScrollToEnd from a worker while frames run: the ListState is
// ui's, on the main thread (listbox.go, table.go), read and written by layout
// without mu. Red only under -race (CI).
func TestFollowEndSettersFromAWorkerDoNotRace(t *testing.T) {
	tb := testTable(followRows(1, 200)...)
	lb := newListBox().SetItems(followItems(1, 200)).Expand()
	tt := tester(t, SplitContainer(tb, lb))
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 200 {
			tb.FollowEnd(i%2 == 0).ScrollToEnd()
			lb.FollowEnd(i%2 == 0).ScrollToEnd()
		}
	}()
	for range 100 {
		tt.Frame()
	}
	wg.Wait()
}

// Kept rows: a table following its end builds only the rows added.
func TestTableFollowEndKeepsRows(t *testing.T) {
	for _, follow := range []bool{false, true} {
		var calls atomic.Int64
		rows := followRows(1, 2000)
		tb := DataGridView(func(a *acct) int { return a.id },
			Column[acct]{Title: "Name", Text: func(a *acct) string { calls.Add(1); return a.name }, Width: 200},
			Column[acct]{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) }, Width: 100},
		).SetRows(rows).Expand().FollowEnd(follow)
		tt := tester(t, tb)
		if !follow {
			tb.ScrollToEnd()
		}
		for range 4 {
			tt.Frame()
		}
		before := calls.Load()
		for range 5 {
			tt.Frame()
		}
		idle := calls.Load() - before
		before = calls.Load()
		tb.AddRows(followRows(2001, 1))
		tt.Frame()
		add := calls.Load() - before
		before = calls.Load()
		tt.Frame()
		after := calls.Load() - before
		if idle != 0 || after != 0 || add > 4 {
			t.Errorf("follow=%v: %d text calls at rest, %d building one row added, %d the frame after", follow, idle, add, after)
		}
		if follow && !inView(tt, "r2001") {
			t.Error("the row added is not in view")
		}
	}
}

package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func followRows(from, n int) []acct {
	rows := make([]acct, n)
	for i := range rows {
		rows[i] = acct{from + i, fmt.Sprintf("r%04d", from+i), from + i}
	}
	return rows
}

// inView reports whether the text s lies within the 600 high window.
func inView(tt *ui.Tester, s string) bool {
	r, ok := tt.Find(s)
	return ok && r.H > 0 && r.Y >= 0 && r.Y+r.H <= 600
}

func TestTableFollowEnd(t *testing.T) {
	tb := testTable(followRows(1, 200)...).FollowEnd(true)
	tt := tester(t, tb)
	if !inView(tt, "r0200") {
		t.Fatalf("a table that follows its end starts at it: %q", tt.Texts())
	}
	for i := range 5 {
		tb.AddRows(followRows(201+i*50, 50))
		tt.Frame()
		if last := fmt.Sprintf("r%04d", 250+i*50); !inView(tt, last) {
			t.Fatalf("after adding %d rows, %s is not in view", 50*(i+1), last)
		}
	}
}

func TestTableWithoutFollowEndStaysPut(t *testing.T) {
	tb := testTable(followRows(1, 200)...)
	tt := tester(t, tb)
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if inView(tt, "r0250") || !inView(tt, "r0001") {
		t.Fatalf("a table that does not follow its end moved: %q", tt.Texts())
	}
}

func TestTableFollowEndStopsWhenTheUserScrollsUpAndResumesAtTheEnd(t *testing.T) {
	tb := testTable(followRows(1, 200)...).FollowEnd(true)
	tt := tester(t, tb)
	tt.Scroll(300, 300, 0, -400)
	tt.Frame()
	tt.Frame()
	top := func() string { // the first row name in view
		for i := 1; i <= 400; i++ {
			if s := fmt.Sprintf("r%04d", i); inView(tt, s) {
				return s
			}
		}
		return ""
	}
	before := top()
	if before == "" || inView(tt, "r0200") {
		t.Fatalf("scrolling up did not leave the end: top %q", before)
	}
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if after := top(); after != before || inView(tt, "r0250") {
		t.Fatalf("rows added while scrolled away moved the table: top %q, was %q", after, before)
	}
	tt.Scroll(300, 300, 0, 100000)
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatalf("back at the end, r0250 is not in view: %q", tt.Texts())
	}
	tb.AddRows(followRows(251, 50))
	tt.Frame()
	if !inView(tt, "r0300") {
		t.Fatal("back at the end, the table does not follow it again")
	}
}

func TestTableScrollToEnd(t *testing.T) {
	tb := testTable(followRows(1, 200)...)
	tt := tester(t, tb)
	tb.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0200") {
		t.Fatalf("ScrollToEnd left r0200 out of view: %q", tt.Texts())
	}
}

func TestTableFollowEndOffStopsFollowing(t *testing.T) {
	tb := testTable(followRows(1, 200)...).FollowEnd(true)
	tt := tester(t, tb)
	tb.FollowEnd(false)
	tb.AddRows(followRows(201, 50))
	tt.Frame()
	if inView(tt, "r0250") {
		t.Fatal("FollowEnd(false) still follows")
	}
}

func followItems(from, n int) []litem {
	items := make([]litem, n)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", from+i), fmt.Sprintf("r%04d", from+i)}
	}
	return items
}

func TestListBoxFollowEnd(t *testing.T) {
	lb := newListBox().SetItems(followItems(1, 200)).FollowEnd(true).Expand()
	tt := tester(t, lb)
	if !inView(tt, "r0200") {
		t.Fatalf("a list box that follows its end starts at it: %q", tt.Texts())
	}
	lb.AddItems(followItems(201, 50))
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatal("rows added at the end are not followed")
	}
	tt.Scroll(300, 300, 0, -400)
	tt.Frame()
	lb.AddItems(followItems(251, 50))
	tt.Frame()
	if inView(tt, "r0300") {
		t.Fatal("a list box the user scrolled away from the end follows it")
	}
	lb.ScrollToEnd()
	tt.Frame()
	tt.Frame()
	if !inView(tt, "r0300") {
		t.Fatalf("ScrollToEnd left r0300 out of view: %q", tt.Texts())
	}
}

// A list view in Details follows its end through Details().
func TestListViewDetailsFollowEnd(t *testing.T) {
	lv := ListView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).Expand()
	lv.Details().FollowEnd(true)
	lv.SetRows(followRows(1, 200))
	tt := tester(t, lv)
	if !inView(tt, "r0200") {
		t.Fatalf("a list view that follows its end starts at it: %q", tt.Texts())
	}
	lv.AddRows(followRows(201, 50))
	tt.Frame()
	if !inView(tt, "r0250") {
		t.Fatal("rows added at the end are not followed")
	}
}

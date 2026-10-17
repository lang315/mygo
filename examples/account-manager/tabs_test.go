package main

import "testing"

func TestTabsShowTheirColumns(t *testing.T) {
	a := newApp(1000)
	tt := newTester(t, a)
	tt.SetSize(3200, 800)
	tt.Frame()
	tt.want("Mật khẩu")
	if tt.HasText("TKQC") {
		t.Fatal("the first tab shows the Ads columns")
	}
	tt.do(tt.Click("Ads"))
	tt.want("TKQC")
	if tt.HasText("RefreshID") {
		t.Fatal("the Ads tab shows the Email columns")
	}
	tt.do(tt.Click("Email"))
	tt.want("RefreshID")
	tt.want("Tổng số dòng: 1000")
	tt.do(tt.Click("Tài khoản"))
	tt.want("Bạn bè")
}

func TestChecksAreOfTheTabShown(t *testing.T) {
	a := newApp(1000)
	tt := newTester(t, a)
	tt.do(tt.Click("Check all"))
	tt.want("Đã chọn: 1000")
	tt.do(tt.Click("Ads"))
	tt.want("Đã chọn: 0")
	tt.do(tt.Click("Check all"))
	tt.want("Đã chọn: 1000")
	a.showFolder() // a folder switch clears every tab
	if n := a.table.Checked.Len() + a.ads.Checked.Len() + a.email.Checked.Len(); n != 0 {
		t.Fatalf("%d rows still checked after a folder switch", n)
	}
}

func TestSearchIsKeptPerTab(t *testing.T) {
	a := newApp(1000)
	tt := newTester(t, a)
	uid := a.s.accounts[7].UID
	a.keyword = uid
	tt.do(tt.Click("Tìm kiếm"))
	tt.want("Tổng số dòng: 1")
	tt.do(tt.Click("Ads"))
	if a.keyword != "" || a.column != "UID" {
		t.Fatalf("the Ads tab shows the search %q in %q of the first tab", a.keyword, a.column)
	}
	tt.want("Tổng số dòng: 1000")
	tt.do(tt.Click("Tài khoản"))
	if a.keyword != uid {
		t.Fatalf("the search typed in the first tab is %q after a switch back, want %q", a.keyword, uid)
	}
	tt.want("Tổng số dòng: 1")
}

func TestHiddenTabPlacesTheChangedRowsWhenShownAgain(t *testing.T) {
	a := newApp(100)
	tt := newTester(t, a)
	tt.do(tt.Click("Email"))
	tt.do(tt.Click("Trạng thái")) // sorted by status, all of them empty
	tt.do(tt.Click("Ads"))
	// The tab hidden is told which rows changed, and places them once shown.
	a.s.accounts[3].Status = "zzz"
	a.changed(3)
	tt.Frame()
	tt.do(tt.Click("Email"))
	if last := a.email.At(a.email.Len() - 1); last.ID != 3 {
		t.Fatalf("the last row of the Email tab sorted by status is account %d, want 3 (status %q)", last.ID, "zzz")
	}
}

// TestShowingATabAgainPlacesNoRows: a tab whose rows did not change while
// it was hidden shows them as it left them, without placing them again (at
// 1,000,000 rows, placing them all took 37 to 78 ms a switch).
func TestShowingATabAgainPlacesNoRows(t *testing.T) {
	a := newApp(1000)
	tt := newTester(t, a)
	tt.do(tt.Click("Email"))
	var kept int
	filter := a.email.Filter
	a.email.Filter = func(x *account) bool { kept++; return filter(x) }
	tt.do(tt.Click("Ads"))
	tt.do(tt.Click("Email"))
	if kept != 0 {
		t.Errorf("showing the Email tab again went through %d rows, want none", kept)
	}
	// A row changed while hidden is placed again, not all of them.
	tt.do(tt.Click("Ads"))
	a.s.accounts[3].Status = "zzz"
	a.changed(3)
	tt.do(tt.Click("Email"))
	if kept > 10 {
		t.Errorf("showing the Email tab after one row changed went through %d rows", kept)
	}
}

// TestHiddenTabPlacesTheRowsLeftWhenShownAgain: accounts removed while a
// tab is hidden are gone from it once shown.
func TestHiddenTabPlacesTheRowsLeftWhenShownAgain(t *testing.T) {
	a := newApp(100)
	tt := newTester(t, a)
	tt.do(tt.Click("Email"))
	tt.do(tt.Click("Ads"))
	a.s.remove([]int{0, 1, 2})
	a.forget([]int{0, 1, 2})
	a.reset()
	tt.Frame()
	tt.do(tt.Click("Email"))
	if n := a.email.Len(); n != 97 {
		t.Fatalf("the Email tab shows %d rows after 3 of 100 were removed, want 97", n)
	}
}

func TestForgetDropsTheKeysOfEveryTab(t *testing.T) {
	a := newApp(100)
	newTester(t, a)
	for _, tb := range a.tables() {
		tb.Checked.Add(5)
		tb.Chosen.Add(5)
	}
	a.s.remove([]int{5})
	a.forget([]int{5})
	for i, tb := range a.tables() {
		if tb.Checked.Has(5) || tb.Chosen.Has(5) {
			t.Errorf("tab %d keeps the removed account", i)
		}
	}
}

func TestSwitchingTabsAt100kRows(t *testing.T) {
	a := newApp(100_000)
	tt := newTester(t, a)
	tt.SetSize(3200, 800)
	for _, label := range []string{"Ads", "Email", "Tài khoản", "Ads"} {
		tt.do(tt.Click(label))
		tt.want("Tổng số dòng: 100000")
	}
	tt.do(tt.Click("TKQC")) // sort the Ads tab
	tt.want("Tổng số dòng: 100000")
}

// TestEachTabOpensUnscrolled: the tabs' tables are different widgets, so the
// horizontal scroll of one is not the other's.
func TestEachTabOpensUnscrolled(t *testing.T) {
	a := newApp(1000)
	tt := newTester(t, a) // 1280 px wide: the first tab's 18 columns do not fit
	tt.do(tt.Click("Email"))
	home, ok := tt.Find("UID")
	if !ok {
		t.Fatal("no UID header on the Email tab")
	}
	tt.do(tt.Click("Tài khoản"))
	tt.Scroll(600, 400, 800, 0)
	tt.Frame()
	if now, _ := tt.Find("UID"); now.X >= home.X {
		t.Fatalf("the first tab did not scroll sideways: UID header at %v, was %v", now.X, home.X)
	}
	tt.do(tt.Click("Email"))
	if now, ok := tt.Find("UID"); !ok || now.X != home.X {
		t.Fatalf("the Email tab opens scrolled: UID header at %v (shown %v), want %v", now.X, ok, home.X)
	}
}

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
	// The tab hidden is not told which rows changed (they would pile up, as
	// it never syncs): it places all of them again once shown.
	a.s.accounts[3].Status = "zzz"
	a.changed(3)
	tt.Frame()
	tt.do(tt.Click("Email"))
	if last := a.email.At(a.email.Len() - 1); last.ID != 3 {
		t.Fatalf("the last row of the Email tab sorted by status is account %d, want 3 (status %q)", last.ID, "zzz")
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

package main

import (
	"reflect"
	"slices"
	"testing"
)

func TestGenerateIsTheSameForASeed(t *testing.T) {
	a, b := generate(1000, 0, 1), generate(1000, 0, 1)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("two generations from one seed differ")
	}
	uids := map[string]bool{}
	for i, x := range a {
		if x.ID != i || x.Folder != 0 {
			t.Fatalf("account %d has ID %d, folder %d", i, x.ID, x.Folder)
		}
		// One in a thousand repeats the UID before it, for Lọc trùng UID
		// to find; the others differ.
		if dup := i%1000 == 500; dup != uids[x.UID] {
			t.Fatalf("account %d: UID %s seen before: %v", i, x.UID, uids[x.UID])
		}
		uids[x.UID] = true
	}
	if x := generate(1, 1000, 1)[0]; x.ID != 1000 || uids[x.UID] {
		t.Fatalf("the account after them: ID %d, UID %s", x.ID, x.UID)
	}
}

func TestCompareSortsNumbersAndTimesByValue(t *testing.T) {
	m := compare()
	if len(m) != len(allColumns) {
		t.Fatalf("%d comparators for %d columns", len(m), len(allColumns))
	}
	if m["Bạn bè"](&account{Friends: 9}, &account{Friends: 10}) >= 0 {
		t.Error("9 friends sort after 10, as text does")
	}
	if m["Lần chạy cuối"](&account{LastRunAt: 99}, &account{LastRunAt: 100}) >= 0 {
		t.Error("an earlier run sorts after a later one")
	}
	if m["UID"](&account{UID: "1"}, &account{UID: "2"}) >= 0 {
		t.Error("UIDs out of order")
	}
}

func TestCellText(t *testing.T) {
	x := &account{Friends: 42}
	if got := cellText(x, colFriends); got != "42" {
		t.Errorf("friends %q", got)
	}
	if got := cellText(x, colLastRun); got != "" {
		t.Errorf("a run never made shows %q", got)
	}
}

func TestNewColumnsDoNotChangeTheOthers(t *testing.T) {
	x := generate(5, 0, 1)[4]
	got := []string{x.UID, x.Password, x.TwoFA, x.Cookie, x.Email}
	want := []string{"100000000036390", "mTakBRRrHo", "A5OJR7M4MC3BTY5VEB4BJEGRJKB6IEAN", "c_user=100000000036390; xs=fsC70gCQ9Q6eoxc5uIePl1dj", "ymddtcfp2850@hotmail.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("columns of the first tab changed: %q, want %q", got, want)
	}
}

func TestAdsAndEmailData(t *testing.T) {
	rows := generate(3000, 0, 1)
	var ads int
	for i := range rows {
		x := &rows[i]
		if len(x.RefreshID) != 40 || len(x.EmailPassword) != 10 || x.ClientID == "" {
			t.Fatalf("row %d: email data %q %q %q", i, x.EmailPassword, x.RefreshID, x.ClientID)
		}
		if (x.BM == "") != (x.TKQC == "") {
			t.Fatalf("row %d: BM %q with TKQC %q", i, x.BM, x.TKQC)
		}
		if x.BM != "" {
			ads++
		}
	}
	if ads < 800 || ads > 1200 { // about a third
		t.Errorf("%d of 3000 accounts have ads, want about 1000", ads)
	}
}

func TestTabsListTheirColumns(t *testing.T) {
	want := [][]string{
		{"UID", "Mật khẩu", "2FA", "Cookie", "Trạng thái", "BM", "TKQC"},
		{"UID", "Email", "Mật khẩu email", "Trạng thái", "Email khôi phục", "RefreshID", "ClientID"},
	}
	for i, w := range want {
		if got := tabTitles[tabAds+i]; !slices.Equal(got, w) {
			t.Errorf("tab %d shows %q, want %q", tabAds+i, got, w)
		}
	}
	if len(tabTitles[tabAccounts]) != 18 {
		t.Errorf("the first tab shows %d columns, want 18", len(tabTitles[tabAccounts]))
	}
	seen := map[string]bool{}
	for _, c := range allColumns {
		if seen[c.Title] {
			t.Errorf("two columns titled %q: the sorts are found by title", c.Title)
		}
		seen[c.Title] = true
	}
	x := &account{BM: "1", TKQC: "act_2", EmailPassword: "p", RefreshID: "r", ClientID: "c"}
	for col, w := range map[int]string{colBM: "1", colTKQC: "act_2", colEmailPassword: "p", colRefreshID: "r", colClientID: "c"} {
		if got := cellText(x, col); got != w {
			t.Errorf("cellText(%s) = %q, want %q", allColumns[col].Title, got, w)
		}
	}
}

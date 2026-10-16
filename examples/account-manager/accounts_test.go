package main

import (
	"reflect"
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
	if len(m) != len(columns) {
		t.Fatalf("%d comparators for %d columns", len(m), len(columns))
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

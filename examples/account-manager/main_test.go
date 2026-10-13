package main

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// tester shows a in a headless window, with updates from workers queued
// for wait to run.
type tester struct {
	*ui.Tester
	t       *testing.T
	a       *app
	mu      sync.Mutex
	pending []func()
}

func newTester(t *testing.T, a *app) *tester {
	tt := &tester{t: t, a: a}
	a.update = func(fn func()) {
		tt.mu.Lock()
		tt.pending = append(tt.pending, fn)
		tt.mu.Unlock()
	}
	tt.Tester = ui.NewTester(a.build, 1280, 800)
	return tt
}

func (tt *tester) want(s string) {
	tt.t.Helper()
	if !tt.HasText(s) {
		tt.t.Fatalf("no %q in %q", s, tt.Texts())
	}
}

func (tt *tester) do(err error) {
	tt.t.Helper()
	if err != nil {
		tt.t.Fatal(err)
	}
}

// wait runs the workers' updates until the run ends.
func (tt *tester) wait() {
	tt.t.Helper()
	for end := time.Now().Add(10 * time.Second); tt.a.run != nil; {
		if time.Now().After(end) {
			tt.t.Fatal("the run did not end")
		}
		tt.mu.Lock()
		fns := tt.pending
		tt.pending = nil
		tt.mu.Unlock()
		for _, fn := range fns {
			fn()
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()
}

func TestAppWith100kAccounts(t *testing.T) {
	a := newApp(100_000)
	tt := newTester(t, a)
	tt.want("Tổng số dòng: 100000")

	tt.do(tt.Click("Check all"))
	tt.want("Đã chọn: 100000")
	tt.do(tt.Click("Check all"))
	tt.want("Đã chọn: 0")

	// The search finds two accounts by their UIDs.
	u1, u2 := a.s.accounts[10].UID, a.s.accounts[99_999].UID
	tt.do(tt.Click("Nhập từ khoá (UID1|UID2|…)"))
	tt.Type(u1 + "|" + u2)
	tt.do(tt.Click("Tìm kiếm"))
	tt.want("Tổng số dòng: 2")

	// A right click on a row not chosen moves it alone.
	tt.do(tt.RightClick(u1))
	tt.do(tt.ChooseMenuItem("Chuyển thư mục", "Thư mục 2"))
	tt.want("Tổng số dòng: 1")
	if a.s.accounts[a.s.index[10]].Folder != 1 {
		t.Fatal("the account did not move")
	}

	// Copy > UID puts the UID on the clipboard.
	tt.do(tt.RightClick(u2))
	tt.do(tt.ChooseMenuItem("Copy", "UID"))
	if got := tt.Clipboard(); got != u2 {
		t.Fatalf("clipboard %q, want %q", got, u2)
	}

	// Start with nothing checked says so.
	tt.do(tt.Click("Start"))
	tt.want("Bạn chưa chọn tài khoản nào để chạy!")

	// Space checks the row chosen; a run reports on it.
	tt.do(tt.Click(u2))
	tt.Key(0, ui.KeySpace)
	tt.want("Đã chọn: 1")
	a.action = "Backup" // the select's popup is the select's own test's
	tt.do(tt.Click("Start"))
	if a.run == nil {
		t.Fatal("Start did not run")
	}
	tt.want("Stop")
	tt.wait()
	tt.want("Start")
	if !tt.HasText("Backup: Thành công") && !tt.HasText("Lỗi: checkpoint") {
		t.Fatalf("no result shown in %q", tt.Texts())
	}

	// Xóa sends it to the trash; there, Xóa deletes it.
	tt.do(tt.RightClick(u2))
	tt.do(tt.ChooseMenuItem("Xóa 1 tài khoản"))
	tt.want("Tổng số dòng: 0")
	a.folder = "Thùng rác"
	a.showFolder()
	tt.Frame()
	tt.want("Tổng số dòng: 1")
	tt.do(tt.RightClick(u2))
	tt.do(tt.ChooseMenuItem("Xóa 1 tài khoản"))
	tt.want("Tổng số dòng: 0")
	if len(a.s.accounts) != 99_999 {
		t.Fatalf("%d accounts", len(a.s.accounts))
	}
}

func TestApplyDropsResultsOfAccountsGone(t *testing.T) {
	a := newApp(3)
	r := &runner{done: make(chan struct{})}
	r.push(result{id: 1, state: succeeded, status: "ok", at: 1})
	r.push(result{id: 2, state: failed, status: "Lỗi: checkpoint", at: 1})
	a.s.remove([]int{1})
	a.view.Reset()
	a.apply(r)
	if x := a.s.accounts[a.s.index[2]]; x.Status != "Lỗi: checkpoint" || x.state != failed {
		t.Fatalf("account 2: %+v", x)
	}
	for _, x := range a.s.accounts {
		if x.Status == "ok" {
			t.Fatalf("a result for account 1 reached account %d", x.ID)
		}
	}
}

func TestDeletingTheFolderShownFallsBack(t *testing.T) {
	a := newApp(2)
	tt := newTester(t, a)
	tt.do(tt.Click("Xóa thư mục"))
	tt.want("Tổng số dòng: 0") // now on Thư mục 2
	tt.do(tt.Click("Xóa thư mục"))
	if a.folder != "Thùng rác" {
		t.Fatalf("showing %q", a.folder)
	}
	tt.want("Tổng số dòng: 2")
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "Thùng rác") {
		t.Fatal("the folder select does not show the trash")
	}
}

// The filter goes through every row whenever it changes: a row it copies
// to the heap costs a million allocations at a million rows.
func TestFilterDoesNotAllocatePerRow(t *testing.T) {
	a := newApp(10)
	a.column, a.keyword = "UID", a.s.accounts[3].UID+"|x"
	keep := a.keep()
	x := a.s.accounts[3]
	if !keep(x) {
		t.Fatal("the account searched for is not kept")
	}
	if n := testing.AllocsPerRun(100, func() { keep(x) }); n != 0 {
		t.Fatalf("%v allocations a row", n)
	}
}

func TestStartSaysTheRowsCheckedAreHidden(t *testing.T) {
	a := newApp(100)
	tt := newTester(t, a)
	tt.do(tt.Click("Check " + a.s.accounts[0].UID))
	a.keyword = a.s.accounts[1].UID
	a.filter()
	tt.Frame()
	a.action = "Backup"
	tt.do(tt.Click("Start"))
	if a.run != nil {
		t.Fatal("a run started on rows the search hides")
	}
	tt.want("Các dòng đã chọn đang bị ẩn bởi tìm kiếm!")
}

func TestAddedAccountShowsInTheOrderSorted(t *testing.T) {
	a := newApp(100)
	tt := newTester(t, a)
	a.sort = ui.SortOrder{Column: "Bạn bè"} // its header is out of view
	tt.Frame()
	tt.do(tt.Click("Thêm tài khoản"))
	tt.want("Tổng số dòng: 101")
	x := a.s.accounts[100]
	for i := range a.view.Len() {
		if a.view.Row(i) == 100 {
			if i > 0 && a.s.accounts[a.view.Row(i-1)].Friends > x.Friends {
				t.Fatalf("the account added is out of order at %d", i)
			}
			return
		}
	}
	t.Fatal("the account added is not shown")
}

func TestDedupeRemovesTheRepeatedUID(t *testing.T) {
	a := newApp(1000)
	tt := newTester(t, a)
	uid := a.s.accounts[500].UID
	a.keyword = uid
	a.filter()
	tt.Frame()
	tt.want("Tổng số dòng: 2")
	tt.do(tt.Click(uid))
	tt.Key(ui.Shift, ui.KeyDown)
	tt.want("Số dòng đang bôi đen: 2")
	tt.do(tt.RightClick(uid))
	tt.do(tt.ChooseMenuItem("Lọc trùng UID 2 tài khoản"))
	tt.want("Tổng số dòng: 1")
	tt.want("Đã lọc 1 tài khoản trùng")
}

func TestFoldersChangeWhileRunning(t *testing.T) {
	a := newApp(50)
	tt := newTester(t, a)
	tt.do(tt.Click("Check all"))
	a.action = "Backup"
	tt.do(tt.Click("Start"))
	if a.run == nil {
		t.Fatal("Start did not run")
	}
	tt.do(tt.Click("Thêm thư mục"))
	tt.do(tt.Click("Đổi tên"))
	if len(a.s.folders) != 4 || a.folder != "Thư mục 3 (đổi tên)" {
		t.Fatalf("folders %q, showing %q", a.s.names(), a.folder)
	}
	a.run.cancel()
	tt.wait()
}

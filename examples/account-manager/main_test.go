package main

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// tester shows a in a headless window, with updates from workers queued
// for its frames to run.
type tester struct {
	*ui.Tester
	t *testing.T
	a *app
}

func newTester(t *testing.T, a *app) *tester {
	tt := &tester{t: t, a: a, Tester: ui.NewTester(a.build, 1280, 800)}
	a.update = tt.Update
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

// wait runs the updates of the run *r until it ends.
func (tt *tester) wait(r **runner) {
	tt.t.Helper()
	for end := time.Now().Add(10 * time.Second); *r != nil; {
		if time.Now().After(end) {
			tt.t.Fatal("the run did not end")
		}
		tt.Frame()
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
	tt.wait(&a.run)
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
	a.table.Reset()
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
	a.table.Sort = ui.SortOrder{Column: "Bạn bè"} // its header is out of view
	tt.Frame()
	tt.do(tt.Click("Thêm tài khoản"))
	tt.want("Tổng số dòng: 101")
	x := a.s.accounts[100]
	for i := range a.table.Len() {
		if a.table.At(i).ID == x.ID {
			if i > 0 && a.table.At(i-1).Friends > x.Friends {
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
	tt.wait(&a.run)
}

func TestApplyAddsTheChanges(t *testing.T) {
	a := newApp(3)
	x0 := a.s.accounts[1]
	r := &runner{done: make(chan struct{})}
	r.push(result{id: 1, state: running, status: "Đang chạy: bước 2/7", friends: 3, groups: -1, note: "Cập nhật"})
	r.push(result{id: 2, state: running, status: "Đang chạy: bước 1/7", friends: -1_000_000, groups: -1_000_000})
	a.apply(r)
	x := a.s.accounts[1]
	if x.Friends != x0.Friends+3 || x.Groups != max(0, x0.Groups-1) || x.Note != "Cập nhật" || x.Status != "Đang chạy: bước 2/7" {
		t.Fatalf("account 1: %+v, was %+v", x, x0)
	}
	if y := a.s.accounts[2]; y.Friends != 0 || y.Groups != 0 || y.Note != "" {
		t.Fatalf("account 2: %+v", y)
	}
}

func TestStressShowsInTheTable(t *testing.T) {
	a := newApp(100)
	a.stats = newStats(time.Now())
	tt := newTester(t, a)
	a.startStress(2000)
	time.Sleep(300 * time.Millisecond)
	a.stress.cancel()
	tt.wait(&a.stress)
	if a.stats.total == 0 {
		t.Fatal("no change applied")
	}
	if !tt.HasText("Đang chạy: bước") && !tt.HasText("Thành công") && !tt.HasText("Lỗi: checkpoint") {
		t.Fatalf("no status shown in %q", tt.Texts())
	}
}

// BenchmarkLiveUpdates applies a batch of k changes to n rows and builds a
// frame, as a stress run does ten times a second. At a million rows only
// the batch of a run of 10,000 changes a second is measured.
func BenchmarkLiveUpdates(b *testing.B) {
	for _, n := range []int{100_000, 1_000_000} {
		ks := []int{10, 100, 1000}
		if n > 100_000 {
			ks = ks[2:]
		}
		for _, sorted := range []string{"", "Bạn bè", "Trạng thái"} {
			for _, k := range ks {
				b.Run(fmt.Sprintf("rows=%d/sort=%s/k=%d", n, cmp.Or(sorted, "none"), k), func(b *testing.B) {
					a := newApp(n)
					a.table.Sort = ui.SortOrder{Column: sorted}
					tt := ui.NewTester(a.build, 1280, 800)
					rnd := rand.New(rand.NewPCG(1, 2))
					now := time.Now()
					r := &runner{done: make(chan struct{})}
					for b.Loop() {
						for range k {
							r.push(change(rnd, n, now))
						}
						a.apply(r)
						tt.Frame()
					}
				})
			}
		}
	}
}

// BenchmarkSearch searches a million rows sorted by Bạn bè for "77" in
// their UIDs, then clears the search: each goes through all rows once, and
// builds the frame.
func BenchmarkSearch(b *testing.B) {
	a := newApp(1_000_000)
	a.table.Sort = ui.SortOrder{Column: "Bạn bè"}
	tt := ui.NewTester(a.build, 1280, 800)
	on := false
	for b.Loop() {
		if on = !on; on {
			a.keyword = "77"
		} else {
			a.keyword = ""
		}
		a.filter()
		tt.Frame()
	}
}

// The table filters by folder from the start.
func TestNewAppShowsOnlyItsFolder(t *testing.T) {
	a := newApp(3)
	tt := newTester(t, a)
	a.s.accounts[0].Folder = 1
	a.table.Changed(0)
	tt.Frame()
	if a.table.Len() != 2 {
		t.Fatalf("%d rows shown in Mặc định, want 2", a.table.Len())
	}
}

func TestMoveSortsOnlyTheRowMoved(t *testing.T) {
	a := newApp(10_000)
	calls := 0
	for i := range a.table.Columns {
		f := a.table.Columns[i].Compare
		a.table.Columns[i].Compare = func(x, y *account) int { calls++; return f(x, y) }
	}
	a.table.Sort = ui.SortOrder{Column: "Bạn bè"}
	f := a.s.addFolder()
	tt := newTester(t, a)
	tt.Frame()
	u := a.table.At(0).UID
	calls = 0
	tt.do(tt.RightClick(u))
	tt.do(tt.ChooseMenuItem("Chuyển thư mục", f.Name))
	tt.Frame()
	tt.want("Tổng số dòng: 9999")
	// A sort of 10,000 rows compares them about 120,000 times.
	if calls > 1000 {
		t.Fatalf("moving one account compared rows %d times", calls)
	}
}

// BenchmarkSearchStatus searches the statuses a run leaves, Vietnamese
// text, which a lowercase copy of each cell would allocate for, and builds
// the frame.
func BenchmarkSearchStatus(b *testing.B) {
	a := newApp(1_000_000)
	statuses := []string{"Kiểm tra: Thành công", "Lỗi: checkpoint", "Đã dừng", "Đang chạy..."}
	for i := range a.s.accounts {
		a.s.accounts[i].Status = statuses[i%len(statuses)]
	}
	a.table.Sort = ui.SortOrder{Column: "Bạn bè"}
	a.column = "Trạng thái"
	tt := ui.NewTester(a.build, 1280, 800)
	on := false
	for b.Loop() {
		if on = !on; on {
			a.keyword = "Thành công"
		} else {
			a.keyword = ""
		}
		a.filter()
		tt.Frame()
	}
}

func TestAppShowsOneSearchBar(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	if tt.HasText("Search") || tt.HasText("Column") {
		t.Fatalf("the table's own search bar is shown: %q", tt.Texts())
	}
	tt.want("Tìm kiếm")
}

// A folder switch keeps the search applied, filters by both, and clears the
// accounts checked and highlighted, hidden by the search or not.
func TestFolderSwitchKeepsTheSearchAndClearsTheChecks(t *testing.T) {
	a := newApp(100)
	tt := newTester(t, a)
	other := a.s.addFolder()
	a.s.move([]int{a.s.accounts[0].ID, a.s.accounts[1].ID, a.s.accounts[2].ID}, other.ID)
	a.table.Changed(0, 1, 2)
	tt.Frame()
	tt.do(tt.Click("Check " + a.s.accounts[5].UID))
	tt.do(tt.Click(a.s.accounts[6].UID))
	tt.want("Đã chọn: 1")
	tt.want("Số dòng đang bôi đen: 1")
	a.keyword = a.s.accounts[1].UID
	a.filter() // hides the rows checked and highlighted, and all of folder 0
	tt.Frame()
	a.folder = other.Name
	a.showFolder()
	tt.Frame()
	tt.want("Tổng số dòng: 1") // the search is kept: not the three accounts of the folder
	tt.want("Đã chọn: 0")
	tt.want("Số dòng đang bôi đen: 0")
}

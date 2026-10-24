package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

// findItem finds the menu item labeled label, in any submenu.
func findItem(items []*mygo.MenuItem, label string) *mygo.MenuItem {
	for _, it := range items {
		if it.Label == label {
			return it
		}
		if sub := findItem(it.Submenu, label); sub != nil {
			return sub
		}
	}
	return nil
}

// choose clicks the menu item labeled label, as the menu bar does, and builds
// the next frame, as Window.Update does.
func choose(t *testing.T, tt *tester, label string) {
	t.Helper()
	it := findItem(tt.a.menu(), label)
	if it == nil {
		t.Fatalf("no menu item %q", label)
	}
	// A click must ask for a frame, as Window.Update does: with no input in
	// the window, nothing else would.
	// (a.update is not swapped here: the app's goroutines read it.)
	before := tt.asked.Load()
	it.Click(it, nil)
	if tt.asked.Load() == before {
		t.Fatalf("the click on %q asked for no frame", label)
	}
	tt.Frame()
}

func TestMenuBarHasTheAccountAndFolderMenus(t *testing.T) {
	a := newApp(10)
	var bar []string
	for _, it := range a.menu() {
		bar = append(bar, it.Label+string(it.Role))
	}
	for _, want := range []string{"Tài khoản", "Thư mục", string(mygo.RoleAppMenu), string(mygo.RoleEditMenu), string(mygo.RoleWindowMenu)} {
		if !strings.Contains(strings.Join(bar, "|"), want) {
			t.Errorf("no %q in the menu bar %q", want, bar)
		}
	}
	for _, label := range []string{"Thêm tài khoản", "Đăng nhập", "Backup", "Bật 2FA", "Kiểm tra tài nguyên", "Dừng", "Thêm thư mục", "Đổi tên", "Xóa thư mục"} {
		if findItem(a.menu(), label) == nil {
			t.Errorf("no menu item %q", label)
		}
	}
}

func TestMenuAddsAnAccountWithoutWindowInput(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	tt.want("Tổng số dòng: 10")
	choose(t, tt, "Thêm tài khoản")
	tt.want("Tổng số dòng: 11")
}

func TestMenuActionNeedsCheckedAccounts(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	choose(t, tt, "Đăng nhập")
	tt.want("Bạn chưa chọn tài khoản nào để chạy!")
	if a.run != nil {
		t.Fatal("a run started with nothing checked")
	}
}

func TestMenuActionRunsOnTheTabShown(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	a.loops = 1000 // a run that cannot end before the test stops it
	tt.do(tt.Click("Ads"))
	tt.do(tt.Click("Check all"))
	choose(t, tt, "Backup")
	if a.run == nil || a.action != "Backup" {
		t.Fatalf("run %v, action %q after the menu's Backup", a.run, a.action)
	}
	choose(t, tt, "Đăng nhập")
	tt.want("Đang chạy!")
	if a.action != "Backup" {
		t.Fatalf("a second action changed %q while running", a.action)
	}
	choose(t, tt, "Dừng")
	tt.wait(&a.run)
}

func TestMenuFolders(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	choose(t, tt, "Thêm thư mục")
	if a.folder != "Thư mục 3" || len(a.s.folders) != 4 {
		t.Fatalf("folder %q of %d", a.folder, len(a.s.folders))
	}
	choose(t, tt, "Đổi tên")
	if !strings.HasSuffix(a.folder, "(đổi tên)") {
		t.Fatalf("folder %q not renamed", a.folder)
	}
	choose(t, tt, "Xóa thư mục")
	if a.folder != "Mặc định" || len(a.s.folders) != 3 {
		t.Fatalf("after deleting: folder %q of %d", a.folder, len(a.s.folders))
	}
	a.folder = "Thùng rác"
	a.showFolder()
	choose(t, tt, "Xóa thư mục")
	if len(a.s.folders) != 3 {
		t.Fatal("the trash was deleted")
	}
	tt.want("Không xóa được thùng rác!")
}

func TestMenuDoesNotDeleteAFolderWhileRunning(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	a.loops = 1000
	tt.do(tt.Click("Check all"))
	choose(t, tt, "Backup")
	if a.run == nil {
		t.Fatal("no run started")
	}
	choose(t, tt, "Thêm thư mục") // the results reach the rows by ID: adding is fine
	choose(t, tt, "Xóa thư mục")
	if len(a.s.folders) != 4 {
		t.Fatalf("a folder was deleted while a run goes on: %d folders", len(a.s.folders))
	}
	tt.want("Đang chạy, chưa xóa được thư mục!")
	choose(t, tt, "Dừng")
	tt.wait(&a.run)
}

func TestMenuRenameOfTheTrashSaysSo(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	a.folder = "Thùng rác"
	a.showFolder()
	choose(t, tt, "Đổi tên")
	if a.folder != "Thùng rác" {
		t.Fatalf("the trash was renamed: %q", a.folder)
	}
	tt.want("Không đổi tên được thùng rác!")
}

func TestMenuStopWithNothingRunningSaysSo(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	choose(t, tt, "Dừng")
	tt.want("Không có lượt chạy nào để dừng!")
}

func TestMenuActionRefusedLeavesTheActionSelectAlone(t *testing.T) {
	a := newApp(10)
	tt := newTester(t, a)
	a.action = "Backup" // chosen in the toolbar's select
	choose(t, tt, "Bật 2FA")
	tt.want("Bạn chưa chọn tài khoản nào để chạy!")
	if a.action != "Backup" {
		t.Fatalf("a refused action changed the select to %q", a.action)
	}
}

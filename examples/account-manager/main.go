// Account-manager rebuilds the account manager of fhtool, a GTK tool that
// runs jobs on many accounts, in native UI: 100,000 made-up accounts in
// folders, in a table that sorts, searches and checks them, with a menu
// on its rows, and workers whose results the rows show as they come.
//
//	go run ./examples/account-manager
//	go run ./examples/account-manager -rows 1000000
//
// MYGO_FRAME_STATS=8 logs the frames slower than 8 ms.
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// actions are the jobs Start runs, the first of fhtool's.
var actions = []string{"Đăng nhập", "Backup", "Bật 2FA", "Kiểm tra tài nguyên"}

// app is the state the window shows.
type app struct {
	s       *store
	view    *ui.TableView[account]
	list    ui.ListState
	sort    ui.SortOrder
	cursor  int               // the row chosen last, -1 for none
	chosen  ui.Selection[int] // the accounts highlighted, by ID
	checked ui.Selection[int] // the accounts checked, by ID
	folder  string            // the folder shown
	// The search: the title of the column searched, and what to find.
	column, keyword string
	action          string
	loops, threads  float64
	run             *runner // the run going on, nil when none
	stopping        bool
	// update runs fn on the main thread, then builds a frame:
	// Window.Update, or a test's.
	update func(fn func())
}

func newApp(rows int) *app {
	a := &app{s: newStore(rows, 1), view: ui.NewTableView(compare()), cursor: -1, folder: "Mặc định", column: "UID", threads: 10}
	a.filter()
	return a
}

func (a *app) build(c *ui.Context) {
	ui.Column(c).Fill().Padding(10).Gap(10).Children(func() {
		a.toolbar(c)
		a.table(c)
		a.footer(c)
	})
}

// toolbar is fhtool's: a run's controls on the left, the folders' on the
// right.
func (a *app) toolbar(c *ui.Context) {
	busy := a.run != nil
	cur, _ := a.s.folderByName(a.folder)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "Thêm tài khoản").Clicked() {
			a.s.add(cur.ID)
			a.view.Reset()
		}
		ui.Select(c, &a.action, actions).Label("Hành động").Width(190)
		ui.NumberInput(c, &a.loops, 0, 1_000_000, 1).Label("Số lần lặp").Tooltip("Số lần lặp")
		ui.NumberInput(c, &a.threads, 1, 1_000_000, 1).Label("Số luồng").Tooltip("Số luồng")
		if !busy {
			if ui.PrimaryButton(c, "Start").Clicked() {
				a.start(c)
			}
		} else if ui.Button(c, "Stop").Disabled(a.stopping).Clicked() {
			a.stopping = true
			a.run.cancel()
		}
		ui.Spacer(c)
		if ui.Select(c, &a.folder, a.s.names()).Label("Thư mục").Width(220).Changed() {
			a.showFolder()
		}
		if ui.Button(c, "Thêm thư mục").Disabled(busy).Clicked() {
			a.folder = a.s.addFolder().Name
			a.showFolder()
		}
		if ui.Button(c, "Đổi tên").Disabled(busy || cur.ID == trash).Clicked() {
			a.folder = a.s.renameFolder(cur.ID)
		}
		if ui.Button(c, "Xóa thư mục").Disabled(busy || cur.ID == trash).Clicked() {
			a.s.deleteFolder(cur.ID)
			a.folder = a.s.folders[0].Name
			a.showFolder()
		}
	})
}

// showFolder shows the folder a.folder names afresh, none of its
// accounts checked or highlighted, as fhtool loads a folder.
func (a *app) showFolder() {
	a.checked.Clear()
	a.chosen.Clear()
	a.filter()
}

// filter shows the accounts of the folder shown that the search finds.
func (a *app) filter() { a.view.SetFilter(a.keep()) }

// keep tells whether the folder shown has an account the search finds.
func (a *app) keep() func(account) bool {
	f, _ := a.s.folderByName(a.folder)
	match := matcher(columnIndex(a.column), a.keyword)
	return func(x account) bool { return x.Folder == f.ID && match(x) }
}

func (a *app) table(c *ui.Context) {
	accounts, view := a.s.accounts, a.view
	view.Sync(accounts, a.sort)
	a.list.Key = func(row int) any { return accounts[view.Row(row)].ID }
	a.list.Label = func(row int) string { return accounts[view.Row(row)].UID }
	a.list.Selected, a.list.Selection, a.list.Checked = &a.cursor, &a.chosen, &a.checked
	a.list.Sort = &a.sort
	a.list.Version = view.Version()
	a.list.Copy = func(rows []int) string { return a.copy(rows, colUID, colPassword, colTwoFA) }
	a.list.RowMenu = func(m *ui.Menu, rows []int) { a.rowMenu(c, m, rows) }
	t := c.Theme()
	ui.Table(c, &a.list, columns, view.Len(), func(row, col int) {
		x := &accounts[view.Row(row)]
		cell := ui.Text(c, cellText(x, col)).SingleLine()
		if col != colStatus {
			return
		}
		switch x.state {
		case running:
			cell.TextColor(ui.RGB(250, 204, 21)).TextBackground(ui.RGB(0, 0, 0)).Bold()
		case failed:
			cell.TextColor(t.Danger)
		case succeeded:
			cell.TextColor(t.Success)
		}
	}).Grow(1)
}

// copy is columns cols of rows, | between them, a line each.
func (a *app) copy(rows []int, cols ...int) string {
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		x := &a.s.accounts[a.view.Row(row)]
		for j, col := range cols {
			if j > 0 {
				b.WriteByte('|')
			}
			b.WriteString(cellText(x, col))
		}
	}
	return b.String()
}

// copies are the Copy submenu's items, and the columns each copies.
var copies = []struct {
	label string
	cols  []int
}{
	{"UID", []int{colUID}},
	{"Mật khẩu", []int{colPassword}},
	{"2FA", []int{colTwoFA}},
	{"Cookie", []int{colCookie}},
	{"Proxy", []int{colProxy}},
	{"UID|Mật khẩu|2FA", []int{colUID, colPassword, colTwoFA}},
}

// rowMenu is fhtool's menu of the rows chosen, but for its items opening
// Chrome, proxies and tokens.
func (a *app) rowMenu(c *ui.Context, m *ui.Menu, rows []int) {
	ids := make([]int, len(rows))
	for i, row := range rows {
		ids[i] = a.s.accounts[a.view.Row(row)].ID
	}
	m.Submenu("Chọn/Bỏ chọn", func(m *ui.Menu) {
		if m.Item("Chọn").Chosen() {
			for _, id := range ids {
				a.checked.Add(id)
			}
		}
		if m.Item("Bỏ chọn").Chosen() {
			for _, id := range ids {
				a.checked.Remove(id)
			}
		}
	})
	m.Submenu("Copy", func(m *ui.Menu) {
		for _, it := range copies {
			if m.Item(it.label).Chosen() {
				c.WriteClipboard(a.copy(rows, it.cols...))
				c.Toast(fmt.Sprintf("Đã copy %d dòng", len(rows)))
			}
		}
	})
	cur, _ := a.s.folderByName(a.folder)
	m.Submenu("Chuyển thư mục", func(m *ui.Menu) {
		for _, f := range a.s.folders {
			if f.ID != cur.ID && m.Item(f.Name).Chosen() {
				a.s.move(ids, f.ID)
				a.gone(c, ids)
			}
		}
	})
	m.Separator()
	if len(ids) > 1 && m.Item(fmt.Sprintf("Lọc trùng UID %d tài khoản", len(ids))).Chosen() {
		dups := a.s.dedupe(ids)
		a.gone(c, dups)
		c.Toast(fmt.Sprintf("Đã lọc %d tài khoản trùng", len(dups)))
	}
	if m.Item(fmt.Sprintf("Xóa %d tài khoản", len(ids))).Chosen() {
		if cur.ID == trash {
			a.s.remove(ids)
		} else {
			a.s.move(ids, trash)
		}
		a.gone(c, ids)
	}
}

// gone shows the accounts ids left the folder shown.
func (a *app) gone(c *ui.Context, ids []int) {
	for _, id := range ids {
		a.checked.Remove(id)
		a.chosen.Remove(id)
	}
	a.view.Reset()
	c.Invalidate()
}

func (a *app) footer(c *ui.Context) {
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "Tổng số dòng: %d", a.view.Len())
		ui.Textf(c, "Số dòng đang bôi đen: %d", a.chosen.Len())
		ui.Textf(c, "Đã chọn: %d", a.checked.Len())
		ui.Spacer(c)
		titles := make([]string, len(columns))
		for i, col := range columns {
			titles[i] = col.Title
		}
		column := ui.Select(c, &a.column, titles).Label("Cột tìm kiếm").Width(170).Changed()
		const hint = "Nhập từ khoá (UID1|UID2|…)"
		// A text input, as fhtool's entry: a search field shows "Search".
		search := ui.TextInput(c, &a.keyword).Label(hint).Placeholder(hint).Width(400)
		// The search runs on Tìm kiếm or Enter, as fhtool's, or once
		// cleared: on every key it would go through all the rows.
		cleared := search.Changed() && a.keyword == ""
		if ui.Button(c, "Tìm kiếm").Clicked() || search.Submitted() || column || cleared {
			a.filter()
			c.Invalidate()
		}
	})
}

// start runs the action on the accounts checked, in the order shown.
func (a *app) start(c *ui.Context) {
	var ids []int
	for i := range a.view.Len() {
		if id := a.s.accounts[a.view.Row(i)].ID; a.checked.Has(id) {
			ids = append(ids, id)
		}
	}
	switch {
	case len(ids) == 0:
		c.Toast("Bạn chưa chọn tài khoản nào để chạy!")
		return
	case a.action == "":
		c.Toast("Bạn chưa chọn hành động để chạy!")
		return
	}
	r := run(ids, int(a.loops), int(a.threads), fakeWork(a.action))
	a.run, a.stopping = r, false
	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				a.update(func() { a.apply(r) })
			case <-r.done:
				a.update(func() {
					a.apply(r)
					a.run, a.stopping = nil, false
				})
				return
			}
		}
	}()
}

// apply shows the results r's workers gave since the last batch.
func (a *app) apply(r *runner) {
	for _, res := range r.take() {
		i, ok := a.s.index[res.id]
		if !ok {
			continue // deleted meanwhile
		}
		x := &a.s.accounts[i]
		x.state, x.Status = res.state, res.status
		if res.at != 0 {
			x.LastRunAt, x.IP = res.at, res.ip
		}
		a.view.Changed(i)
	}
}

func main() {
	rows := flag.Int("rows", 100_000, "how many accounts to make")
	flag.Parse()
	a := newApp(*rows)
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Quản lý tài khoản", Width: 1280, Height: 800, Content: ui.View(a.build)})
		a.update = w.Update
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

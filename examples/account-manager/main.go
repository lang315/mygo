// Account-manager rebuilds the account manager of fhtool, a GTK tool that
// runs jobs on many accounts, in native UI: 100,000 made-up accounts in
// folders, in a table that sorts, searches and checks them, with a menu
// on its rows, and workers whose results the rows show as they come.
//
//	go run ./examples/account-manager
//	go run ./examples/account-manager -rows 1000000
//
// MYGO_FRAME_STATS=8 logs the frames slower than 8 ms. -stress changes
// that many rows a second, as a busy run would, and logs how the app
// keeps up once a second, as -stats does without it:
//
//	go run ./examples/account-manager -stress 10000
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
	stress          *runner // the stress run going on, nil when none
	stats           *stats  // what -stats logs, nil without it
	filtered        search  // the search the view shows the rows of
	// update runs fn on the main thread, then builds a frame:
	// Window.Update, or a test's.
	update func(fn func())
}

func newApp(rows int) *app {
	a := &app{s: newStore(rows, 1), view: ui.NewTableView(compare()), cursor: -1, column: "UID", threads: 10}
	a.folder = a.s.folders[0].Name
	a.filtered.folder = -1 // no folder: the first filter is given
	a.filter()
	return a
}

func (a *app) build(c *ui.Context) {
	if a.stats != nil {
		start := time.Now()
		defer func() {
			a.stats.frame(start, time.Now())
			a.logStats()
		}()
	}
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
			// Sync places a row appended, no need to place them all again.
			a.s.add(cur.ID)
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
		// Results reach the rows by ID, so only deleting a folder, which
		// moves its rows, waits for the run to end.
		if ui.Button(c, "Thêm thư mục").Clicked() {
			a.folder = a.s.addFolder().Name
			a.showFolder()
		}
		if ui.Button(c, "Đổi tên").Disabled(cur.ID == trash).Clicked() {
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

// search is what decides the rows shown: a folder, and the keyword sought
// in a column.
type search struct {
	folder          int
	column, keyword string
}

// filter shows the accounts of the folder shown that the search finds.
// It does nothing when they are the rows shown already: a filter given to
// the view sorts all rows again, a second at a million.
func (a *app) filter() {
	f, _ := a.s.folderByName(a.folder)
	s := search{f.ID, a.column, strings.TrimSpace(a.keyword)}
	if s.keyword == "" {
		s.column = "" // the column searched keeps all rows alike
	}
	if s == a.filtered {
		return
	}
	a.filtered = s
	a.view.SetFilter(keep(s))
}

// keep tells whether an account is in the folder of s and s finds it.
func keep(s search) func(*account) bool {
	match := matcher(columnIndex(s.column), s.keyword)
	return func(x *account) bool { return x.Folder == s.folder && match(x) }
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
		if len(dups) > 0 {
			a.gone(c, dups)
		}
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

// gone shows the accounts ids left the folder shown: the view places again
// only the rows moved, and all of them after rows were removed.
func (a *app) gone(c *ui.Context, ids []int) {
	for _, id := range ids {
		a.checked.Remove(id)
		a.chosen.Remove(id)
		if i, ok := a.s.index[id]; ok {
			a.view.Changed(i)
		} else {
			a.view.Reset()
		}
	}
	c.Invalidate()
}

func (a *app) footer(c *ui.Context) {
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Textf(c, "Tổng số dòng: %d", a.view.Len())
		ui.Textf(c, "Số dòng đang bôi đen: %d", a.chosen.Len())
		ui.Textf(c, "Đã chọn: %d", a.checked.Len())
		ui.Spacer(c)
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
	case len(ids) == 0 && a.checked.Len() > 0:
		c.Toast("Các dòng đã chọn đang bị ẩn bởi tìm kiếm!")
		return
	case len(ids) == 0:
		c.Toast("Bạn chưa chọn tài khoản nào để chạy!")
		return
	case a.action == "":
		c.Toast("Bạn chưa chọn hành động để chạy!")
		return
	}
	r := run(ids, int(a.loops), int(a.threads), fakeWork(a.action))
	a.run, a.stopping = r, false
	a.follow(r, func() { a.run, a.stopping = nil, false })
}

// startStress changes rate rows a second until a.stress is cancelled.
func (a *app) startStress(rate int) {
	r := stress(a.s.nextID, rate, 2)
	a.stress = r
	a.follow(r, func() { a.stress = nil })
}

// follow applies r's results to the rows ten times a second, then, once r
// is done, runs end on the main thread.
func (a *app) follow(r *runner, end func()) {
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
					end()
				})
				return
			}
		}
	}()
}

// apply shows the results r's workers gave since the last batch.
func (a *app) apply(r *runner) {
	start := time.Now()
	batch := r.take()
	for _, res := range batch {
		i, ok := a.s.index[res.id]
		if !ok {
			continue // deleted meanwhile
		}
		x := &a.s.accounts[i]
		x.state, x.Status = res.state, res.status
		if res.at != 0 {
			x.LastRunAt, x.IP = res.at, res.ip
		}
		x.Friends, x.Groups = max(0, x.Friends+res.friends), max(0, x.Groups+res.groups)
		if res.note != "" {
			x.Note = res.note
		}
		a.view.Changed(i)
	}
	if a.stats != nil {
		a.stats.applied(len(batch), time.Since(start))
		a.logStats()
	}
}

// logStats logs the line of the second just ended, if one did.
func (a *app) logStats() {
	if line := a.stats.flush(time.Now(), a); line != "" {
		log.Print(line)
	}
}

func main() {
	rows := flag.Int("rows", 100_000, "how many accounts to make")
	rate := flag.Int("stress", 0, "how many rows to change a second, as a busy run would")
	logStats := flag.Bool("stats", false, "log how the app keeps up once a second")
	flag.Parse()
	a := newApp(*rows)
	if *logStats || *rate > 0 {
		a.stats = newStats(time.Now())
	}
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Quản lý tài khoản", Width: 1280, Height: 800, Content: ui.View(a.build)})
		a.update = w.Update
		if *rate > 0 {
			a.startStress(*rate)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

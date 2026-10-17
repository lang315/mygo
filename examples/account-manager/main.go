// Account-manager rebuilds the account manager of fhtool, a GTK tool that
// runs jobs on many accounts, in native UI: 100,000 made-up accounts in
// folders, in a table that sorts, searches and checks them, with a menu
// on its rows, and workers whose results the rows show as they come. Tabs
// show the Ads and Email columns of the same accounts, and a menu bar has
// the actions on accounts and on folders.
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
	s     *store
	table ui.DataTableState[account, int] // the accounts of the folder shown that the search finds
	// The other tabs' tables: the same accounts, other columns, and their
	// own checks, highlights, sort and search.
	ads, email ui.DataTableState[account, int]
	tab        int    // the tab shown
	folder     string // the folder shown
	// searches are the column and keyword typed in each tab; those of the
	// tab shown are column and keyword below.
	searches [len(tabColumns)]struct{ column, keyword string }
	// The search typed: the title of the column searched, and what to find.
	column, keyword string
	action          string
	loops, threads  float64
	run             *runner // the run going on, nil when none
	stopping        bool
	stress          *runner // the stress run going on, nil when none
	stats           *stats  // what -stats logs, nil without it
	// pending is the work the menu bar's clicks queued, for the next frame to
	// run with a Context.
	pending []func(c *ui.Context)
	// update runs fn on the main thread, then builds a frame:
	// Window.Update, or a test's.
	update func(fn func())
}

// newApp does not search or read the table, so the table is not started
// before the first frame: a test can still change its columns.
func newApp(rows int) *app {
	a := &app{s: newStore(rows, 1), column: "UID", threads: 10}
	a.folder = a.s.folders[0].Name
	a.filterFolder()
	for i := range a.searches {
		a.searches[i].column = "UID"
	}
	for i, t := range a.tables() {
		t.Key = func(x *account) int { return x.ID }
		t.Columns = dataColumns(tabColumns[i])
		// The search bar is the footer's, with fhtool's texts.
		t.NoSearchBar = true
		t.Filter = func(x *account) bool { return x.Folder == t.FilterKey }
		t.Copy = func(rows []*account) string { return copyText(rows, colUID, colPassword, colTwoFA) }
	}
	return a
}

// tables are the tabs' tables, in the order of the strip.
func (a *app) tables() [len(tabColumns)]*ui.DataTableState[account, int] {
	return [...]*ui.DataTableState[account, int]{&a.table, &a.ads, &a.email}
}

// shown is the table of the tab shown.
func (a *app) shown() *ui.DataTableState[account, int] { return a.tables()[a.tab] }

// selectTab shows tab i: the search typed goes with the tab left, and that
// of tab i comes back.
func (a *app) selectTab(i int) {
	a.searches[a.tab].column, a.searches[a.tab].keyword = a.column, a.keyword
	a.tab = i
	a.column, a.keyword = a.searches[i].column, a.searches[i].keyword
	// Rows changed while it was hidden were not told to it.
	a.shown().Reset()
}

// changed says that the account at index i of the store changed. Only the
// tab shown is told: a table that is not built never syncs, so the rows
// named to it would pile up. The others place all rows again once shown.
func (a *app) changed(i int) {
	for t, tb := range a.tables() {
		if t == a.tab {
			tb.Changed(i)
		} else {
			tb.Reset()
		}
	}
}

// reset says that accounts were removed.
func (a *app) reset() {
	for _, tb := range a.tables() {
		tb.Reset()
	}
}

func (a *app) build(c *ui.Context) {
	if a.stats != nil {
		start := time.Now()
		defer func() {
			a.stats.frame(start, time.Now())
			a.logStats()
		}()
	}
	if pending := a.pending; len(pending) > 0 {
		a.pending = nil
		for _, fn := range pending {
			fn(c)
		}
		c.Invalidate()
	}
	ui.Column(c).Fill().Padding(10).Gap(10).Children(func() {
		a.toolbar(c)
		a.tabStrip(c)
		a.showTable(c)
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
			a.addAccount()
		}
		ui.Select(c, &a.action, actions).Label("Hành động").Width(190)
		ui.NumberInput(c, &a.loops, 0, 1_000_000, 1).Label("Số lần lặp").Tooltip("Số lần lặp")
		ui.NumberInput(c, &a.threads, 1, 1_000_000, 1).Label("Số luồng").Tooltip("Số luồng")
		if !busy {
			if ui.PrimaryButton(c, "Start").Clicked() {
				a.start(c)
			}
		} else if ui.Button(c, "Stop").Disabled(a.stopping).Clicked() {
			a.stop()
		}
		ui.Spacer(c)
		if ui.Select(c, &a.folder, a.s.names()).Label("Thư mục").Width(220).Changed() {
			a.showFolder()
		}
		// Results reach the rows by ID, so only deleting a folder, which
		// moves its rows, waits for the run to end.
		if ui.Button(c, "Thêm thư mục").Clicked() {
			a.addFolder()
		}
		if ui.Button(c, "Đổi tên").Disabled(cur.ID == trash).Clicked() {
			a.renameFolder()
		}
		if ui.Button(c, "Xóa thư mục").Disabled(busy || cur.ID == trash).Clicked() {
			a.deleteFolder(c)
		}
	})
}

// addAccount appends one generated account to the folder shown. Sync places
// a row appended, no need to place them all again.
func (a *app) addAccount() {
	cur, _ := a.s.folderByName(a.folder)
	a.s.add(cur.ID)
}

func (a *app) addFolder() {
	a.folder = a.s.addFolder().Name
	a.showFolder()
}

// renameFolder renames the folder shown, but the trash.
func (a *app) renameFolder() {
	if cur, _ := a.s.folderByName(a.folder); cur.ID != trash {
		a.folder = a.s.renameFolder(cur.ID)
	}
}

// deleteFolder deletes the folder shown, but the trash, and none while a run
// goes on: its results reach the rows by ID.
func (a *app) deleteFolder(c *ui.Context) {
	cur, _ := a.s.folderByName(a.folder)
	switch {
	case cur.ID == trash:
		c.Toast("Không xóa được thùng rác!")
	case a.run != nil:
		c.Toast("Đang chạy, chưa xóa được thư mục!")
	default:
		a.s.deleteFolder(cur.ID)
		a.folder = a.s.folders[0].Name
		a.showFolder()
	}
}

// stop asks the run going on to end.
func (a *app) stop() {
	if a.run != nil && !a.stopping {
		a.stopping = true
		a.run.cancel()
	}
}

// begin runs action on the accounts checked in the tab shown, as Start does
// once an action is chosen.
func (a *app) begin(c *ui.Context, action string) {
	if a.run != nil {
		c.Toast("Đang chạy!")
		return
	}
	a.action = action
	a.start(c)
}

// later runs fn in the next frame, with a Context: the toasts and the redraw
// need one, and a menu's Click has none. Window.Update builds that frame, so
// a click with no input in the window runs it too.
func (a *app) later(fn func(c *ui.Context)) {
	a.update(func() { a.pending = append(a.pending, fn) })
}

// menu is the menu bar: fhtool's menus for the actions on accounts and for
// folders, between the usual ones.
func (a *app) menu() []*mygo.MenuItem {
	item := func(label string, fn func(c *ui.Context)) *mygo.MenuItem {
		return &mygo.MenuItem{Label: label, Click: func(*mygo.MenuItem, *mygo.Window) { a.later(fn) }}
	}
	accounts := []*mygo.MenuItem{item("Thêm tài khoản", func(*ui.Context) { a.addAccount() }), mygo.Separator()}
	for _, act := range actions {
		accounts = append(accounts, item(act, func(c *ui.Context) { a.begin(c, act) }))
	}
	accounts = append(accounts, mygo.Separator(), item("Dừng", func(*ui.Context) { a.stop() }))
	return []*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "Tài khoản", Submenu: accounts},
		{Label: "Thư mục", Submenu: []*mygo.MenuItem{
			item("Thêm thư mục", func(*ui.Context) { a.addFolder() }),
			item("Đổi tên", func(*ui.Context) { a.renameFolder() }),
			item("Xóa thư mục", a.deleteFolder),
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}
}

// tabStrip shows the tabs and switches to the one clicked.
func (a *app) tabStrip(c *ui.Context) {
	tab := a.tab
	if ui.Tabs(c, &tab, tabLabels...).Changed() {
		a.selectTab(tab)
	}
}

// showFolder shows the folder a.folder names afresh, none of its
// accounts checked or highlighted, as fhtool loads a folder.
func (a *app) showFolder() {
	for _, tb := range a.tables() {
		tb.Checked.Clear()
		tb.Chosen.Clear()
	}
	a.filterFolder()
}

// filterFolder has the table show the accounts of the folder a.folder names.
// The table does nothing when its key is the one it has: the rows are
// filtered again only when the folder changes.
func (a *app) filterFolder() {
	f, _ := a.s.folderByName(a.folder)
	for _, tb := range a.tables() {
		tb.FilterKey = f.ID
	}
}

// filter shows the accounts of the folder shown that the search finds. The
// table does nothing when they are the rows shown already: a search goes
// through all rows.
func (a *app) filter() { a.shown().SetSearch(a.column, a.keyword) }

func (a *app) showTable(c *ui.Context) {
	t := a.shown()
	t.RowMenu = func(m *ui.Menu, rows []*account) { a.rowMenu(c, m, rows) }
	ui.DataTable(c, t, a.s.accounts).Grow(1).Key(a.tab)
}

// copyText is columns cols of rows, | between them, a line each.
func copyText(rows []*account, cols ...int) string {
	var b strings.Builder
	for i, x := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
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
func (a *app) rowMenu(c *ui.Context, m *ui.Menu, rows []*account) {
	ids := make([]int, len(rows))
	for i, x := range rows {
		ids[i] = x.ID
	}
	m.Submenu("Chọn/Bỏ chọn", func(m *ui.Menu) {
		if m.Item("Chọn").Chosen() {
			for _, id := range ids {
				a.shown().Checked.Add(id)
			}
		}
		if m.Item("Bỏ chọn").Chosen() {
			for _, id := range ids {
				a.shown().Checked.Remove(id)
			}
		}
	})
	m.Submenu("Copy", func(m *ui.Menu) {
		for _, it := range copies {
			if m.Item(it.label).Chosen() {
				c.WriteClipboard(copyText(rows, it.cols...))
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

// forget drops the accounts ids from every tab's checks and highlights, and
// has the tables place again only the rows moved, and all of them after rows
// were removed.
func (a *app) forget(ids []int) {
	for _, id := range ids {
		for _, tb := range a.tables() {
			tb.Checked.Remove(id)
			tb.Chosen.Remove(id)
		}
		if i, ok := a.s.index[id]; ok {
			a.changed(i)
		} else {
			a.reset()
		}
	}
}

// gone shows the accounts ids left the folder shown.
func (a *app) gone(c *ui.Context, ids []int) {
	a.forget(ids)
	c.Invalidate()
}

func (a *app) footer(c *ui.Context) {
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		t := a.shown()
		ui.Textf(c, "Tổng số dòng: %d", t.Len())
		ui.Textf(c, "Số dòng đang bôi đen: %d", t.Chosen.Len())
		ui.Textf(c, "Đã chọn: %d", t.Checked.Len())
		ui.Spacer(c)
		column := ui.Select(c, &a.column, tabTitles[a.tab]).Label("Cột tìm kiếm").Width(170).Changed()
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
	t := a.shown()
	var ids []int
	for i := range t.Len() {
		if id := t.At(i).ID; t.Checked.Has(id) {
			ids = append(ids, id)
		}
	}
	switch {
	case len(ids) == 0 && t.Checked.Len() > 0:
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
		a.changed(i)
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
		mygo.App.SetMenu(mygo.NewMenu(a.menu()))
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

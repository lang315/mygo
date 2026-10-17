package main

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// state is where an account's last run is, which colors its status.
type state uint8

const (
	idle state = iota
	running
	succeeded
	failed
)

// account is a row of the table: an account of fhtool's, made up.
type account struct {
	ID, Folder                                            int
	UID, Password, TwoFA, Email, RecoveryEmail, Proxy, IP string
	LastRunAt                                             int64
	Status                                                string
	state                                                 state
	Cookie, Name                                          string
	Friends, Groups                                       int
	Country, Resource                                     string
	CreatedAt, BackupAt                                   int64
	Note                                                  string
	BM, TKQC, EmailPassword, RefreshID, ClientID          string // the Ads and Email tabs' columns
}

// The columns, in the order the table shows them at first.
const (
	colUID = iota
	colPassword
	colTwoFA
	colEmail
	colRecovery
	colProxy
	colIP
	colLastRun
	colStatus
	colCookie
	colName
	colFriends
	colGroups
	colCountry
	colResource
	colCreated
	colBackup
	colNote
	colBM
	colTKQC
	colEmailPassword
	colRefreshID
	colClientID
)

// columns are the table's, titled as fhtool's; compare() has a sort for each.
var columns = []ui.TableColumn{
	{Title: "UID", Width: 150},
	{Title: "Mật khẩu", Width: 110},
	{Title: "2FA", Width: 150},
	{Title: "Email", Width: 210},
	{Title: "Email khôi phục", Width: 210},
	{Title: "Proxy", Width: 170},
	{Title: "IP", Width: 120},
	{Title: "Lần chạy cuối", Width: 150},
	{Title: "Trạng thái", Width: 170},
	{Title: "Cookie", Width: 240},
	{Title: "Tên", Width: 150},
	{Title: "Bạn bè", Width: 70, Align: ui.End},
	{Title: "Nhóm", Width: 60, Align: ui.End},
	{Title: "Quốc Gia", Width: 80},
	{Title: "Tài Nguyên", Width: 100},
	{Title: "Năm tạo", Width: 70, Align: ui.End},
	{Title: "Ngày backup", Width: 100},
	{Title: "Ghi Chú", Width: 160},
}

// extraColumns are the columns only the Ads and Email tabs show, in the order
// of colBM on.
var extraColumns = []ui.TableColumn{
	{Title: "BM", Width: 120},
	{Title: "TKQC", Width: 140},
	{Title: "Mật khẩu email", Width: 150},
	{Title: "RefreshID", Width: 300},
	{Title: "ClientID", Width: 280},
}

// allColumns are the columns of every tab, indexed by the col constants.
var allColumns = append(slices.Clone(columns), extraColumns...)

// The tabs, in the order the strip shows them.
const (
	tabAccounts = iota
	tabAds
	tabEmail
)

// tabLabels are the tabs' labels.
var tabLabels = []string{"Tài khoản", "Ads", "Email"}

// tabColumns are the columns each tab shows, as indexes of allColumns:
// fhtool's.
var tabColumns = [...][]int{
	tabAccounts: {colUID, colPassword, colTwoFA, colEmail, colRecovery, colProxy, colIP, colLastRun, colStatus, colCookie, colName, colFriends, colGroups, colCountry, colResource, colCreated, colBackup, colNote},
	tabAds:      {colUID, colPassword, colTwoFA, colCookie, colStatus, colBM, colTKQC},
	tabEmail:    {colUID, colEmail, colEmailPassword, colStatus, colRecovery, colRefreshID, colClientID},
}

// tabTitles are the columns' titles of each tab, for the column searched.
var tabTitles = func() (t [len(tabColumns)][]string) {
	for i, cols := range tabColumns {
		for _, col := range cols {
			t[i] = append(t[i], allColumns[col].Title)
		}
	}
	return t
}()

// cellText is what column col shows of a: the cells, the search and the
// copies read it.
func cellText(a *account, col int) string {
	switch col {
	case colUID:
		return a.UID
	case colPassword:
		return a.Password
	case colTwoFA:
		return a.TwoFA
	case colEmail:
		return a.Email
	case colRecovery:
		return a.RecoveryEmail
	case colProxy:
		return a.Proxy
	case colIP:
		return a.IP
	case colLastRun:
		return day(a.LastRunAt, "02/01/2006 15:04:05")
	case colStatus:
		return a.Status
	case colCookie:
		return a.Cookie
	case colName:
		return a.Name
	case colFriends:
		return strconv.Itoa(a.Friends)
	case colGroups:
		return strconv.Itoa(a.Groups)
	case colCountry:
		return a.Country
	case colResource:
		return a.Resource
	case colCreated:
		return day(a.CreatedAt, "2006")
	case colBackup:
		return day(a.BackupAt, "02/01/2006")
	case colNote:
		return a.Note
	case colBM:
		return a.BM
	case colTKQC:
		return a.TKQC
	case colEmailPassword:
		return a.EmailPassword
	case colRefreshID:
		return a.RefreshID
	case colClientID:
		return a.ClientID
	}
	return ""
}

// day formats the unix time t by layout, and 0 as nothing.
func day(t int64, layout string) string {
	if t == 0 {
		return ""
	}
	return time.Unix(t, 0).Format(layout)
}

// compare orders the rows by each column of every tab, for the tables'
// views: by the text shown, but numbers and times by value.
func compare() map[string]func(a, b *account) int {
	m := make(map[string]func(a, b *account) int, len(allColumns))
	for col, c := range allColumns {
		m[c.Title] = func(a, b *account) int { return strings.Compare(cellText(a, col), cellText(b, col)) }
	}
	m[columns[colLastRun].Title] = func(a, b *account) int { return cmp.Compare(a.LastRunAt, b.LastRunAt) }
	m[columns[colFriends].Title] = func(a, b *account) int { return cmp.Compare(a.Friends, b.Friends) }
	m[columns[colGroups].Title] = func(a, b *account) int { return cmp.Compare(a.Groups, b.Groups) }
	m[columns[colCreated].Title] = func(a, b *account) int { return cmp.Compare(a.CreatedAt, b.CreatedAt) }
	m[columns[colBackup].Title] = func(a, b *account) int { return cmp.Compare(a.BackupAt, b.BackupAt) }
	return m
}

// dataColumns are the columns of a DataTable showing cols, as indexes of
// allColumns: the cells, the sorts and the search read cellText.
func dataColumns(cols []int) []ui.DataColumn[account] {
	sorts := compare()
	out := make([]ui.DataColumn[account], len(cols))
	for i, col := range cols {
		c := allColumns[col]
		out[i] = ui.DataColumn[account]{
			TableColumn: c,
			Text:        func(a *account) string { return cellText(a, col) },
			Compare:     sorts[c.Title],
		}
		if col == colStatus {
			out[i].Cell = statusCell
		}
	}
	return out
}

// statusCell shows the status colored by where the account's run is.
func statusCell(c *ui.Context, x *account) {
	cell := ui.Text(c, x.Status).SingleLine()
	switch x.state {
	case running:
		cell.TextColor(ui.RGB(250, 204, 21)).TextBackground(ui.RGB(0, 0, 0)).Bold()
	case failed:
		cell.TextColor(c.Theme().Danger)
	case succeeded:
		cell.TextColor(c.Theme().Success)
	}
}

var (
	lastNames  = []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Phan", "Vũ", "Đặng", "Bùi", "Đỗ"}
	firstNames = []string{"Anh", "Bình", "Châu", "Dũng", "Giang", "Hà", "Hùng", "Khánh", "Linh", "Minh", "Nam", "Phương", "Quân", "Thảo", "Trang", "Tuấn", "Vy"}
	countries  = []string{"VN", "US", "TH", "ID", "PH", "IN", "BR"}
)

// clientIDs are the few client IDs the made-up email accounts share.
var clientIDs = []string{"9e5f94bc-e8a4-4e73-b8be-63364c29d753", "d3590ed6-52b3-4102-aeff-aad2292ab01c", "1fec8e78-bce4-4aaf-ab81-d7a78c1d1f08"}

const (
	alnum  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
)

// generate makes n accounts in folder 0, their IDs counting from first:
// the same ones for the same seed and first, their UIDs different but for
// one in a thousand.
func generate(n, first int, seed uint64) []account {
	r := rand.New(rand.NewPCG(seed, uint64(first)))
	// The Ads and Email columns come from a source of their own, so the
	// other columns keep the values they had.
	ex := rand.New(rand.NewPCG(seed^0x9e3779b97f4a7c15, uint64(first)))
	pick := func(r *rand.Rand, chars string, n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = chars[r.IntN(len(chars))]
		}
		return string(b)
	}
	out := make([]account, n)
	for i := range out {
		id := first + i
		// IDs apart by 7919 keep the UIDs apart, but for one in a thousand
		// that repeats the UID before it, as an account imported twice, for
		// Lọc trùng UID to find.
		uid := strconv.Itoa(100000000000000 + id*7919 + r.IntN(7919))
		if id%1000 == 500 && i > 0 {
			uid = out[i-1].UID
		}
		a := account{
			ID:        id,
			UID:       uid,
			Password:  pick(r, alnum, 10),
			TwoFA:     pick(r, base32, 32),
			Email:     fmt.Sprintf("%s%d@hotmail.com", pick(r, alnum[:26], 8), r.IntN(10000)),
			Proxy:     fmt.Sprintf("%d.%d.%d.%d:%d", 1+r.IntN(223), r.IntN(256), r.IntN(256), r.IntN(256), 1024+r.IntN(60000)),
			Cookie:    "c_user=" + uid + "; xs=" + pick(r, alnum, 24),
			Name:      lastNames[r.IntN(len(lastNames))] + " " + firstNames[r.IntN(len(firstNames))],
			Friends:   r.IntN(5000),
			Groups:    r.IntN(300),
			Country:   countries[r.IntN(len(countries))],
			CreatedAt: time.Date(2009+r.IntN(16), time.Month(1+r.IntN(12)), 1+r.IntN(28), 0, 0, 0, 0, time.UTC).Unix(),
		}
		if r.IntN(2) == 0 {
			a.RecoveryEmail = pick(r, alnum[:26], 8) + "@gmail.com"
		}
		if r.IntN(3) > 0 {
			a.BackupAt = time.Date(2026, time.Month(1+r.IntN(9)), 1+r.IntN(28), 0, 0, 0, 0, time.UTC).Unix()
		}
		if ex.IntN(3) == 0 {
			a.BM = strconv.Itoa(100000000000000 + ex.IntN(900000000000000))
			a.TKQC = "act_" + strconv.Itoa(100000000000000+ex.IntN(900000000000000))
		}
		a.EmailPassword = pick(ex, alnum, 10)
		a.RefreshID = pick(ex, alnum, 40)
		a.ClientID = clientIDs[ex.IntN(len(clientIDs))]
		out[i] = a
	}
	return out
}

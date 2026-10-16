package main

import (
	"bytes"
	"cmp"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

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
)

// columns are the table's, titled as fhtool's; all of them sort.
var columns = []ui.TableColumn{
	{Title: "UID", Width: 150, Sortable: true},
	{Title: "Mật khẩu", Width: 110, Sortable: true},
	{Title: "2FA", Width: 150, Sortable: true},
	{Title: "Email", Width: 210, Sortable: true},
	{Title: "Email khôi phục", Width: 210, Sortable: true},
	{Title: "Proxy", Width: 170, Sortable: true},
	{Title: "IP", Width: 120, Sortable: true},
	{Title: "Lần chạy cuối", Width: 150, Sortable: true},
	{Title: "Trạng thái", Width: 170, Sortable: true},
	{Title: "Cookie", Width: 240, Sortable: true},
	{Title: "Tên", Width: 150, Sortable: true},
	{Title: "Bạn bè", Width: 70, Align: ui.End, Sortable: true},
	{Title: "Nhóm", Width: 60, Align: ui.End, Sortable: true},
	{Title: "Quốc Gia", Width: 80, Sortable: true},
	{Title: "Tài Nguyên", Width: 100, Sortable: true},
	{Title: "Năm tạo", Width: 70, Align: ui.End, Sortable: true},
	{Title: "Ngày backup", Width: 100, Sortable: true},
	{Title: "Ghi Chú", Width: 160, Sortable: true},
}

// titles are the columns' titles, for the column searched.
var titles = func() []string {
	t := make([]string, len(columns))
	for i, col := range columns {
		t[i] = col.Title
	}
	return t
}()

// columnIndex is the column titled title, -1 for none.
func columnIndex(title string) int {
	for i, col := range columns {
		if col.Title == title {
			return i
		}
	}
	return -1
}

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

// compare orders the rows by each column, for the table's view: by the
// text shown, but numbers and times by value.
func compare() map[string]func(a, b *account) int {
	m := make(map[string]func(a, b *account) int, len(columns))
	for col, c := range columns {
		m[c.Title] = func(a, b *account) int { return strings.Compare(cellText(a, col), cellText(b, col)) }
	}
	m[columns[colLastRun].Title] = func(a, b *account) int { return cmp.Compare(a.LastRunAt, b.LastRunAt) }
	m[columns[colFriends].Title] = func(a, b *account) int { return cmp.Compare(a.Friends, b.Friends) }
	m[columns[colGroups].Title] = func(a, b *account) int { return cmp.Compare(a.Groups, b.Groups) }
	m[columns[colCreated].Title] = func(a, b *account) int { return cmp.Compare(a.CreatedAt, b.CreatedAt) }
	m[columns[colBackup].Title] = func(a, b *account) int { return cmp.Compare(a.BackupAt, b.BackupAt) }
	return m
}

var (
	lastNames  = []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Phan", "Vũ", "Đặng", "Bùi", "Đỗ"}
	firstNames = []string{"Anh", "Bình", "Châu", "Dũng", "Giang", "Hà", "Hùng", "Khánh", "Linh", "Minh", "Nam", "Phương", "Quân", "Thảo", "Trang", "Tuấn", "Vy"}
	countries  = []string{"VN", "US", "TH", "ID", "PH", "IN", "BR"}
)

const (
	alnum  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
)

// generate makes n accounts in folder 0, their IDs counting from first:
// the same ones for the same seed and first, their UIDs different but for
// one in a thousand.
func generate(n, first int, seed uint64) []account {
	r := rand.New(rand.NewPCG(seed, uint64(first)))
	pick := func(chars string, n int) string {
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
			Password:  pick(alnum, 10),
			TwoFA:     pick(base32, 32),
			Email:     fmt.Sprintf("%s%d@hotmail.com", pick(alnum[:26], 8), r.IntN(10000)),
			Proxy:     fmt.Sprintf("%d.%d.%d.%d:%d", 1+r.IntN(223), r.IntN(256), r.IntN(256), r.IntN(256), 1024+r.IntN(60000)),
			Cookie:    "c_user=" + uid + "; xs=" + pick(alnum, 24),
			Name:      lastNames[r.IntN(len(lastNames))] + " " + firstNames[r.IntN(len(firstNames))],
			Friends:   r.IntN(5000),
			Groups:    r.IntN(300),
			Country:   countries[r.IntN(len(countries))],
			CreatedAt: time.Date(2009+r.IntN(16), time.Month(1+r.IntN(12)), 1+r.IntN(28), 0, 0, 0, 0, time.UTC).Unix(),
		}
		if r.IntN(2) == 0 {
			a.RecoveryEmail = pick(alnum[:26], 8) + "@gmail.com"
		}
		if r.IntN(3) > 0 {
			a.BackupAt = time.Date(2026, time.Month(1+r.IntN(9)), 1+r.IntN(28), 0, 0, 0, 0, time.UTC).Unix()
		}
		out[i] = a
	}
	return out
}

// matcher keeps the accounts whose column col matches keyword: a keyword
// with | lists the values to match exactly, as fhtool's UID1|UID2 search,
// another matches the values containing it, case aside. A keyword
// listing nothing keeps all.
func matcher(col int, keyword string) func(a *account) bool {
	keyword = strings.TrimSpace(keyword)
	if strings.Contains(keyword, "|") {
		set := map[string]bool{}
		for v := range strings.SplitSeq(keyword, "|") {
			if v = strings.TrimSpace(v); v != "" {
				set[v] = true
			}
		}
		if len(set) > 0 {
			return func(a *account) bool { return set[cellText(a, col)] }
		}
		keyword = ""
	}
	if keyword == "" {
		return func(*account) bool { return true }
	}
	kw := []byte(strings.ToLower(keyword))
	// The filter runs on one goroutine, so one buffer holds each cell
	// lowered in turn.
	var buf []byte
	return func(a *account) bool {
		buf = appendLower(buf[:0], cellText(a, col))
		return bytes.Contains(buf, kw)
	}
}

// appendLower appends s as strings.ToLower lowers it, without allocating
// for a buffer large enough.
func appendLower(buf []byte, s string) []byte {
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			buf = append(buf, c)
			i++
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		buf = utf8.AppendRune(buf, unicode.ToLower(r))
		i += n
	}
	return buf
}

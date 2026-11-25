package widget

import (
	"image/color"
	"strconv"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// The cases of P2c's plan (badge cells, formats, the empty state), one test
// each, and the review focus's.

func nameCol() Column[acct] {
	return Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true, Filterable: true}
}

func scoreCol() Column[acct] {
	return Column[acct]{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) }, Width: 100, Align: End, Sortable: true}
}

// statusGrid is a grid of ann (30), bob (10) and cat (20) in cols.
func statusGrid(cols ...Column[acct]) *DataGridViewWidget[acct, int] {
	return DataGridView(func(a *acct) int { return a.id }, cols...).
		SetRows([]acct{{1, "ann", 30}, {2, "bob", 10}, {3, "cat", 20}}).Expand()
}

// textOf is the box of the text s: the last node of RoleText named s, the
// text itself inside a badge read as s.
func textOf(t *testing.T, tt *ui.Tester, s string) ui.Rect {
	t.Helper()
	var r ui.Rect
	found := false
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleText && n.Label == s {
			r, found = n.Bounds, true
		}
	}
	if !found { // a text inside a badge, which is read as the badge's name
		return find(t, tt, s)
	}
	return r
}

// cellOf is the box of the grid cell that shows the text s.
func cellOf(t *testing.T, tt *ui.Tester, s string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleCell && n.Label == s { // a cell of a grid that edits is named by its text
			return n.Bounds
		}
	}
	r := textOf(t, tt, s)
	x, y := r.X+r.W/2, r.Y+r.H/2
	for _, n := range tt.Nodes() {
		b := n.Bounds
		if (n.Role == ui.RoleCell || n.Role == ui.RoleRowHeader) && x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H {
			return b
		}
	}
	t.Fatalf("no cell holds %q", s)
	return ui.Rect{}
}

// px is the pixel at (x, y) in DIPs.
func px(tt *ui.Tester, x, y float32) color.RGBA { return tt.Image().RGBAAt(int(x), int(y)) }

// edges are the pixels of a cell box's edges, each the outermost pixel
// inside it: left, right, top and bottom, through its middle, and 1 DIP
// inside its left edge.
func cellEdges(tt *ui.Tester, b ui.Rect) map[string]color.RGBA {
	midX, midY := b.X+b.W/2, b.Y+b.H/2
	return map[string]color.RGBA{
		"left":          px(tt, b.X, midY),
		"1 inside left": px(tt, b.X+1, midY),
		"right":         px(tt, b.X+b.W-1, midY),
		"top":           px(tt, b.X+3, b.Y),
		"bottom":        px(tt, b.X+3, b.Y+b.H-1),
		"top middle":    px(tt, midX, b.Y),
	}
}

// fills fails unless every edge pixel of the cell of s is want.
func fills(t *testing.T, tt *ui.Tester, s string, want ui.Color, what string) {
	t.Helper()
	b := cellOf(t, tt, s)
	for name, c := range cellEdges(tt, b) {
		if !nearTol(c, want, 3) {
			t.Errorf("%s: the %s edge of %q's cell %v is %v, want %v", what, name, s, b, c, want)
		}
	}
}

// cellInk is the core of the glyphs of s: the pixel of its box farthest from
// the color of the cell beside it.
func cellInk(t *testing.T, tt *ui.Tester, s string) color.RGBA {
	t.Helper()
	r := textOf(t, tt, s)
	b := cellOf(t, tt, s)
	return inkOf(tt.Image(), r, px(tt, b.X+1, b.Y+b.H/2))
}

// Case 1: a Badge column returning StatusSuccess under Calm draws a pill of
// Success.Subtle with Success.Text, 20 DIPs high inside the row.
func TestGridBadgeCalm(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	col := nameCol()
	col.Badge = func(*acct) Status { return StatusSuccess }
	tt := tester(t, statusGrid(col, scoreCol()))
	pill := badgeBox(t, tt, "ann")
	if c := px(tt, pill.X+3, pill.Y+pill.H/2); !near(c, tk.Success.Subtle) {
		t.Errorf("the pill's fill is %v, want Success.Subtle %v", c, tk.Success.Subtle)
	}
	if c := darkest(tt, pill); !nearTol(c, tk.Success.Text, 12) {
		t.Errorf("the pill's text is %v, want Success.Text %v", c, tk.Success.Text)
	}
	cell := cellOf(t, tt, "ann")
	if !sameSize(pill.H, 20) || !sameSize(cell.H, tk.RowHeight) || pill.Y < cell.Y || pill.Y+pill.H > cell.Y+cell.H {
		t.Errorf("the pill %v in the cell %v: want 20 high inside a row of %v", pill, cell, tk.RowHeight)
	}
}

// Case 2: in a Calm Compact row of 26 the pill fits, 20 high.
func TestGridBadgeCompact(t *testing.T) {
	useSkinNamed(t, "Calm", true)
	col := nameCol()
	col.Badge = func(*acct) Status { return StatusRunning }
	tt := tester(t, statusGrid(col))
	pill, cell := badgeBox(t, tt, "ann"), cellOf(t, tt, "ann")
	if !sameSize(cell.H, 26) || !sameSize(pill.H, 20) || pill.Y < cell.Y || pill.Y+pill.H > cell.Y+cell.H {
		t.Errorf("the pill %v in the cell %v: want 20 high inside 26", pill, cell)
	}
}

// Case 3: a Badge returning StatusNone draws the plain Text, pixel for
// pixel as a column without Badge, and builds the cell as one: no format's
// box around it.
func TestGridBadgeNoneIsPlain(t *testing.T) {
	calmOff()
	col := nameCol()
	col.Badge = func(*acct) Status { return StatusNone }
	tb := statusGrid(col, scoreCol())
	tt := tester(t, tb)
	if tb.boxes != 0 {
		t.Errorf("%d cells of StatusNone and no format were built in a format's box", tb.boxes)
	}
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	a, b := tt.Image(), plain.Image()
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatalf("a StatusNone badge column differs from a plain one at byte %d", i)
		}
	}
}

// Case 4: StatusRunning with BadgeDetail "Bước 2/5" draws a dot, the pill,
// then the step in TextMuted, read "Live, Bước 2/5". For another status
// BadgeDetail is neither drawn nor called.
func TestGridBadgeRunningDetail(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	status, calls := StatusRunning, 0
	col := Column[acct]{Title: "State", Text: func(*acct) string { return "Live" }, Width: 300,
		Badge:       func(*acct) Status { return status },
		BadgeDetail: func(*acct) string { calls++; return "Bước 2/5" }}
	tb := DataGridView(func(a *acct) int { return a.id }, col).SetRows([]acct{{1, "ann", 1}}).Expand()
	tt := tester(t, tb)
	box := badgeBox(t, tt, "Live, Bước 2/5")
	detail := textOf(t, tt, "Bước 2/5")
	// The dot is 6 DIPs after the pill's padding of 8, before the text.
	if !inkNear(tt, ui.Rect{X: box.X + 6, Y: box.Y, W: 10, H: box.H}, tk.Info.Base, 8) {
		t.Errorf("no dot of Info.Base %v at the pill's start", tk.Info.Base)
	}
	// "Live" and a dot make a pill over 40 DIPs wide; the detail ends the badge.
	if detail.X < box.X+40 || !sameSize(detail.X+detail.W, box.X+box.W) {
		t.Errorf("the detail %v is not after the pill, at the end of the badge %v", detail, box)
	}
	if c := darkest(tt, detail); !nearTol(c, tk.TextMuted, 12) {
		t.Errorf("the detail is %v, want TextMuted %v", c, tk.TextMuted)
	}
	if calls == 0 {
		t.Error("BadgeDetail was not called for StatusRunning")
	}
	status, calls = StatusSuccess, 0
	tb.SetRowFormat(nil) // builds the rows again
	tt.Frame()
	if calls != 0 || tt.HasText("Bước 2/5") {
		t.Errorf("for StatusSuccess BadgeDetail was called %d times, drawn %v", calls, tt.HasText("Bước 2/5"))
	}
	for _, n := range tt.Nodes() {
		if n.Label == "Live, Bước 2/5" {
			t.Error("the detail is read for StatusSuccess")
		}
	}
}

// Case 5: a badge on a selected row keeps its fill and its text.
func TestGridBadgeSelected(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	col := nameCol()
	col.Badge = func(*acct) Status { return StatusSuccess }
	tt := tester(t, statusGrid(col, scoreCol()))
	click(t, tt, "30")
	tt.Frame()
	pill := badgeBox(t, tt, "ann")
	if c := px(tt, pill.X+3, pill.Y+pill.H/2); !near(c, tk.Success.Subtle) {
		t.Errorf("the selected row's pill is %v, want Success.Subtle %v", c, tk.Success.Subtle)
	}
	if c := darkest(tt, pill); !nearTol(c, tk.Success.Text, 12) {
		t.Errorf("the selected row's pill text is %v, want Success.Text %v", c, tk.Success.Text)
	}
	if c := px(tt, pill.X+pill.W+20, pill.Y+pill.H/2); near(c, tk.Success.Subtle) {
		t.Errorf("past the pill is %v: the row is not selected", c)
	}
}

// Case 6: sorting, searching and copying a badge column read Text only:
// the detail is in none of them.
func TestGridBadgeSortSearchCopy(t *testing.T) {
	calmOff()
	col := nameCol()
	col.Badge = func(*acct) Status { return StatusRunning }
	col.BadgeDetail = func(a *acct) string { return "zz" + strconv.Itoa(4-a.id) } // sorts the other way
	tb := statusGrid(col)
	tt := tester(t, tb)
	click(t, tt, "Name")
	tt.Frame()
	inOrder(t, tt, "ann", "bob", "cat")
	tb.Search("Name", "zz")
	tt.Frame()
	if tt.HasText("ann") || tt.HasText("bob") {
		t.Errorf("a search for the detail kept rows: %q", tt.Texts())
	}
	tb.Search("Name", "bo")
	tt.Frame()
	if !tt.HasText("bob") || tt.HasText("ann") {
		t.Errorf("a search of Text: %q", tt.Texts())
	}
	click(t, tt, "bob")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "bob" {
		t.Errorf("copied %q, want Text alone", got)
	}
}

// Case 7: Format {Status: StatusWarning} fills the cell's whole box,
// edge to edge and the row's full height, with Warning.Subtle, and writes in
// Warning.Text: under Calm (a row height metric), without a skin (padding
// above and below) and with SetRowHeight.
func TestGridFormatFillsTheBox(t *testing.T) {
	warn := func(*acct) CellFormat { return CellFormat{Status: StatusWarning} }
	t.Run("Calm", func(t *testing.T) {
		useCalm(t, true)
		t.Cleanup(calmOff)
		tk := skinhook.Get(false)
		col := nameCol()
		col.Format = warn
		tt := tester(t, statusGrid(col, scoreCol()))
		fills(t, tt, "ann", tk.Warning.Subtle, "Calm")
		if c := cellInk(t, tt, "ann"); !nearTol(c, tk.Warning.Text, 12) {
			t.Errorf("the text is %v, want Warning.Text %v", c, tk.Warning.Text)
		}
	})
	for _, h := range []float32{0, 44} {
		t.Run("plain row height "+strconv.Itoa(int(h)), func(t *testing.T) {
			calmOff()
			th := ui.LightTheme()
			col := nameCol()
			col.Format = warn
			tb := statusGrid(col, scoreCol()).SetRowHeight(h)
			tt := tester(t, tb)
			plain := tester(t, statusGrid(nameCol(), scoreCol()).SetRowHeight(h))
			b := cellOf(t, plain, "ann")
			under := px(plain, b.X+1, b.Y+1)
			fills(t, tt, "ann", th.Warning.Alpha(0.14).Over(ui.RGB(under.R, under.G, under.B)), "plain")
			if c := cellInk(t, tt, "ann"); !nearTol(c, th.Text, 12) {
				t.Errorf("the text is %v, want the theme's Text %v", c, th.Text)
			}
		})
	}
}

// Case 8: Fill and Text override a status's, and Bold draws heavier text.
func TestGridFormatFields(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	x, y := ui.Hex("#c0ffee"), ui.Hex("#7a0050")
	for _, c := range []struct {
		name       string
		f          CellFormat
		fill, text ui.Color
	}{
		{"fill", CellFormat{Fill: x}, x, tk.Text},
		{"text", CellFormat{Text: y}, tk.Background, y},
		{"status and fill", CellFormat{Status: StatusDanger, Fill: x}, x, tk.Danger.Text},
	} {
		t.Run(c.name, func(t *testing.T) {
			col := nameCol()
			col.Format = func(*acct) CellFormat { return c.f }
			tt := tester(t, statusGrid(col, scoreCol()))
			fills(t, tt, "ann", c.fill, c.name)
			if got := cellInk(t, tt, "ann"); !nearTol(got, c.text, 12) {
				t.Errorf("the text is %v, want %v", got, c.text)
			}
		})
	}
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Bold: true} }
	bold := tester(t, statusGrid(col, scoreCol()))
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	if b, p := textOf(t, bold, "ann").W, textOf(t, plain, "ann").W; b <= p {
		t.Errorf("bold text is %v wide, plain %v: not bold", b, p)
	}
}

// Case 9: SetRowFormat formats every cell of the row; a column's Format wins
// field by field: a row's Danger and a column's Text y give Danger's fill
// and y.
func TestGridRowFormat(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	y := ui.Hex("#7a0050")
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Text: y} }
	tb := statusGrid(col, scoreCol()).SetRowFormat(func(a *acct) CellFormat {
		if a.id == 1 {
			return CellFormat{Status: StatusDanger}
		}
		return CellFormat{}
	})
	tt := tester(t, tb)
	fills(t, tt, "ann", tk.Danger.Subtle, "Name")
	fills(t, tt, "30", tk.Danger.Subtle, "Score")
	if c := cellInk(t, tt, "ann"); !nearTol(c, y, 12) {
		t.Errorf("the Name cell's text is %v, want the column's %v", c, y)
	}
	if c := cellInk(t, tt, "30"); !nearTol(c, tk.Danger.Text, 12) {
		t.Errorf("the Score cell's text is %v, want the row's Danger.Text %v", c, tk.Danger.Text)
	}
	if b := cellOf(t, tt, "bob"); near(px(tt, b.X, b.Y+b.H/2), tk.Danger.Subtle) {
		t.Error("bob's row, of a zero format, is filled")
	}
	// A field both set: the column's Success wins over the row's Danger, and
	// the row's Bold, which the column leaves unset, stays.
	both := nameCol()
	both.Format = func(*acct) CellFormat { return CellFormat{Status: StatusSuccess} }
	tt = tester(t, statusGrid(both, scoreCol()).SetRowFormat(func(*acct) CellFormat {
		return CellFormat{Status: StatusDanger, Bold: true}
	}))
	fills(t, tt, "ann", tk.Success.Subtle, "the column's status over the row's")
	fills(t, tt, "30", tk.Danger.Subtle, "the row's status")
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	if b, p := textOf(t, tt, "ann").W, textOf(t, plain, "ann").W; b <= p {
		t.Errorf("the Name cell's text is %v wide, plain %v: the row's Bold is lost", b, p)
	}
}

// Case 10: on a selected row the selection's fill and text win over every
// format.
func TestGridFormatSelectionWins(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Status: StatusWarning, Text: ui.Hex("#7a0050"), Bold: true} }
	tt := tester(t, statusGrid(col, scoreCol()).SetRowFormat(func(*acct) CellFormat { return CellFormat{Fill: ui.Hex("#c0ffee")} }))
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	click(t, tt, "ann")
	click(t, plain, "ann")
	tt.Frame()
	plain.Frame()
	for _, s := range []string{"ann", "30"} {
		b := cellOf(t, plain, s)
		for name, want := range cellEdges(plain, b) {
			if got := cellEdges(tt, cellOf(t, tt, s))[name]; got != want {
				t.Errorf("%s's %s edge is %v on a selected row, the selection's is %v", s, name, got, want)
			}
		}
		if got, want := cellInk(t, tt, s), cellInk(t, plain, s); !nearTol(got, ui.RGB(want.R, want.G, want.B), 12) {
			t.Errorf("%s's text on a selected row is %v, the selection's %v", s, got, want)
		}
	}
	if near(px(tt, 700, cellOf(t, tt, "ann").Y+5), tk.Warning.Subtle) {
		t.Error("the selected row shows the format")
	}
}

// Case 11: Format.Text wins over Column.Color; without it Column.Color
// colors the text as before.
func TestGridFormatOverColor(t *testing.T) {
	calmOff()
	blue := ui.RGB(0, 0, 200)
	for _, c := range []struct {
		name string
		f    CellFormat
		want ui.Color
	}{{"format text", CellFormat{Text: blue}, blue}, {"bold only", CellFormat{Bold: true}, red}} {
		t.Run(c.name, func(t *testing.T) {
			col := nameCol()
			col.Color = func(*acct) ui.Color { return red }
			col.Format = func(*acct) CellFormat { return c.f }
			tt := tester(t, statusGrid(col))
			if got := cellInk(t, tt, "ann"); !nearTol(got, c.want, 30) {
				t.Errorf("the text is %v, want %v", got, c.want)
			}
		})
	}
}

// Case 12: under HighContrast (FlatStatus) a format's Status draws no fill,
// the family's Text and bold; an explicit Fill still draws.
func TestGridFormatFlatStatus(t *testing.T) {
	useSkinNamed(t, "HighContrast", false)
	tk := skinhook.Get(false)
	if !skinhook.FlatStatus() {
		t.Fatal("HighContrast in use does not set skinhook's FlatStatus")
	}
	col := nameCol()
	col.Format = func(a *acct) CellFormat {
		if a.id == 2 {
			return CellFormat{Status: StatusSuccess, Fill: ui.Hex("#c0ffee")}
		}
		return CellFormat{Status: StatusSuccess}
	}
	tt := tester(t, statusGrid(col, scoreCol()))
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	b := cellOf(t, plain, "ann")
	under := px(plain, b.X+1, b.Y+b.H/2)
	fills(t, tt, "ann", ui.RGB(under.R, under.G, under.B), "flat")
	if c := cellInk(t, tt, "ann"); !nearTol(c, tk.Success.Text, 12) {
		t.Errorf("the text is %v, want Success.Text %v", c, tk.Success.Text)
	}
	if b, p := textOf(t, tt, "ann").W, textOf(t, plain, "ann").W; b <= p {
		t.Errorf("the text is %v wide, plain %v: not bold", b, p)
	}
	fills(t, tt, "bob", ui.Hex("#c0ffee"), "an explicit fill")
}

// Case 13: a retained row does not call Format, Badge, BadgeDetail or the
// row's format again; UpdateRow calls them again for that row only.
func TestGridStatusRetainedRows(t *testing.T) {
	calmOff()
	format, badge, detail, row := map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{}
	col := nameCol()
	col.Format = func(a *acct) CellFormat { format[a.id]++; return CellFormat{Bold: true} }
	col.Badge = func(a *acct) Status { badge[a.id]++; return StatusRunning }
	col.BadgeDetail = func(a *acct) string { detail[a.id]++; return "Bước 2/5" }
	tb := statusGrid(col, scoreCol()).SetRowFormat(func(a *acct) CellFormat { row[a.id]++; return CellFormat{Status: StatusInfo} })
	tt := tester(t, tb)
	counts := func() [4][4]int {
		var c [4][4]int
		for id := 1; id <= 3; id++ {
			c[id] = [4]int{format[id], badge[id], detail[id], row[id]}
		}
		return c
	}
	first := counts()
	for id := 1; id <= 3; id++ {
		if first[id][0] != 1 || first[id][1] != 1 || first[id][2] != 1 || first[id][3] == 0 {
			t.Errorf("row %d: Format, Badge, BadgeDetail and the row's format called %v times, want once each", id, first[id])
		}
	}
	for range 3 {
		tt.Frame()
	}
	if c := counts(); c != first {
		t.Errorf("retained rows called the functions again: %v, then %v", first, c)
	}
	tb.UpdateRow(1, func(a *acct) { a.score++ })
	tt.Frame()
	c := counts()
	if c[2] != first[2] || c[3] != first[3] {
		t.Errorf("updating ann called bob's or cat's: %v, then %v", first, c)
	}
	if c[1][0] != first[1][0]+1 || c[1][1] != first[1][1]+1 || c[1][2] != first[1][2]+1 || c[1][3] <= first[1][3] {
		t.Errorf("updating ann: %v, then %v: want each once more", first[1], c[1])
	}
}

// Case 14: an empty state shows centered over the body of a grid with no row,
// or none its search keeps, under the headers, which stay and sort; a row
// takes it away, and so does nil.
func TestGridEmptyState(t *testing.T) {
	calmOff()
	tb := DataGridView(func(a *acct) int { return a.id }, nameCol(), scoreCol()).Expand().
		SetEmptyState(EmptyState("Chưa có tài khoản", "Thêm một tài khoản để bắt đầu."))
	tt := tester(t, tb)
	if !tt.HasText("Chưa có tài khoản") || !tt.HasText("Name") {
		t.Fatalf("an empty grid shows %q, want its headers and the empty state", tt.Texts())
	}
	title, head := textOf(t, tt, "Chưa có tài khoản"), nodeOf(t, tt, ui.RoleColumnHeader, "Name").Bounds
	body := ui.Rect{X: 0, Y: head.Y + head.H, W: 800, H: 600 - head.Y - head.H}
	if title.Y <= head.Y+head.H || title.X+title.W/2 < 380 || title.X+title.W/2 > 420 || title.Y < body.Y+body.H/2-80 || title.Y > body.Y+body.H/2+40 {
		t.Errorf("the title %v is not centered in the body %v", title, body)
	}
	click(t, tt, "Name")         // the header still sorts under the empty state
	tb.AddRow(acct{2, "bob", 1}) // one row is enough to take it away
	tt.Frame()
	if tt.HasText("Chưa có tài khoản") {
		t.Error("the empty state shows over one row")
	}
	tb.AddRow(acct{1, "ann", 2})
	tt.Frame()
	if tt.HasText("Chưa có tài khoản") {
		t.Error("the empty state shows over rows")
	}
	inOrder(t, tt, "ann", "bob")
	tb.Search("Name", "bo") // keeps exactly one row
	tt.Frame()
	if tt.HasText("Chưa có tài khoản") || !tt.HasText("bob") || tt.HasText("ann") {
		t.Errorf("a search that keeps one row shows %q, want bob alone and no empty state", tt.Texts())
	}
	tb.Search("Name", "zzz")
	tt.Frame()
	if !tt.HasText("Chưa có tài khoản") {
		t.Errorf("a search that keeps no row shows %q, want the empty state", tt.Texts())
	}
	tb.SetEmptyState(nil)
	tt.Frame()
	if tt.HasText("Chưa có tài khoản") {
		t.Error("SetEmptyState(nil) still draws it")
	}
}

// Case 15: a grid with no Badge, Format or SetRowFormat has no status hook;
// a Badge or Format column, or SetRowFormat, sets it.
func TestGridStatusHookOnlyWhenUsed(t *testing.T) {
	if tb := statusGrid(nameCol(), scoreCol()); tb.statusHook != nil {
		t.Error("a plain grid has a status hook")
	}
	b, f := nameCol(), nameCol()
	b.Badge = func(*acct) Status { return StatusNone }
	f.Format = func(*acct) CellFormat { return CellFormat{} }
	for name, tb := range map[string]*DataGridViewWidget[acct, int]{
		"Badge":        statusGrid(b),
		"Format":       statusGrid(f),
		"SetRowFormat": statusGrid(nameCol()).SetRowFormat(func(*acct) CellFormat { return CellFormat{} }),
	} {
		if tb.statusHook == nil {
			t.Errorf("a grid with %s has no status hook", name)
		}
	}
}

// Review focus 1: the fill follows its cell into the index column, which is
// fixed, and into a column moved by the layout.
func TestGridFormatFixedAndMoved(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Status: StatusSuccess} }
	tb := statusGrid(col, scoreCol()).IndexColumn().
		SetRowFormat(func(*acct) CellFormat { return CellFormat{Status: StatusWarning} }).
		SetColumnsLayout(ui.TableLayout{Order: []string{"#", "Score", "Name"}})
	tt := tester(t, tb)
	if n, s := cellOf(t, tt, "ann"), cellOf(t, tt, "30"); n.X <= s.X {
		t.Fatalf("Name at %v is not after Score at %v", n.X, s.X)
	}
	fills(t, tt, "ann", tk.Success.Subtle, "the moved Name")
	fills(t, tt, "30", tk.Warning.Subtle, "Score")
	fills(t, tt, "1", tk.Warning.Subtle, "the index column")
}

// Ours: a format moves no text: a Start and an End column's texts are where
// an unformatted grid's are, with a skin's row height and without, and an
// image column keeps its image.
func TestGridFormatKeepsPlaces(t *testing.T) {
	for _, skin := range []bool{false, true} {
		useCalm(t, skin)
		t.Cleanup(calmOff)
		fill := func(*acct) CellFormat { return CellFormat{Fill: ui.Hex("#c0ffee")} }
		n, s := nameCol(), scoreCol()
		n.Format, s.Format = fill, fill
		tt := tester(t, statusGrid(n, s))
		plain := tester(t, statusGrid(nameCol(), scoreCol()))
		for _, text := range []string{"ann", "30", "cat", "20"} {
			if a, p := textOf(t, tt, text), textOf(t, plain, text); a != p {
				t.Errorf("skin %v: %q is at %v formatted, %v plain", skin, text, a, p)
			}
		}
	}
}

// Review focus 2: with grid lines the fill stops at the cell's end line.
func TestGridFormatGridLines(t *testing.T) {
	calmOff()
	th := ui.LightTheme()
	x := ui.Hex("#c0ffee")
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Fill: x} }
	tt := tester(t, statusGrid(col, scoreCol()).SetGridLines(true))
	b := cellOf(t, tt, "ann")
	if c := px(tt, b.X+b.W-1, b.Y+b.H/2); !near(c, th.Border) {
		t.Errorf("the cell's end line is %v, want the Border %v", c, th.Border)
	}
	if c := px(tt, b.X+b.W-2, b.Y+b.H/2); !near(c, x) {
		t.Errorf("before the end line is %v, want the fill %v", c, x)
	}
	if c := px(tt, b.X, b.Y+b.H/2); !near(c, x) {
		t.Errorf("the left edge is %v, want the fill %v", c, x)
	}
}

// Review focus 3: a formatted cell edits in place: the editor shows in it
// (its row, selected by the edit, shows the selection, which wins over the
// format), and the fill shows again once the edit ends and the row is no
// longer selected.
func TestGridFormatEditing(t *testing.T) {
	calmOff()
	x := ui.Hex("#c0ffee")
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Fill: x} }
	tb := statusGrid(col, scoreCol()).EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) { a.name = s }})
	tt := tester(t, tb)
	b := cellOf(t, tt, "ann")
	if !tb.BeginEdit(1, "Name") {
		t.Fatal("BeginEdit refused")
	}
	tt.Frame()
	tt.Frame()
	editor := false
	for _, n := range tt.Nodes() {
		if n.Focused && n.Value == "ann" && n.Bounds.X >= b.X && n.Bounds.X+n.Bounds.W <= b.X+b.W {
			editor = true
		}
	}
	if !editor {
		t.Error("no editor of \"ann\" in the formatted cell")
	}
	// The editor is where a plain cell's is: the format's box pads back its
	// negative margins.
	plain := statusGrid(nameCol(), scoreCol()).EditColumn("Name", CellEditor[acct]{Set: func(a *acct, s string) { a.name = s }})
	pt := tester(t, plain)
	plain.BeginEdit(1, "Name")
	pt.Frame()
	pt.Frame()
	focused := func(tt *ui.Tester) ui.Rect {
		for _, n := range tt.Nodes() {
			if n.Focused && n.Value == "ann" {
				return n.Bounds
			}
		}
		return ui.Rect{}
	}
	if a, p := focused(tt), focused(pt); a != p {
		t.Errorf("the formatted cell's editor is at %v, a plain cell's at %v", a, p)
	}
	if c := px(tt, b.X, b.Y+b.H/2); near(c, x) {
		t.Errorf("the edited row's cell is the format's fill %v, not the selection", c)
	}
	tb.CancelEdit()
	click(t, tt, "cat")
	tt.Frame()
	fills(t, tt, "ann", x, "after the edit")
}

// Review focus 4: a translucent fill is drawn over the row's hover color; a
// Format that panics panics the frame as a Text that panics does, and leaves
// the status roles and the lock as they were.
func TestGridFormatTranslucentAndPanic(t *testing.T) {
	calmOff()
	x := ui.RGB(255, 0, 0).Alpha(0.5)
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Fill: x} }
	tt := tester(t, statusGrid(col, scoreCol()))
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	b := cellOf(t, plain, "ann")
	for _, h := range []*ui.Tester{tt, plain} {
		h.Move(b.X+b.W/2, b.Y+b.H/2)
		h.Frame()
	}
	hover := px(plain, b.X+1, b.Y+b.H/2)
	if hover == page(plain) {
		t.Fatal("the hovered row has no color of its own")
	}
	fills(t, tt, "ann", x.Over(ui.RGB(hover.R, hover.G, hover.B)), "translucent over hover")

	boom := nameCol()
	boom.Format = func(*acct) CellFormat { panic("boom") }
	func() {
		defer func() {
			if v := recover(); v != "boom" {
				t.Errorf("the frame recovered %v, want the Format's panic", v)
			}
		}()
		tester(t, statusGrid(boom))
	}()
	if styling != StatusNone {
		t.Errorf("a Format's panic left the status roles at %v", styling)
	}
	if !mu.TryLock() {
		t.Fatal("a Format's panic left the lock held")
	}
	mu.Unlock()
}

// Review focus 5: dark with no skin, a Status fills with the theme's color
// at 0.14 alpha, and the text stays the theme's Text.
func TestGridFormatDarkPlain(t *testing.T) {
	calmOff()
	th := ui.DarkTheme()
	col := nameCol()
	col.Format = func(*acct) CellFormat { return CellFormat{Status: StatusSuccess} }
	tt := tester(t, statusGrid(col, scoreCol()))
	plain := tester(t, statusGrid(nameCol(), scoreCol()))
	for _, h := range []*ui.Tester{tt, plain} {
		h.SetDark(true)
		h.Frame()
	}
	b := cellOf(t, plain, "ann")
	under := px(plain, b.X+1, b.Y+b.H/2)
	fills(t, tt, "ann", th.Success.Alpha(0.14).Over(ui.RGB(under.R, under.G, under.B)), "dark")
	if c := cellInk(t, tt, "ann"); !nearTol(c, th.Text, 12) {
		t.Errorf("the dark text is %v, want Text %v", c, th.Text)
	}
}

// Ours: a badge cell under a look whose border is border-status takes its
// own status's border (the status roles resolve to the cell's status); on
// a selected row a running badge's detail takes the selection's text, which
// TextMuted may not read on.
func TestGridBadgeLookAndSelectedDetail(t *testing.T) {
	useLooks(t, map[string]string{"Badge": "rounded-sm border border-status"})
	tk := skinhook.Get(false)
	col := Column[acct]{Title: "State", Text: func(a *acct) string { return a.name }, Width: 300,
		Badge:       func(a *acct) Status { return Status(a.score) },
		BadgeDetail: func(*acct) string { return "Bước 2/5" }}
	tb := DataGridView(func(a *acct) int { return a.id }, col).
		SetRows([]acct{{1, "ann", int(StatusWarning)}, {2, "bob", int(StatusRunning)}}).Expand()
	tt := tester(t, tb)
	box := badgeBox(t, tt, "ann")
	if c := px(tt, box.X, box.Y+box.H/2); !near(c, tk.Warning.Border) {
		t.Errorf("a Warning badge cell's edge is %v, want Warning.Border %v", c, tk.Warning.Border)
	}
	if styling != StatusNone {
		t.Errorf("a badge cell left the status roles at %v", styling)
	}
	click(t, tt, "Bước 2/5")
	tt.Frame()
	plain := tester(t, statusGrid(nameCol()))
	click(t, plain, "ann")
	plain.Frame()
	if got, want := cellInk(t, tt, "Bước 2/5"), cellInk(t, plain, "ann"); !nearTol(got, ui.RGB(want.R, want.G, want.B), 12) {
		t.Errorf("the detail on a selected row is %v, the selection's text %v", got, want)
	}
}

// Ours: frames at rest of a grid with badges, formats and a row format
// allocate no more than a plain grid's.
func TestGridStatusFrameAllocs(t *testing.T) {
	calmOff()
	rows := manyAccts(200)
	mk := func(on bool) *ui.Tester {
		col := nameCol()
		tb := DataGridView(func(a *acct) int { return a.id }, col, scoreCol())
		if on {
			col.Badge = func(*acct) Status { return StatusRunning }
			col.BadgeDetail = func(*acct) string { return "Bước 2/5" }
			col.Format = func(*acct) CellFormat { return CellFormat{Status: StatusWarning, Bold: true} }
			tb = DataGridView(func(a *acct) int { return a.id }, col, scoreCol()).
				SetRowFormat(func(*acct) CellFormat { return CellFormat{Fill: ui.Hex("#c0ffee")} })
		}
		tt := ui.NewTester(content(tb.SetRows(rows).Expand()), 800, 600)
		for range 10 {
			tt.Frame()
		}
		return tt
	}
	plain, formatted := mk(false), mk(true)
	p := allocrun.Least(30, plain.Frame)
	f := allocrun.Least(30, formatted.Frame)
	if f > p {
		t.Errorf("a frame at rest of a formatted grid allocates %v, a plain one %v", f, p)
	}
}

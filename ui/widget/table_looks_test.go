package widget

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The panel counts the rows Selected and Checked return: those shown.
func TestTablePanelCountsTheRowsShown(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}, acct{2, "bob", 20}, acct{3, "cat", 30}).CheckboxColumn().BottomPanel()
	tt := tester(t, tb)
	click(t, tt, "bob")
	if err := tt.ClickWith(ui.Cmd, "cat"); err != nil {
		t.Fatal(err)
	}
	tb.SetChecked([]int{1, 2}, true)
	tb.Search("Name", "b")
	tt.Frame()
	if n, sel := tb.CountSelected(), tb.Selected(); n != len(sel) || n != 1 {
		t.Fatalf("CountSelected %d, Selected %v", n, sel)
	}
	for _, s := range []string{"Selected: 1", "Checked: 1"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	tb.Search("Name", "")
	tt.Frame()
	for _, s := range []string{"Selected: 2", "Checked: 2"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func noVerbErrors(t *testing.T, tt *ui.Tester) {
	t.Helper()
	for _, s := range tt.Texts() {
		if strings.Contains(s, "%!") {
			t.Fatalf("a count shows %q", s)
		}
	}
}

func TestTablePanelTextWithoutACount(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}).CheckboxColumn().BottomPanel().
		PanelText(PanelText{Rows: "%d rows", Selected: "%d%% of them", Checked: "Ticked 100%%", Search: "Go"})
	tt := tester(t, tb)
	noVerbErrors(t, tt)
	if !tt.HasText("Ticked 100%") || !tt.HasText("1 rows") || !tt.HasText("0% of them") {
		t.Fatalf("panel: %q", tt.Texts())
	}
}

func TestTablePanelHidesAnEmptyCount(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}).CheckboxColumn().BottomPanel().
		PanelText(PanelText{Rows: "Rows: %d", Selected: "Selected: %d", Search: "Go"})
	tt := tester(t, tb)
	noVerbErrors(t, tt)
	if !tb.checkedLabel.hidden {
		t.Fatalf("the checked count shows: %q", tt.Texts())
	}
}

// boundsOf returns the box of the first element of role, labeled label unless
// that is "".
func boundsOf(t *testing.T, tt *ui.Tester, role ui.Role, label string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == role && (label == "" || n.Label == label) {
			return n.Bounds
		}
	}
	t.Fatalf("no element of role %v", role)
	return ui.Rect{}
}

func TestTablePanelLooks(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}).BottomPanel()
	tt := tester(t, tb)
	before := find(t, tt, "Rows: 1")
	tb.ApplyPanel(Style{}.FontSize(26)).SetSearchWidth(300)
	tt.Frame()
	if after := find(t, tt, "Rows: 1"); after.H <= before.H {
		t.Fatalf("Rows: 1 is %v tall with a font of 26, was %v", after.H, before.H)
	}
	if w := boundsOf(t, tt, ui.RoleTextField, "").W; w != 300 {
		t.Fatalf("the keyword is %v wide", w)
	}
	combo, field := boundsOf(t, tt, ui.RolePopUpButton, ""), boundsOf(t, tt, ui.RoleTextField, "")
	if field.X-(combo.X+combo.W) < 8 {
		t.Fatalf("not joined, yet the combo ends at %v and the keyword starts at %v", combo.X+combo.W, field.X)
	}
	tb.JoinSearch(true)
	tt.Frame()
	combo, field = boundsOf(t, tt, ui.RolePopUpButton, ""), boundsOf(t, tt, ui.RoleTextField, "")
	button := boundsOf(t, tt, ui.RoleButton, "Search")
	if d := field.X - (combo.X + combo.W); d != 0 {
		t.Fatalf("joined, the keyword starts %v after the combo", d)
	}
	if d := button.X - (field.X + field.W); d > 1 {
		t.Fatalf("joined, the button starts %v after the keyword", d)
	}
}

func TestTableRowHeightAndGridLines(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}, acct{2, "bob", 20})
	tt := tester(t, tb)
	plain := slices.Clone(tt.Image().Pix)
	tb.SetRowHeight(40)
	tt.Frame()
	if a, b := find(t, tt, "ann"), find(t, tt, "bob"); b.Y-a.Y != 40 {
		t.Fatalf("rows %v apart with a height of 40", b.Y-a.Y)
	}
	tb.SetRowHeight(0).SetGridLines(true)
	tt.Frame()
	if slices.Equal(plain, tt.Image().Pix) {
		t.Fatal("grid lines draw nothing")
	}
}

// Under a search, the panel's count follows rows that changed in the source
// while the view shows the same places.
func TestTablePanelCountFollowsRowsChangedUnderASearch(t *testing.T) {
	for _, c := range []struct {
		name string
		rows []acct
		act  func(tb *DataGridViewWidget[acct, int])
	}{
		{"SetRows", []acct{{1, "xa", 1}, {2, "yb", 2}}, func(tb *DataGridViewWidget[acct, int]) {
			tb.SetRows([]acct{{2, "xb", 2}, {1, "ya", 1}})
		}},
		{"InsertRows and UpdateRow", []acct{{1, "xa", 1}}, func(tb *DataGridViewWidget[acct, int]) {
			tb.InsertRows(0, acct{2, "xb", 2})
			tb.UpdateRow(1, func(a *acct) { a.name = "ya" })
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tb := testTable(c.rows...).BottomPanel()
			tt := tester(t, tb)
			click(t, tt, "xa")
			tb.Search("Name", "x")
			settleTable(tt)
			if !tt.HasText("Selected: 1") {
				t.Fatalf("before: %q", tt.Texts())
			}
			c.act(tb)
			settleTable(tt)
			if n := tb.CountSelected(); n != 0 || !tt.HasText("Selected: 0") {
				t.Fatalf("CountSelected %d, panel %q", n, tt.Texts())
			}
		})
	}
}

// inkIn counts the dark pixels inside r, inset past its border: the text the
// control draws.
func inkIn(tt *ui.Tester, r ui.Rect) (n int) {
	img := tt.Image()
	scale := float32(img.Rect.Dx()) / 800 // tester's window is 800 wide
	for y := int((r.Y + 4) * scale); y < int((r.Y+r.H-4)*scale); y++ {
		for x := int((r.X + 4) * scale); x < int((r.X+r.W-4)*scale); x++ {
			c := img.RGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) < 3*128 {
				n++
			}
		}
	}
	return n
}

// ApplyPanel's text size reaches the search's controls, not only the counts:
// each draws its text with more ink.
func TestTablePanelTextSizeReachesTheSearch(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}).BottomPanel()
	tt := tester(t, tb)
	tb.Search("Name", "kw")
	tt.Frame()
	type part struct {
		role  ui.Role
		label string
	}
	parts := []part{{ui.RolePopUpButton, ""}, {ui.RoleTextField, ""}, {ui.RoleButton, "Search"}}
	before := make([]int, len(parts))
	for i, p := range parts {
		before[i] = inkIn(tt, boundsOf(t, tt, p.role, p.label))
	}
	tb.ApplyPanel(Style{}.TextSize(XL))
	tt.Frame()
	for i, p := range parts {
		after := inkIn(tt, boundsOf(t, tt, p.role, p.label))
		t.Logf("%v %q: ink %d, was %d", p.role, p.label, after, before[i])
		if before[i] == 0 || after*10 < before[i]*12 {
			t.Errorf("%v %q: ink %d with TextSize(XL), was %d", p.role, p.label, after, before[i])
		}
	}
}

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
		PanelText(PanelText{Rows: "%d rows", Selected: "Selected: %d", Checked: "Ticked", Search: "Go"})
	tt := tester(t, tb)
	noVerbErrors(t, tt)
	if !tt.HasText("Ticked") || !tt.HasText("1 rows") {
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

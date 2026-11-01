package widget

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func accountRows(n int) []acct {
	rows := make([]acct, n)
	for i := range rows {
		rows[i] = acct{i + 1, fmt.Sprintf("name%04d", i+1), i}
	}
	return rows
}

// namesBuilt counts the table's rows built: texts "nameNNNN".
func namesBuilt(tt *Tester) int {
	n := 0
	for _, s := range tt.Tester.Texts() {
		if strings.HasPrefix(s, "name") {
			n++
		}
	}
	return n
}

// A page that passes the room down to a viewport through a Card (which
// isViewport does not look into), or a Custom control drawing a ui.List, is
// wrapped in the FillViewport scroller: the viewport is laid out at an
// unlimited height, builds all its rows, and the table's bottom panel goes
// below the fold. On main both filled the room and virtualized.
func TestSidebarViewportInsideCardOrCustom(t *testing.T) {
	mk := func() *DataGridViewWidget[acct, int] { return testTable(accountRows(2000)...).BottomPanel() }
	t.Run("Card", func(t *testing.T) {
		page := Card().SetTitle(Label("T")).SetBody(mk().Expand()).Expand()
		tt := NewTester(t, Sidebar().Add("a", "One", page), 800, 400)
		tt.Frame()
		r, ok := tt.Tester.Find("Search")
		built := namesBuilt(tt)
		if !ok || r.Y+r.H > 400 || built > 200 {
			t.Fatalf("Search at %v (found %v), %d row texts built in a 400 high window", r, ok, built)
		}
	})
	// Past listMaxRows (4096) rows built at once, the last rows of a table laid
	// out at an unlimited height: can the wheel still reach the last one?
	t.Run("Card, 6000 rows, last row reachable", func(t *testing.T) {
		page := Card().SetTitle(Label("T")).SetBody(testTable(accountRows(6000)...).Expand()).Expand()
		tt := NewTester(t, Sidebar().Add("a", "One", page), 800, 400)
		tt.Frame()
		for range 400 {
			tt.Tester.Scroll(500, 200, 0, 2000)
			tt.Frame()
			if r, ok := tt.Tester.Find("name6000"); ok && r.H > 0 && r.Y >= 0 && r.Y+r.H <= 400 {
				return
			}
		}
		r, ok := tt.Tester.Find("name6000")
		t.Fatalf("the wheel never brought the last row into view: %v %v, %d row texts built", r, ok, namesBuilt(tt))
	})
	t.Run("Custom ui.List", func(t *testing.T) {
		page := func() Control {
			var s ui.ListState
			return Custom(func(c *ui.Context) ui.Element {
				l := ui.List(c, &s, 2000).Grow(1)
				l.Rows(func(r ui.ListRow) { ui.Textf(r.Context, "name%04d", r.Index) })
				return l
			}).Expand()
		}
		for host, mk := range map[string]func() Control{
			"Sidebar":    func() Control { return Sidebar().Add("a", "One", page()) },
			"TabControl": func() Control { return TabControl().AddPage("One", page()).Expand() },
		} {
			tt := NewTester(t, mk(), 800, 400)
			tt.Frame()
			if built := namesBuilt(tt); built > 200 {
				t.Errorf("%s: %d rows of a 2000 row ui.List built in a 400 high window", host, built)
			}
		}
	})
}

// A page whose hidden list shows (a long form that reveals a result list) gets
// unwrapped: the text box the user is typing in must keep its focus and text.
func TestSidebarPageShowingAListKeepsFocus(t *testing.T) {
	list := newListBox().SetVisible(false)
	tb := TextBox()
	page := VBox(4).Add(tb, list)
	tt := NewTester(t, Sidebar().Add("a", "One", page), 500, 300)
	tt.Frame()
	tb.Focus()
	tt.Frame()
	tt.Tester.Type("abc")
	tt.Frame()
	focused := func() string {
		for _, n := range tt.Nodes() {
			if n.Focused {
				return fmt.Sprintf("%v %q %q", n.Role, n.Label, n.Value)
			}
		}
		return "none"
	}
	before := focused()
	list.SetVisible(true)
	tt.Frame()
	tt.Frame()
	after := focused()
	if before != after {
		t.Fatalf("focus was %s, now %s after a list on the page showed", before, after)
	}
	tt.Tester.Type("d")
	tt.Frame()
	if got := tb.Text(); got != "abcd" {
		t.Fatalf("text box has %q after typing abc, showing the list, typing d", got)
	}
}

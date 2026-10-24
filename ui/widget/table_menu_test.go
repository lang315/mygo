package widget

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// menuTable is a table of ann, bob and cat whose menu is built by fn.
func menuTable(fn func([]int) []MenuItem) (*DataGridViewWidget[acct, int], *ui.Tester) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).ContextMenuFunc(fn)
	tt := ui.NewTester(content(tb), 800, 600)
	tt.Frame()
	return tb, tt
}

// posted returns a channel that closes once the event goroutine has run
// everything posted before the call.
func posted() <-chan struct{} {
	done := make(chan struct{})
	Post(func() { close(done) })
	return done
}

func TestContextMenuFuncRunsOffTheLockAndOpensItsItems(t *testing.T) {
	var tb *DataGridViewWidget[acct, int]
	var got struct{ sel, count int }
	tb, tt := menuTable(func(sel []int) []MenuItem {
		got.sel, got.count = len(tb.Selected()), tb.CountSelected() // would deadlock under mu
		return []MenuItem{{Text: fmt.Sprintf("Move %d rows", len(sel))}}
	})
	click(t, tt, "bob")
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	received(t, posted())
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{"Move 1 rows"}) {
		t.Fatalf("menu %q", m)
	}
	if got.sel != 1 || got.count != 1 {
		t.Fatalf("Selected %d, CountSelected %d inside fn", got.sel, got.count)
	}
}

func TestContextMenuFuncGetsTheKeysSelectedAtTheClick(t *testing.T) {
	gate := make(chan struct{})
	Post(func() { <-gate }) // the event goroutine is busy: fn has not started
	var keys []int
	tb, tt := menuTable(func(sel []int) []MenuItem {
		keys = sel
		return []MenuItem{{Text: "Go"}}
	})
	click(t, tt, "bob")
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	tb.chosen.value.Add(3) // the selection changes before fn starts
	mu.Unlock()
	close(gate)
	received(t, posted())
	tt.Frame()
	if !slices.Equal(keys, []int{2}) {
		t.Fatalf("fn got %v, want [2]: the selection at the click", keys)
	}
	if m := tt.Menu(); !slices.Equal(m, []string{"Go"}) {
		t.Fatalf("menu %q", m)
	}
}

func TestContextMenuFuncDropsItsItemsAfterEscapeOrAClickElsewhere(t *testing.T) {
	for name, undo := range map[string]func(tt *ui.Tester){
		"escape": func(tt *ui.Tester) { tt.Key(0, ui.KeyEscape) },
		"click":  func(tt *ui.Tester) { tt.ClickAt(5, 5) },
	} {
		t.Run(name, func(t *testing.T) {
			release := make(chan struct{})
			_, tt := menuTable(func([]int) []MenuItem {
				<-release
				return []MenuItem{{Text: "Late"}}
			})
			if err := tt.RightClick("bob"); err != nil {
				t.Fatal(err)
			}
			undo(tt)
			close(release)
			received(t, posted())
			tt.Frame()
			if m := tt.Menu(); m != nil {
				t.Fatalf("menu %q opened after the user dismissed it", m)
			}
		})
	}
}

func TestContextMenuFuncOnlyTheLatestRightClickOpens(t *testing.T) {
	first, second := make(chan struct{}), make(chan struct{})
	var calls int
	_, tt := menuTable(func(sel []int) []MenuItem {
		calls++
		if calls == 1 {
			<-first
			return []MenuItem{{Text: "First"}}
		}
		<-second
		return []MenuItem{{Text: "Second"}}
	})
	if err := tt.RightClick("ann"); err != nil {
		t.Fatal(err)
	}
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	close(first)
	close(second)
	received(t, posted())
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{"Second"}) {
		t.Fatalf("menu %q, want only the latest right-click's", m)
	}
}

func TestContextMenuFuncEmptyOpensNothing(t *testing.T) {
	_, tt := menuTable(func([]int) []MenuItem { return nil })
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	received(t, posted())
	tt.Frame()
	if m := tt.Menu(); m != nil {
		t.Fatalf("menu %q", m)
	}
}

func TestContextMenuFuncItemOnClickRuns(t *testing.T) {
	ran := make(chan int, 1)
	_, tt := menuTable(func(sel []int) []MenuItem {
		return []MenuItem{{Text: "Run", OnClick: func() { ran <- sel[0] }}}
	})
	click(t, tt, "bob")
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	received(t, posted())
	tt.Frame()
	if err := tt.ChooseMenuItem("Run"); err != nil {
		t.Fatal(err)
	}
	if k := received(t, ran); k != 2 {
		t.Fatalf("OnClick ran for %d", k)
	}
}

func TestContextMenuFuncWinsOverTheStaticMenu(t *testing.T) {
	tb, tt := menuTable(func([]int) []MenuItem { return []MenuItem{{Text: "Built"}} })
	tb.ContextMenu(MenuItem{Text: "Static"})
	tt.Frame()
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	received(t, posted())
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{"Built"}) {
		t.Fatalf("menu %q", m)
	}
}

func TestContextMenuFuncRace(t *testing.T) {
	tb, tt := menuTable(func(sel []int) []MenuItem {
		return []MenuItem{{Text: "x"}}
	})
	for i := range 20 {
		go tb.UpdateRow(1, func(a *acct) { a.score = i })
		if err := tt.RightClick("bob"); err != nil {
			t.Fatal(err)
		}
		tt.CloseMenu()
	}
	received(t, posted())
	tt.Frame()
}

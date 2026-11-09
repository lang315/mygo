package widget

import (
	"slices"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSetSelectedChoosesTheRowsAndIgnoresUnknownKeysWithoutAnEvent(t *testing.T) {
	var events atomic.Int32
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).OnSelectionChanged(func() { events.Add(1) })
	tt := ui.NewTester(content(tb), 800, 600)
	tt.Frame()
	click(t, tt, "ann")
	<-posted() // the click's event has run
	events.Store(0)
	if got := tb.SetSelected(3, 99, 2); got != tb {
		t.Fatal("SetSelected does not return the table")
	}
	if sel := tb.Selected(); !slices.Equal(sel, []int{2, 3}) || tb.CountSelected() != 2 {
		t.Fatalf("Selected %v, CountSelected %d, want [2 3], 2", sel, tb.CountSelected())
	}
	tt.Frame()
	<-posted()
	if n := events.Load(); n != 0 {
		t.Fatalf("OnSelectionChanged fired %d times for a change made by code", n)
	}
	if sel := tb.Selected(); !slices.Equal(sel, []int{2, 3}) {
		t.Fatalf("after a frame Selected %v, want [2 3]", sel)
	}
}

func TestSetSelectedInsideContextMenuFuncShowsAtOnceAndTheItemSeesIt(t *testing.T) {
	inClick := make(chan []int, 1)
	type seen struct {
		sel   []int
		count int
	}
	inFn := make(chan seen, 1)
	var tb *DataGridViewWidget[acct, int]
	tb, tt := menuTable(func([]int) []MenuItem {
		tb.SetSelected(2, 3) // must not deadlock; keeps the right-clicked row
		inFn <- seen{tb.Selected(), tb.CountSelected()}
		return []MenuItem{{Text: "Go", OnClick: func() { inClick <- tb.Selected() }}}
	})
	click(t, tt, "bob")
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	got := soon(t, inFn)
	tt.Frame()
	if !slices.Equal(got.sel, []int{2, 3}) || got.count != 2 {
		t.Fatalf("in fn Selected %v, CountSelected %d, want [2 3], 2", got.sel, got.count)
	}
	if err := tt.ChooseMenuItem("Go"); err != nil {
		t.Fatal(err)
	}
	if got := soon(t, inClick); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("OnClick saw %v, want [2 3]", got)
	}
}

func TestSetSelectedFromAnotherGoroutine(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2})
	done := make(chan struct{})
	go func() { tb.SetSelected(2); close(done) }()
	<-done
	if sel := tb.Selected(); !slices.Equal(sel, []int{2}) {
		t.Fatalf("Selected %v, want [2]", sel)
	}
}

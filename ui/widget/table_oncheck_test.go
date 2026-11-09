package widget

import (
	"fmt"
	"slices"
	"testing"
)

// checkEvents collects what OnCheck reports, as "key on" strings.
func checkEvents(tb *DataGridViewWidget[acct, int]) chan string {
	ev := make(chan string, 64)
	tb.OnCheck(func(k int, on bool) { ev <- fmt.Sprint(k, " ", on) })
	return ev
}

func drainChecks(ev chan string) []string {
	<-posted()
	var got []string
	for {
		select {
		case s := <-ev:
			got = append(got, s)
		default:
			slices.Sort(got)
			return got
		}
	}
}

func TestDataGridViewOnCheckFiresForTheUsersTicks(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).CheckboxColumn()
	ev := checkEvents(tb)
	tt := tester(t, tb)
	click(t, tt, "Check bob")
	if got := drainChecks(ev); !slices.Equal(got, []string{"2 true"}) {
		t.Fatalf("a tick: events %v", got)
	}
	click(t, tt, "Check bob")
	if got := drainChecks(ev); !slices.Equal(got, []string{"2 false"}) {
		t.Fatalf("an untick: events %v", got)
	}
}

func TestDataGridViewOnCheckFiresForEachRowTheHeaderBoxFlips(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).CheckboxColumn()
	ev := checkEvents(tb)
	tt := tester(t, tb)
	click(t, tt, "Check all")
	if got := drainChecks(ev); !slices.Equal(got, []string{"1 true", "2 true", "3 true"}) {
		t.Fatalf("Check all: events %v", got)
	}
	click(t, tt, "Check all")
	if got := drainChecks(ev); !slices.Equal(got, []string{"1 false", "2 false", "3 false"}) {
		t.Fatalf("Check all again: events %v", got)
	}
}

func TestDataGridViewOnCheckIgnoresCode(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).CheckboxColumn()
	ev := checkEvents(tb)
	tt := tester(t, tb)
	tb.SetChecked([]int{1, 2}, true)
	tt.Frame()
	tb.SetChecked([]int{1}, false)
	tt.Frame()
	if got := drainChecks(ev); got != nil {
		t.Fatalf("SetChecked fired %v", got)
	}
}

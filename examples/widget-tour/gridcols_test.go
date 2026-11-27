package main

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// A right-click on the tour's table's header lists its columns, and
// unchecking one hides it.
func TestTourTableColumnChooser(t *testing.T) {
	tt := tourTableTab(t)
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	menu := tt.Menu()
	for _, want := range []string{"ID", "Name", "Status", "Score", "Best fit", "Best fit (all columns)"} {
		if !slices.Contains(menu, want) {
			t.Fatalf("no %q in the header's menu %q", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("Score"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if tt.HasText("Score") {
		t.Fatalf("Score still shows after unchecking it: %q", tt.Texts())
	}
}

// The pointer resting on a Status cell shows the column's Hint.
func TestTourTableStatusHint(t *testing.T) {
	tt := tourTableTab(t)
	var cell ui.Rect // the first row's Status, "ok"
	for _, n := range tt.Nodes() {
		if n.Label == "ok" && n.Bounds.W > 0 {
			cell = n.Bounds
			break
		}
	}
	if cell.W == 0 {
		t.Fatal("no Status cell")
	}
	tt.Move(cell.X+cell.W/2, cell.Y+cell.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("The last check of acct000000 passed") {
		t.Fatalf("no Status hint; texts %q", tt.Texts())
	}
}

// The Best fit button fits the columns to what they show: ID, 90 DIPs wide,
// narrows to its few digits, and Name moves left.
func TestTourTableBestFit(t *testing.T) {
	tt := tourTableTab(t)
	before, ok := tt.Find("Name")
	if !ok {
		t.Fatal("no Name header")
	}
	if err := tt.Click("Best fit"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Frame()
	after, _ := tt.Find("Name")
	if after.X >= before.X-10 {
		t.Fatalf("the Name header at %v, before Best fit at %v: ID not fitted", after.X, before.X)
	}
}

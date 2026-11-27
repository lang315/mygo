package main

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The tour's table groups by status at the toggle's click, and the group's
// header line shows the Score total, under the Score header.
func TestTourTableGroupTotals(t *testing.T) {
	tt := tourTableTab(t)
	if err := tt.Click("Group by status"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	settle(tt)
	if !tt.HasText("ok (100000)") {
		t.Fatalf("no group header after Group by status; texts %q", tt.Texts())
	}
	want := 0
	for i := range 100_000 {
		want += i % 997
	}
	total := tourNode(t, tt, ui.RoleText, strconv.Itoa(want)).Bounds
	score := tourNode(t, tt, ui.RoleColumnHeader, "Score").Bounds
	if total.X < score.X-1 || total.X+total.W > score.X+score.W+1 {
		t.Fatalf("the total at %v is not under Score's header at %v", total, score)
	}
}

// The group toggle and the row dragging toggles live side by side: with
// Reorder rows and then Drag rows out on, the grouped table keeps its
// header and its total, and grouping off brings the rows back.
func TestTourTableGroupTotalsWithDragging(t *testing.T) {
	tt := tourTableTab(t)
	for _, toggle := range []string{"Group by status", "Reorder rows", "Drag rows out"} {
		if err := tt.Click(toggle); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		settle(tt)
	}
	want := 0
	for i := range 100_000 {
		want += i % 997
	}
	if !tt.HasText("ok (100000)") || !tt.HasText(strconv.Itoa(want)) {
		t.Fatalf("with the drag toggles on, the group or its total is gone; texts %q", tt.Texts())
	}
	if err := tt.Click("Group by status"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	settle(tt)
	if tt.HasText("ok (100000)") || !tt.HasText("acct000000") {
		t.Fatalf("grouping off with the drag toggles on; texts %q", tt.Texts())
	}
}

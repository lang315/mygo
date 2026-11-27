package main

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// tourNode returns the tour's element of role r labeled l.
func tourNode(t *testing.T, tt *ui.Tester, r ui.Role, l string) ui.TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r && n.Label == l {
			return n
		}
	}
	t.Fatalf("no element of role %v labeled %q", r, l)
	return ui.TestNode{}
}

// The tour's table filters from its header: typing in the Name column's
// field of the auto filter row keeps the matching rows, the filter panel
// says so, and its Clear brings every row back; the Status column's funnel
// opens a check list of its values.
func TestTourTableFilters(t *testing.T) {
	tt := tourTableTab(t)
	f := tourNode(t, tt, ui.RoleTextField, "Filter Name").Bounds
	tt.ClickAt(f.X+f.W/2, f.Y+f.H/2)
	tt.Frame()
	tt.Type("acct00001")
	settle(tt)
	texts := tt.Texts()
	if slices.Contains(texts, "acct000000") || !slices.Contains(texts, "acct000010") {
		t.Fatalf("typed acct00001 in the Name field; texts %q", texts)
	}
	if !tt.HasText(`Name contains "acct00001"`) {
		t.Fatalf("the filter panel does not show the filter; texts %q", texts)
	}
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !slices.Contains(tt.Texts(), "acct000000") || tt.HasText(`Name contains`) {
		t.Fatalf("after the panel's Clear; texts %q", tt.Texts())
	}
	b := tourNode(t, tt, ui.RoleButton, "Filter Status").Bounds
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	settle(tt)
	tourNode(t, tt, ui.RoleCheckBox, "ok")
	tourNode(t, tt, ui.RoleTextField, "Contains")
}

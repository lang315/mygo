package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// navigationPage opens the tour's Navigation page.
func navigationPage(t *testing.T) *widget.Tester {
	t.Helper()
	tt := widget.NewTester(t, tour(nil), 1400, 1400)
	tt.Find(widget.ByLabel("Navigation").Role(ui.RoleTab)).Click()
	return tt
}

func TestTourSearchBoxFilters(t *testing.T) {
	tt := navigationPage(t)
	tt.Find(widget.ByRole(ui.RoleTextField).Nth(0)).Type("hu")
	if !tt.Has(widget.ByLabel("Hue")) || tt.Has(widget.ByLabel("Hanoi, Hue, Da Nang, Ho Chi Minh City, Can Tho")) {
		t.Fatalf("typing hu did not filter the cities: %q", tt.Texts())
	}
}

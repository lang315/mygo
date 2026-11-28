package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Choices tab's LookUpEdit of 10,000 products searches them in its
// drop-down, which builds only the rows in view, and Enter chooses one.
func TestTourSearchLookUp(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	visit(tt, "Choices")
	tt.Find(widget.ByLabel("Product")).Click()
	tt.Settle()
	if n := len(tt.FindAll(widget.ByRole(ui.RoleListItem))); n == 0 || n > 16 {
		t.Fatalf("the drop-down of 10,000 products built %d rows", n)
	}
	tt.Find(widget.ByLabel("Search")).Type("product 9999")
	tt.Settle()
	rows := tt.FindAll(widget.ByRole(ui.RoleListItem))
	if len(rows) != 1 || rows[0].Label() != "P09999 Product 9999 Garden" {
		t.Fatalf("the rows for product 9999: %v", rows)
	}
	tt.Tester.Key(0, ui.KeyDown)
	tt.Settle()
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	if !tt.HasText("Product 9999 chosen") {
		t.Fatalf("Enter did not choose Product 9999: %q", tt.Texts())
	}
}

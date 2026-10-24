package ui

import (
	"fmt"
	"testing"
)

func TestTesterNodesDescribeTheLastFrame(t *testing.T) {
	d := &demo{name: "Ada", volume: 30}
	tt := coreNewTester(d.view, 640, 600)
	find := func(role Role, label string) TestNode {
		t.Helper()
		for _, n := range tt.Nodes() {
			if n.Role == role && n.Label == label {
				return n
			}
		}
		t.Fatalf("no %d labeled %q in %+v", role, label, tt.Nodes())
		return TestNode{}
	}
	inc := find(RoleButton, "Increment")
	if inc.Bounds.W <= 0 || inc.Disabled {
		t.Errorf("Increment: %+v", inc)
	}
	if find(RoleCheckBox, "I agree").Checked {
		t.Error("the check box is checked")
	}
	if s := find(RoleSlider, ""); s.Min != 0 || s.Max != 100 || s.Now != 30 {
		t.Errorf("slider %+v", s)
	}
	if f := find(RoleTextField, ""); f.Value != "Ada" {
		t.Errorf("field %+v", f)
	}
	if err := tt.Click("I agree"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !find(RoleCheckBox, "I agree").Checked {
		t.Error("the nodes still show the check box unchecked after a frame")
	}
}

func TestTesterNodesAreACopy(t *testing.T) {
	tt := coreNewTester((&demo{}).view, 640, 600)
	a := tt.Nodes()
	if len(a) == 0 {
		t.Fatal("no nodes")
	}
	a[0].Label = "changed"
	if tt.Nodes()[0].Label == "changed" {
		t.Fatal("a test changed the access tree through Nodes")
	}
}

// scrollList is 20 buttons in a scroll area of 100 DIPs.
func scrollList(c *Context) {
	Scroll(c).Height(100).Children(func() {
		for i := 0; i < 20; i++ {
			Button(c, fmt.Sprint("row ", i))
		}
	})
}

func TestTesterNodeBoundsAreClippedByTheScrollAreaAround(t *testing.T) {
	tt := NewTester(scrollList, 400, 600)
	var row TestNode
	for _, n := range tt.Nodes() {
		if n.Label == "row 5" {
			row = n
		}
	}
	if row.ID == 0 || row.Bounds.W != 0 && row.Bounds.H != 0 {
		t.Fatalf("row 5 is scrolled out of view, and its visible box is %+v", row.Bounds)
	}
	if !row.Scrolls {
		t.Error("row 5 is in a scroll area, and Scrolls is false")
	}
	tt.ScrollIntoView(row.ID)
	tt.Frame()
	for _, n := range tt.Nodes() {
		if n.ID == row.ID && (n.Bounds.H <= 0 || n.Bounds.Y+n.Bounds.H > 100) {
			t.Fatalf("after ScrollIntoView, row 5 is at %+v", n.Bounds)
		}
	}
}

func TestTesterFocusNodeFocusesWithoutPressing(t *testing.T) {
	clicks := 0
	view := func(c *Context) {
		if Button(c, "OK").Clicked() {
			clicks++
		}
	}
	tt := NewTester(view, 400, 300)
	var ok TestNode
	for _, n := range tt.Nodes() {
		if n.Label == "OK" {
			ok = n
		}
	}
	tt.FocusNode(ok.ID)
	tt.Frame()
	if clicks != 0 {
		t.Fatalf("focusing pressed the button %d times", clicks)
	}
	for _, n := range tt.Nodes() {
		if n.Label == "OK" && !n.Focused {
			t.Fatal("the button is not focused")
		}
	}
}

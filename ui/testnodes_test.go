package ui

import "testing"

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

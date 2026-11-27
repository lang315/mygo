package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// probeNode returns the first node of the last frame with role r (and label
// l when not empty).
func probeNode(t *testing.T, tt *ui.Tester, r ui.Role, l string) ui.TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r && (l == "" || n.Label == l) {
			return n
		}
	}
	t.Fatalf("no node of role %v labeled %q", r, l)
	return ui.TestNode{}
}

// Step 0 of the filter task: a HeaderCell callback adds a button and a text
// field to a header cell; a click on the button reaches the button and sorts
// nothing, and a click in the field then typing edits the field.
func TestFilterProbeHeaderCellChildTakesPointer(t *testing.T) {
	var s ui.ListState
	order := ui.SortOrder{}
	s.Sort = &order
	clicks, text := 0, ""
	cols := []ui.TableColumn{{Title: "Name", Width: 200, Sortable: true}, {Title: "Kind", Width: 150, Sortable: true}}
	var cc *ui.Context
	s.HeaderCell = func(cell ui.Element, col int) {
		if col != 0 {
			return
		}
		if ui.Button(cc, "F").Clicked() {
			clicks++
		}
		ui.TextInput(cc, &text).Width(80)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		cc = c
		ui.Column(c).Fill().Children(func() {
			ui.Table(c, &s, cols, 3, func(row, col int) { ui.Text(c, "x") }).Grow(1)
		})
	}, 600, 400)
	tt.Frame()
	if err := tt.Click("F"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if clicks != 1 || order.Column != "" {
		t.Fatalf("a click on the header's button: %d clicks, sort %+v", clicks, order)
	}
	f := probeNode(t, tt, ui.RoleTextField, "")
	tt.ClickAt(f.Bounds.X+f.Bounds.W/2, f.Bounds.Y+f.Bounds.H/2)
	tt.Frame()
	tt.Type("an")
	tt.Frame()
	if text != "an" || order.Column != "" {
		t.Fatalf("typed into the header's field: %q, sort %+v", text, order)
	}
}

// Step 0, the drop-down: a Popover anchored on a header's button holds a
// check box and a field, which take clicks and typing, sort nothing, and
// leave it open; in a Sortable header's own field, keys, Enter and a drag
// neither sort nor move the column.
func TestFilterProbeHeaderPopoverAndKeys(t *testing.T) {
	var s ui.ListState
	order := ui.SortOrder{}
	s.Sort = &order
	open, checked, inPop, inHead := false, true, "", ""
	changes := 0
	cols := []ui.TableColumn{{Title: "Name", Width: 200, Sortable: true}, {Title: "Kind", Width: 150, Sortable: true}}
	var cc *ui.Context
	s.HeaderCell = func(cell ui.Element, col int) {
		if col != 0 {
			return
		}
		b := ui.Button(cc, "F")
		if b.Clicked() {
			open = !open
		}
		ui.Popover(cc, b, &open, func() {
			if ui.Checkbox(cc, &checked, "Doc").Changed() {
				changes++
			}
			ui.TextInput(cc, &inPop).Width(120)
		})
		ui.TextInput(cc, &inHead).Width(80)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		cc = c
		ui.Column(c).Fill().Children(func() {
			ui.Table(c, &s, cols, 3, func(row, col int) { ui.Text(c, "x") }).Grow(1)
		})
	}, 600, 400)
	tt.Frame()
	if err := tt.Click("F"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !open {
		t.Fatal("the popover did not open")
	}
	if err := tt.Click("Doc"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if checked || changes != 1 || !open || order.Column != "" {
		t.Fatalf("after a click on the popover's box: checked %v, %d changes, open %v, sort %+v", checked, changes, open, order)
	}
	var pf ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTextField && n.Bounds.Y > 40 {
			pf = n
		}
	}
	if pf.ID == 0 {
		t.Fatal("no field in the popover")
	}
	tt.ClickAt(pf.Bounds.X+pf.Bounds.W/2, pf.Bounds.Y+pf.Bounds.H/2)
	tt.Frame()
	tt.Type("do")
	tt.Frame()
	if inPop != "do" || !open || order.Column != "" {
		t.Fatalf("typed in the popover: %q, open %v, sort %+v", inPop, open, order)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if open {
		t.Fatal("Escape left the popover open")
	}
	// The header's own field.
	var hf ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTextField && n.Bounds.Y < 40 {
			hf = n
		}
	}
	x, y := hf.Bounds.X+hf.Bounds.W/2, hf.Bounds.Y+hf.Bounds.H/2
	tt.ClickAt(x, y)
	tt.Frame()
	tt.Type("a b")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Press(x, y)
	tt.Frame()
	for d := float32(5); d <= 30; d += 5 {
		tt.Move(x+d, y)
		tt.Frame()
	}
	tt.Release(x+30, y)
	tt.Frame()
	if inHead != "a b" || order.Column != "" || len(s.Columns.Order) != 0 {
		t.Fatalf("in the header's field: %q, sort %+v, order %v", inHead, order, s.Columns.Order)
	}
	name := probeNode(t, tt, ui.RoleColumnHeader, "Name")
	if name.Bounds.X != 0 {
		t.Fatalf("the Name column moved to %v", name.Bounds.X)
	}
}

// Step 0, the auto filter row: a field on a second line under the title, the
// header made taller by a theme's HeaderHeight, as wide as its column after
// a resize, scrolling sideways with it and pinned with a Frozen one.
func TestFilterProbeHeaderSecondLine(t *testing.T) {
	var s ui.ListState
	texts := [3]string{}
	cols := []ui.TableColumn{{Title: "A", Width: 120, Frozen: true}, {Title: "B", Width: 400}, {Title: "C", Width: 400}}
	var cc *ui.Context
	const fieldH = 28
	s.HeaderCell = func(cell ui.Element, col int) {
		cell.Padding(0, 10, fieldH, 10)
		ui.TextInput(cc, &texts[col]).Absolute().Bottom(2).Start(2).End(2).Height(fieldH - 4)
	}
	base := ui.LightTheme()
	th := *base
	th.Metrics.HeaderHeight = base.Space(8) + fieldH
	tt := ui.NewTester(func(c *ui.Context) {
		cc = c
		ui.Column(c).Fill().Children(func() {
			ui.Themed(c, &th).Background(ui.Transparent).Grow(1).Children(func() {
				ui.Table(c, &s, cols, 3, func(row, col int) { ui.Text(c, "x") }).Grow(1)
			})
		})
	}, 500, 300)
	tt.Frame()
	fields := func() (out []ui.TestNode) {
		for _, n := range tt.Nodes() {
			if n.Role == ui.RoleTextField {
				out = append(out, n)
			}
		}
		return out
	}
	heads := func() (out []ui.TestNode) {
		for _, n := range tt.Nodes() {
			if n.Role == ui.RoleColumnHeader {
				out = append(out, n)
			}
		}
		return out
	}
	fs, hs := fields(), heads()
	if len(fs) != 3 || len(hs) != 3 {
		t.Fatalf("%d fields, %d headers", len(fs), len(hs))
	}
	for i := range fs {
		h, f := hs[i].Bounds, fs[i].Bounds
		if i == 2 {
			continue // out of view until the scroll
		}
		if f.Y < h.Y+h.H/2-1 || f.Y+f.H > h.Y+h.H+0.5 {
			t.Errorf("field %d at %+v is not on the second line of its header %+v", i, f, h)
		}
		if f.W < h.W-6 || f.W > h.W {
			t.Errorf("field %d is %v wide, its header %v", i, f.W, h.W)
		}
	}
	// Type into B's field.
	tt.ClickAt(fs[1].Bounds.X+fs[1].Bounds.W/2, fs[1].Bounds.Y+fs[1].Bounds.H/2)
	tt.Frame()
	tt.Type("zz")
	tt.Frame()
	if texts[1] != "zz" {
		t.Fatalf("typed into B's field: %q", texts)
	}
	// Resize B by layout.
	s.Columns.Widths = map[string]float32{"B": 300}
	tt.Frame()
	fs, hs = fields(), heads()
	if f, h := fs[1].Bounds, hs[1].Bounds; f.W < h.W-6 || f.W > h.W || h.W != 300 {
		t.Errorf("after a resize to 300, B's field %+v, header %+v", f, h)
	}
	// Scroll sideways.
	before := fs
	tt.Scroll(250, 150, 200, 0)
	tt.Frame()
	fs, hs = fields(), heads()
	if fs[0].Bounds.X != before[0].Bounds.X {
		t.Errorf("the Frozen column's field moved from %v to %v", before[0].Bounds.X, fs[0].Bounds.X)
	}
	if hs[2].Bounds.X == 0 || fs[2].Bounds.X < hs[2].Bounds.X || fs[2].Bounds.X > hs[2].Bounds.X+4 {
		t.Errorf("C's field at %+v, not over its header %+v", fs[2].Bounds, hs[2].Bounds)
	}
}

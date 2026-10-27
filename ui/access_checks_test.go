package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// accessLabels returns the labels of the nodes of role in the order of the
// tree.
func accessLabels(tree *platform.AccessTree, role platform.AccessRole) []string {
	var labels []string
	for _, n := range tree.Nodes {
		if n.Role == role {
			labels = append(labels, n.Label)
		}
	}
	return labels
}

// A row's own check box names itself, not the row: a row without a Label
// takes its name from what is inside it (innerText), which counts the Label
// of an element inside it, and read "Check Cell 0 0 Cell 0 1".
func TestTableRowNameLeavesOutItsCheckBox(t *testing.T) {
	var checked Selection[int]
	s := ListState{Checked: &checked}
	cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, cols, 3, func(row, col int) { coreTextf(c, "Cell %d %d", row, col) }).Grow(1)
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	rows := accessLabels(tt.h.access, platform.RoleRow)
	if want := []string{"Name Size", "Cell 0 0 Cell 0 1", "Cell 1 0 Cell 1 1", "Cell 2 0 Cell 2 1"}; !slices.Equal(rows, want) {
		t.Errorf("rows are named %q, want %q", rows, want)
	}
	boxes := accessLabels(tt.h.access, platform.RoleCheckBox)
	if want := []string{"Check all", "Check", "Check", "Check"}; !slices.Equal(boxes, want) {
		t.Errorf("check boxes are named %q, want %q", boxes, want)
	}
}

// A Label of the list still names the row, and its check box says which.
func TestTableRowLabelNamesItWithChecks(t *testing.T) {
	var checked Selection[int]
	s := ListState{Checked: &checked, Label: func(i int) string { return "Person " + string(rune('0'+i)) }}
	cols := []TableColumn{{Title: "Name"}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, cols, 2, func(row, col int) { coreTextf(c, "Cell %d", row) }).Grow(1)
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	rows := accessLabels(tt.h.access, platform.RoleRow)
	if want := []string{"Name", "Person 0", "Person 1"}; !slices.Equal(rows, want) {
		t.Errorf("rows are named %q, want %q", rows, want)
	}
	boxes := accessLabels(tt.h.access, platform.RoleCheckBox)
	if want := []string{"Check all", "Check Person 0", "Check Person 1"}; !slices.Equal(boxes, want) {
		t.Errorf("check boxes are named %q, want %q", boxes, want)
	}
}

func TestGridItemNameLeavesOutItsCheckBox(t *testing.T) {
	var checked Selection[int]
	g := GridState{Checked: &checked}
	tt := coreNewTester(func(c *context) {
		coreGridView(c, &g, 3, 100, 80, func(i int) { coreTextf(c, "Photo %d", i) }).Grow(1)
	}, 448, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	items := accessLabels(tt.h.access, platform.RoleListItem)
	if want := []string{"Photo 0", "Photo 1", "Photo 2"}; !slices.Equal(items, want) {
		t.Errorf("items are named %q, want %q", items, want)
	}
	boxes := accessLabels(tt.h.access, platform.RoleCheckBox)
	if want := []string{"Check", "Check", "Check"}; !slices.Equal(boxes, want) {
		t.Errorf("check boxes are named %q, want %q", boxes, want)
	}
}

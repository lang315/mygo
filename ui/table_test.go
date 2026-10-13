package ui

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// tableView shows a table of files: name, kind and size.
func tableView(s *ListState, cols []TableColumn, names []string) func(c *context) {
	return func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, s, cols, len(names), func(row, col int) {
				switch col {
				case 0:
					coreText(c, names[row]).SingleLine()
				case 1:
					coreText(c, "Document")
				case 2:
					coreTextf(c, "%d KB", row+1)
				}
			}).Grow(1)
		})
	}
}

func TestTableSort(t *testing.T) {
	sort := SortOrder{Column: "Size"}
	sel := -1
	s := ListState{Sort: &sort, Selected: &sel}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End, Sortable: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "b.txt"}), 500, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleColumnHeader, "Size"); n.States&platform.AccessSortAscending == 0 {
		t.Errorf("the column sorted by: %+v", n)
	}
	tt.Click("Name")
	if sort != (SortOrder{Column: "Name"}) {
		t.Fatalf("a click on Name: %+v", sort)
	}
	tt.Click("Name")
	if sort != (SortOrder{Column: "Name", Descending: true}) {
		t.Fatalf("a second click on Name: %+v", sort)
	}
	tree := tt.h.access
	if n := accessNode(t, tree, platform.RoleColumnHeader, "Name"); n.States&platform.AccessSortDescending == 0 || n.Actions&platform.ActionPress == 0 {
		t.Errorf("Name, sorted descending: %+v", n)
	}
	if n := accessNode(t, tree, platform.RoleColumnHeader, "Size"); n.States&(platform.AccessSortAscending|platform.AccessSortDescending) != 0 {
		t.Errorf("Size, no longer sorted by: %+v", n)
	}
	// A column that does not sort.
	tt.Click("Kind")
	if sort != (SortOrder{Column: "Name", Descending: true}) || sel != -1 {
		t.Errorf("a click on Kind: %+v, chose %d", sort, sel)
	}
}

// header returns the box of a column's header.
func header(t *testing.T, tt *Tester, title string) platform.RectF {
	t.Helper()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	return accessNode(t, tt.h.access, platform.RoleColumnHeader, title).Bounds
}

func TestTableResize(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100, MinWidth: 60}, {Title: "Size", Width: 80, Fixed: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "a much longer name.txt"}), 500, 300)
	kind := header(t, tt, "Kind")
	// Dragging the right edge of Kind's header.
	edge := float32(kind.X+kind.W) - 2
	tt.Press(edge, float32(kind.Y+kind.H/2))
	tt.Move(edge+30, float32(kind.Y+kind.H/2))
	tt.Release(edge+30, float32(kind.Y+kind.H/2))
	if w, h := s.Columns.Widths["Kind"], header(t, tt, "Kind"); w != 130 || h.W != 130 || float32(h.X+h.W)-2 != edge+30 {
		t.Fatalf("dragged 30 DIPs wider: %v, the header %v", w, h)
	}
	// The column before it, sharing the room left, kept its width, for the
	// edge to follow the pointer.
	if w := s.Columns.Widths["Name"]; w != 300 {
		t.Errorf("Name took %v", w)
	}
	edge += 30
	// No narrower than its least.
	tt.Press(edge, float32(kind.Y+kind.H/2))
	tt.Move(edge-130, float32(kind.Y+kind.H/2))
	tt.Release(edge-130, float32(kind.Y+kind.H/2))
	if w := header(t, tt, "Kind").W; w != 60 {
		t.Errorf("dragged narrower than its least: %v", w)
	}
	// A double click fits it to its cells, and the cells follow.
	kind = header(t, tt, "Kind")
	edge = float32(kind.X+kind.W) - 2
	tt.ClickAt(edge, float32(kind.Y+kind.H/2))
	tt.ClickAt(edge, float32(kind.Y+kind.H/2))
	doc, _ := tt.Find("Document")
	if w := float32(header(t, tt, "Kind").W); w < doc.W+20 || w > doc.W+21 || w < 70 {
		t.Errorf("fitted to %v for a text of %v", w, doc.W)
	}
	// A fixed column has no edge to drag.
	size := header(t, tt, "Size")
	tt.Press(float32(size.X+size.W-1), float32(size.Y+size.H/2))
	tt.Move(float32(size.X+size.W+40), float32(size.Y+size.H/2))
	tt.Release(float32(size.X+size.W+40), float32(size.Y+size.H/2))
	if w := header(t, tt, "Size").W; w != 80 {
		t.Errorf("a fixed column resized to %v", w)
	}
}

func TestTableReorder(t *testing.T) {
	sort := SortOrder{}
	s := ListState{Sort: &sort}
	cols := []TableColumn{{Title: "Name", Fixed: true}, {Title: "Kind", Width: 100, Sortable: true}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	kind := header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)
	// Kind dragged past the middle of Size.
	x := float32(kind.X + 20)
	tt.Press(x, y)
	for _, d := range []float32{5, 20, 50, 60} {
		tt.Move(x+d, y)
	}
	tt.Release(x+60, y)
	if got := s.Columns.Order; len(got) != 3 || got[1] != "Size" || got[2] != "Kind" {
		t.Fatalf("the order: %q", got)
	}
	if sort.Column != "" {
		t.Errorf("dragging the header sorted by %q", sort.Column)
	}
	kind, size := header(t, tt, "Kind"), header(t, tt, "Size")
	doc, _ := tt.Find("Document")
	if !(size.X < kind.X) || float64(doc.X) < kind.X {
		t.Errorf("Size at %v, Kind at %v, its cell at %v", size.X, kind.X, doc.X)
	}
	// Name is fixed: it neither moves nor lets others past it.
	tt.Press(float32(size.X+10), y)
	for _, d := range []float32{-5, -40, -100, -200} {
		tt.Move(float32(size.X+10)+d, y)
	}
	tt.Release(float32(size.X+10)-200, y)
	if got := s.Columns.Order; got[0] != "Name" {
		t.Errorf("moved past a fixed column: %q", got)
	}
}

// A column dragged past several others stays the one dragged: its header
// keeps the press as the others move aside.
func TestTableReorderFar(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 80}, {Title: "Size", Width: 80}, {Title: "Date", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 600, 300)
	kind := header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)
	x := float32(kind.X + 20)
	tt.Press(x, y)
	for d := float32(5); d <= 180; d += 15 {
		tt.Move(x+d, y)
	}
	tt.Release(x+180, y)
	if got := s.Columns.Order; len(got) != 4 || got[1] != "Size" || got[2] != "Date" || got[3] != "Kind" {
		t.Fatalf("the order after dragging Kind past Size and Date: %q", got)
	}
}

func TestTableScrollsSideways(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name", Width: 200}, {Title: "Kind", Width: 200}, {Title: "Size", Width: 200}}
	names := make([]string, 50)
	for i := range names {
		names[i] = "file"
	}
	tt := coreNewTester(tableView(&s, cols, names), 400, 300)
	if tt.HasText("Size") {
		size := header(t, tt, "Size")
		if size.X < 390 {
			t.Fatalf("the third column of 200 shows at %v in a table 380 wide", size.X)
		}
	}
	tt.Scroll(200, 150, 300, 0)
	size := header(t, tt, "Size")
	cell, _ := tt.Find("1 KB")
	if size.X > 300 || float64(cell.X) != size.X+10 {
		t.Errorf("scrolled sideways: the header at %v, the text of its cell at %v", size.X, cell.X)
	}
	// Down still moves the rows, and the header stays.
	before := size.Y
	tt.Scroll(200, 150, 0, 300)
	if size := header(t, tt, "Size"); size.Y != before || s.first == 0 {
		t.Errorf("scrolled down: the header at %v (was %v), first row %d", size.Y, before, s.first)
	}
}

func TestEditableTextInTable(t *testing.T) {
	names := []string{"notes.txt", "photo.jpeg", "README"}
	sel, opened, renames := -1, -1, 0
	s := ListState{Selected: &sel}
	cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80}}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			if coreTable(c, &s, cols, len(names), func(row, col int) {
				if col == 0 {
					if coreEditableText(c, &names[row]).Changed() {
						renames++
					}
				} else {
					coreText(c, "1 KB")
				}
			}).Grow(1).Submitted() {
				opened = sel
			}
		})
	}, 400, 300)
	rename := func() {
		if runtime.GOOS == "darwin" {
			tt.Key(0, KeyEnter)
		} else {
			tt.Key(0, KeyF2)
		}
	}
	tt.Click("photo.jpeg")
	rename()
	if opened != -1 {
		t.Fatalf("the key editing the name opened the row")
	}
	// The name before its extension is selected: typing replaces it.
	tt.Type("beach")
	tt.Key(0, KeyEnter)
	if names[1] != "beach.jpeg" || renames != 1 {
		t.Fatalf("renamed to %q (%d)", names[1], renames)
	}
	if !tt.Focused("beach.jpeg") {
		t.Error("the table did not take the focus back")
	}
	// Escape goes back.
	tt.Key(0, KeyF2)
	tt.Type("gone")
	tt.Key(0, KeyEscape)
	if names[1] != "beach.jpeg" || !tt.HasText("beach.jpeg") {
		t.Errorf("Escape left %q", names[1])
	}
	// A click on the name of the row chosen edits it, once a double click
	// is out of the question; the focus leaving keeps what was typed.
	tt.Click("beach.jpeg")
	if tt.HasText("beach.jpeg") != true {
		t.Fatal("the click edited at once")
	}
	time.Sleep(renameDelay + 50*time.Millisecond)
	tt.Frame()
	tt.Type("sea")
	tt.Click("notes.txt")
	if names[1] != "sea.jpeg" || sel != 0 {
		t.Errorf("after a click elsewhere: %q, chose %d", names[1], sel)
	}
	// A double click opens, and does not edit.
	tt.Click("notes.txt")
	tt.Click("notes.txt")
	time.Sleep(renameDelay + 50*time.Millisecond)
	tt.Frame()
	tt.Type("x")
	if opened != 0 || names[0] != "notes.txt" {
		t.Errorf("a double click: opened %d, the name %q", opened, names[0])
	}
}

func TestEditableText(t *testing.T) {
	title := "Untitled.md"
	changes := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			if coreEditableText(c, &title).Changed() {
				changes++
			}
			coreButton(c, "Other")
		})
	}, 400, 300)
	r, _ := tt.Find("Untitled.md")
	tt.ClickAt(r.X+5, r.Y+5)
	tt.ClickAt(r.X+5, r.Y+5)
	tt.Type("Notes")
	tt.Key(0, KeyTab)
	if title != "Notes.md" || changes != 1 {
		t.Fatalf("after Tab: %q (%d)", title, changes)
	}
	tt.Click("Notes.md")
	tt.Key(0, KeyEnter)
	tt.Key(Cmd, KeyA)
	tt.Type("Plan")
	tt.Key(0, KeyEnter)
	if title != "Plan" || !tt.Focused("Plan") {
		t.Errorf("after Enter: %q, focused %v", title, tt.Focused("Plan"))
	}
}

func TestTableRoundRows(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End}}
	// radius returns the corners of row i's box, or of the box inside it
	// that shows the pointer over it in a table that does not choose.
	radius := func(s *ListState, i int, inner bool) [4]float32 {
		t.Helper()
		for _, r := range s.frame.rows {
			if r.i == i {
				if inner {
					return r.e.first.radius
				}
				return r.e.radius
			}
		}
		t.Fatalf("row %d not built", i)
		return [4]float32{}
	}
	r := LightTheme().Radius
	square, round := [4]float32{}, [4]float32{r, r, r, r}

	sel := 1
	s := ListState{Selected: &sel}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 1, false); got != square {
		t.Errorf("a chosen row by default: %v", got)
	}
	s = ListState{Selected: &sel, RoundRows: true}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 1, false); got != round {
		t.Errorf("a chosen row with RoundRows: %v", got)
	}

	// Rows chosen together make one block.
	var chosen Selection[string]
	chosen.Add("a.txt")
	chosen.Add("b.txt")
	s = ListState{Selection: &chosen, Key: func(i int) any { return names[i] }, RoundRows: true}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 0, false); got != [4]float32{r, r, 0, 0} {
		t.Errorf("the first of two rows chosen: %v", got)
	}
	if got := radius(&s, 1, false); got != [4]float32{0, 0, r, r} {
		t.Errorf("the second of two rows chosen: %v", got)
	}

	// A table that does not choose shows the pointer over its rows.
	s = ListState{}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 0, true); got != square {
		t.Errorf("a row of a table that does not choose, by default: %v", got)
	}
	s = ListState{RoundRows: true}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 0, true); got != round {
		t.Errorf("a row of a table that does not choose, with RoundRows: %v", got)
	}
}

func TestTableEvents(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End}}
	sort := SortOrder{Column: "Size"}
	sel := -1
	s := ListState{Sort: &sort, Selected: &sel}
	var seen, calls []string
	view := func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			e := coreTable(c, &s, cols, len(names), func(row, col int) {
				if col == 0 {
					coreText(c, names[row]).SingleLine()
				}
			}).Grow(1)
			for _, q := range []struct {
				name string
				on   bool
			}{{"changed", e.Changed()}, {"selection", e.SelectionChanged()}, {"sort", e.SortChanged()}, {"columns", e.ColumnsChanged()}, {"checked", e.CheckedChanged()}} {
				if q.on {
					seen = append(seen, q.name)
				}
			}
			if row, ok := e.Activated(); ok {
				seen = append(seen, fmt.Sprintf("activated %d", row))
			}
			wrapElement(e).
				OnSelectionChange(func() { calls = append(calls, "selection") }).
				OnSortChange(func() { calls = append(calls, "sort") }).
				OnColumnsChange(func() { calls = append(calls, "columns") }).
				OnActivate(func(row int) { calls = append(calls, fmt.Sprintf("activate %d", row)) })
		})
	}
	tt := coreNewTester(view, 500, 300)
	check := func(what string, wantSeen, wantCalls []string) {
		t.Helper()
		if !slices.Equal(seen, wantSeen) || !slices.Equal(calls, wantCalls) {
			t.Errorf("%s: saw %q, called %q; want %q, %q", what, seen, calls, wantSeen, wantCalls)
		}
		seen, calls = nil, nil
	}
	check("the first frame", nil, nil)
	tt.Click("b.txt")
	check("a click on a row", []string{"changed", "selection"}, []string{"selection"})
	tt.Click("Name")
	check("a click on a header", []string{"changed", "sort"}, []string{"sort"})
	tt.Click("c.txt")
	seen, calls = nil, nil
	tt.Click("c.txt")
	check("a double click", []string{"activated 2"}, []string{"activate 2"})
	tt.Key(0, KeyEnter)
	check("Enter", []string{"activated 2"}, []string{"activate 2"})
	// Resizing a column: drag the right edge of Kind's header, as
	// TestTableResize does.
	kind := header(t, tt, "Kind")
	edge, y := float32(kind.X+kind.W)-2, float32(kind.Y+kind.H/2)
	tt.Press(edge, y)
	tt.Move(edge+30, y)
	tt.Release(edge+30, y)
	if !slices.Contains(seen, "columns") || !slices.Contains(calls, "columns") || slices.Contains(seen, "changed") {
		t.Errorf("resizing a column: saw %q, called %q", seen, calls)
	}
	seen, calls = nil, nil
	// Fitting it with a double click on the edge: reported in the frame
	// after, once the widths are known.
	kind = header(t, tt, "Kind")
	edge = float32(kind.X+kind.W) - 2
	tt.ClickAt(edge, y)
	tt.ClickAt(edge, y)
	if !slices.Contains(seen, "columns") || !slices.Equal(calls, []string{"columns"}) {
		t.Errorf("fitting a column: saw %q, called %q", seen, calls)
	}
	seen, calls = nil, nil

	// A table that does not choose still reports a double click.
	s2 := ListState{}
	var activated []int
	tt = coreNewTester(func(c *context) {
		e := coreTable(c, &s2, cols, len(names), func(row, col int) {
			if col == 0 {
				coreText(c, names[row]).SingleLine()
			}
		}).Grow(1)
		if row, ok := e.Activated(); ok {
			activated = append(activated, row)
		}
	}, 500, 300)
	tt.Click("b.txt")
	tt.Click("b.txt")
	if !slices.Equal(activated, []int{1}) {
		t.Errorf("a double click on a table that does not choose: %v", activated)
	}
}

func TestTableKeepsChoiceAcrossFarResort(t *testing.T) {
	const n = 100_000
	ids := make([]int, n)
	for i := range ids {
		ids[i] = i
	}
	sel := 0
	s := ListState{Selected: &sel, Key: func(i int) any { return ids[i] }}
	cols := []TableColumn{{Title: "ID"}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, cols, n, func(row, col int) { coreTextf(c, "Item %d", ids[row]) }).Grow(1)
	}, 400, 300)
	// Reversed: item 0 goes to the end, further than the list looks
	// without a new Version.
	slices.Reverse(ids)
	s.Version++
	tt.Frame()
	if sel != n-1 {
		t.Fatalf("the choice did not follow its item: row %d", sel)
	}
	// The place stays where it was, rather than following the top row.
	if first, _ := s.Visible(); first != 0 {
		t.Errorf("the table scrolled to row %d", first)
	}
}

func TestTableChecked(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End}}
	var checked Selection[string]
	shown := names
	sel := -1
	s := ListState{Checked: &checked, Selected: &sel, Key: func(i int) any { return shown[i] }}
	events := 0
	maxCol := -1
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			e := coreTable(c, &s, cols, len(shown), func(row, col int) {
				maxCol = max(maxCol, col)
				if col == 0 {
					coreText(c, shown[row]).SingleLine()
				}
			}).Grow(1)
			if e.CheckedChanged() {
				events++
			}
		})
	}, 500, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	boxes := func() []platform.AccessNode {
		var bs []platform.AccessNode
		for _, n := range tt.h.access.Nodes {
			if n.Role == platform.RoleCheckBox {
				bs = append(bs, n)
			}
		}
		return bs
	}
	click := func(n platform.AccessNode) {
		tt.ClickAt(float32(n.Bounds.X+n.Bounds.W/2), float32(n.Bounds.Y+n.Bounds.H/2))
	}
	header := func() platform.AccessNode { return accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all") }

	if maxCol != 2 {
		t.Fatalf("cell got columns up to %d, want the app's 0 to 2", maxCol)
	}
	if bs := boxes(); len(bs) != 4 || bs[0].Label != "Check all" {
		t.Fatalf("checkboxes: %+v", bs)
	}
	// The checkbox column comes first, before the app's columns.
	if h, name := header(), accessNode(t, tt.h.access, platform.RoleColumnHeader, "Name"); h.Bounds.X >= name.Bounds.X {
		t.Errorf("the checkbox column at %v, Name at %v", h.Bounds.X, name.Bounds.X)
	}
	click(boxes()[2]) // b.txt
	if !checked.Has("b.txt") || checked.Len() != 1 || events != 1 {
		t.Fatalf("checking b.txt: %v, %d events", checked.keys, events)
	}
	if sel != -1 {
		t.Errorf("checking a row chose row %d", sel)
	}
	if h := header(); h.States&platform.AccessMixed == 0 {
		t.Errorf("the header with one row checked: %+v", h)
	}
	click(header())
	if checked.Len() != 3 || events != 2 || header().States&platform.AccessChecked == 0 {
		t.Fatalf("checking all: %v, %d events, %+v", checked.keys, events, header())
	}
	click(header())
	if checked.Len() != 0 || events != 3 {
		t.Fatalf("unchecking all: %v, %d events", checked.keys, events)
	}
	// Rows not shown keep their keys: the header checks and unchecks the
	// rows shown only.
	checked.Add("c.txt")
	shown = names[:2]
	tt.Frame()
	if h := header(); h.States&(platform.AccessChecked|platform.AccessMixed) != 0 {
		t.Errorf("the header with none of the rows shown checked: %+v", h)
	}
	click(header())
	if checked.Len() != 3 {
		t.Fatalf("checking the rows shown: %v", checked.keys)
	}
	click(header())
	if checked.Len() != 1 || !checked.Has("c.txt") {
		t.Fatalf("unchecking the rows shown: %v", checked.keys)
	}
}

func TestTableRowMenu(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End}}
	var got [][]int
	menu := func(m *Menu, rows []int) {
		if m.Item("Open").Chosen() {
			got = append(got, slices.Clone(rows))
		}
	}
	open := func(tt *Tester) {
		t.Helper()
		if m := tt.Menu(); !slices.Equal(m, []string{"Open"}) {
			t.Fatalf("the menu: %q", m)
		}
		if err := tt.ChooseMenuItem("Open"); err != nil {
			t.Fatal(err)
		}
	}
	// One row at a time: a right click moves the choice.
	sel := 0
	s := ListState{Selected: &sel, RowMenu: menu}
	tt := coreNewTester(tableView(&s, cols, names), 500, 300)
	tt.RightClick("c.txt")
	if sel != 2 {
		t.Errorf("a right click on c.txt chose %d", sel)
	}
	open(tt)
	// Several rows: a right click on one not chosen chooses it alone, on
	// one chosen keeps them all.
	var chosen Selection[int]
	sel = -1
	s = ListState{Selected: &sel, Selection: &chosen, RowMenu: menu}
	tt = coreNewTester(tableView(&s, cols, names), 500, 300)
	tt.RightClick("b.txt")
	open(tt)
	tt.ClickWith(Cmd, "a.txt")
	tt.ClickWith(Cmd, "c.txt")
	tt.RightClick("c.txt")
	open(tt)
	// Shift+F10 opens it on the rows chosen.
	tt.Key(Shift, KeyF10)
	open(tt)
	// A table that does not choose: the row clicked.
	s = ListState{RowMenu: menu}
	tt = coreNewTester(tableView(&s, cols, names), 500, 300)
	tt.RightClick("b.txt")
	open(tt)
	want := [][]int{{2}, {1}, {0, 1, 2}, {0, 1, 2}, {1}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("the rows: %v, want %v", got, want)
	}
}

func TestTableCopy(t *testing.T) {
	names := []string{"photo.jpeg", "notes.md", "song.mp3"}
	cols := []TableColumn{{Title: "Name"}}
	var chosen Selection[int]
	sel := -1
	s := ListState{Selected: &sel, Selection: &chosen}
	s.Copy = func(rows []int) string {
		var b strings.Builder
		for _, r := range rows {
			b.WriteString(names[r] + "\n")
		}
		return b.String()
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, &s, cols, len(names), func(row, col int) { coreEditableText(c, &names[row]) }).Grow(1)
			coreButton(c, "Elsewhere")
		})
	}, 400, 300)
	tt.SetClipboard("before")
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "before" {
		t.Fatalf("Cmd+C without the focus nor a choice: %q", got)
	}
	// Rows chosen, the focus elsewhere: nothing is copied.
	tt.Click("photo.jpeg")
	tt.ClickWith(Shift, "song.mp3")
	tt.Click("Elsewhere")
	if tt.Focused("photo.jpeg") {
		t.Fatal("the table still has the focus")
	}
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "before" {
		t.Fatalf("Cmd+C with rows chosen but the focus elsewhere: %q", got)
	}
	tt.Command("copy")
	if got := tt.Clipboard(); got != "before" {
		t.Fatalf("Copy with rows chosen but the focus elsewhere: %q", got)
	}
	tt.Click("photo.jpeg")
	tt.ClickWith(Shift, "song.mp3")
	tt.Key(Cmd, KeyC)
	if got, want := tt.Clipboard(), "photo.jpeg\nnotes.md\nsong.mp3\n"; got != want {
		t.Fatalf("Cmd+C: %q, want %q", got, want)
	}
	tt.SetClipboard("")
	tt.Command("copy") // the Edit menu's Copy
	if got := tt.Clipboard(); got != "photo.jpeg\nnotes.md\nsong.mp3\n" {
		t.Fatalf("Copy: %q", got)
	}
	// A name being edited copies its text, not the rows.
	tt.Key(0, KeyHome)
	if runtime.GOOS == "darwin" {
		tt.Key(0, KeyEnter)
	} else {
		tt.Key(0, KeyF2)
	}
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "photo" {
		t.Fatalf("Cmd+C while renaming: %q", got)
	}
}

// checkedTable builds a table with a checkbox column, rows chosen and Copy,
// followed by a button, as the tests of the focus on it need.
func checkedTable(checked *Selection[int], events *int) (*ListState, *Tester) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}}
	var chosen Selection[int]
	sel := -1
	s := &ListState{Checked: checked, Selected: &sel, Selection: &chosen}
	s.Copy = func(rows []int) string {
		var b strings.Builder
		for _, r := range rows {
			b.WriteString(names[r] + "\n")
		}
		return b.String()
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			e := coreTable(c, s, cols, len(names), func(row, col int) { coreText(c, names[row]).SingleLine() }).Grow(1)
			if e.CheckedChanged() {
				*events++
			}
			coreButton(c, "After")
		})
	}, 500, 300)
	return s, tt
}

func TestTableCheckboxesKeepFocus(t *testing.T) {
	var checked Selection[int]
	events := 0
	s, tt := checkedTable(&checked, &events)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("a.txt")
	tt.ClickWith(Shift, "b.txt")
	var rowBox platform.AccessNode
	for _, n := range tt.h.access.Nodes {
		if n.Role == platform.RoleCheckBox && n.Label != "Check all" {
			rowBox = n // the last one, row c.txt
		}
	}
	tt.ClickAt(float32(rowBox.Bounds.X+rowBox.Bounds.W/2), float32(rowBox.Bounds.Y+rowBox.Bounds.H/2))
	if !checked.Has(2) || events != 1 {
		t.Fatalf("clicking the checkbox of c.txt: %v, %d events", checked.keys, events)
	}
	// The table kept the focus: Copy copies the rows chosen.
	tt.SetClipboard("")
	tt.Key(Cmd, KeyC)
	if got, want := tt.Clipboard(), "a.txt\nb.txt\n"; got != want {
		t.Fatalf("Cmd+C after clicking a checkbox: %q, want %q", got, want)
	}
	tt.SetClipboard("")
	tt.Command("copy")
	if got, want := tt.Clipboard(), "a.txt\nb.txt\n"; got != want {
		t.Fatalf("Copy after clicking a checkbox: %q, want %q", got, want)
	}
	// And Down moves the choice.
	tt.Key(0, KeyDown)
	if rows := s.frame.chosenRows(); !slices.Equal(rows, []int{2}) {
		t.Errorf("Down after clicking a checkbox chose rows %v, want [2]", rows)
	}
	// The header's checkbox keeps it too.
	tt.Key(0, KeyUp)
	head := accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all")
	tt.ClickAt(float32(head.Bounds.X+head.Bounds.W/2), float32(head.Bounds.Y+head.Bounds.H/2))
	tt.Key(0, KeyUp)
	if rows := s.frame.chosenRows(); !slices.Equal(rows, []int{0}) {
		t.Errorf("Up after clicking Check all chose rows %v, want [0]", rows)
	}
}

func TestTableTabLeavesInOneStep(t *testing.T) {
	var checked Selection[int]
	events := 0
	_, tt := checkedTable(&checked, &events)
	tt.Click("a.txt")
	tt.Key(0, KeyTab)
	if !tt.Focused("After") {
		t.Fatal("Tab from the table did not reach the button after it")
	}
}

func TestTableSpaceChecksRowsChosen(t *testing.T) {
	var checked Selection[int]
	events := 0
	s, tt := checkedTable(&checked, &events)
	tt.Click("a.txt")
	tt.ClickWith(Cmd, "c.txt") // adds c.txt to the rows chosen
	want := s.frame.chosenRows()
	if !slices.Equal(want, []int{0, 2}) {
		t.Fatalf("rows chosen: %v", want)
	}
	tt.Key(0, KeySpace)
	if checked.Len() != 2 || !checked.Has(0) || !checked.Has(2) || events != 1 {
		t.Fatalf("Space: checked %v, %d events", checked.keys, events)
	}
	tt.Key(0, KeySpace)
	if checked.Len() != 0 || events != 2 {
		t.Fatalf("Space again: checked %v, %d events", checked.keys, events)
	}
	// Some of them checked already: Space checks the rest.
	checked.Add(0)
	tt.Frame()
	events = 0
	tt.Key(0, KeySpace)
	if checked.Len() != 2 || events != 1 {
		t.Fatalf("Space with one of two checked: %v, %d events", checked.keys, events)
	}
}

func TestTableCheckboxLabels(t *testing.T) {
	names := []string{"a.txt", "b.txt"}
	var checked Selection[int]
	sel := -1
	s := ListState{Checked: &checked, Selected: &sel}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, []TableColumn{{Title: "Name"}}, len(names), func(row, col int) { coreText(c, names[row]) }).Grow(1)
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	accessNode(t, tt.h.access, platform.RoleCheckBox, "Check")
	// With Label, the checkboxes tell the rows apart.
	s.Label = func(i int) string { return names[i] }
	tt.Frame()
	accessNode(t, tt.h.access, platform.RoleCheckBox, "Check b.txt")
	// Space checks with Label set too, where typing selects rows.
	tt.Click("a.txt")
	tt.Key(0, KeySpace)
	if !checked.Has(0) || checked.Len() != 1 {
		t.Errorf("Space with Label: checked %v", checked.keys)
	}
}

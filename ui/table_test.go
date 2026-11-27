package ui

import (
	"fmt"
	"runtime"
	"slices"
	"strconv"
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

// FitColumns fits several columns in the next frame, each to its header and
// its cells, as a double click on an edge does, and ignores IDs of no
// column and of Fixed columns; Changed reports the new layout.
func TestTableFitColumns(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name", Width: 40}, {Title: "Kind", Width: 300}, {Title: "Size", Width: 80, Fixed: true}, {Title: "A long title, wider than its cells", Width: 30}}
	changed := 0
	names := []string{"a.txt", "a much longer name.txt"}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			if coreTable(c, &s, cols, len(names), func(row, col int) {
				switch col {
				case 0:
					coreText(c, names[row]).SingleLine()
				case 1:
					coreText(c, "Document")
				case 2:
					coreTextf(c, "%d KB", row+1)
				case 3:
					coreText(c, "x")
				}
			}).Grow(1).Changed() {
				changed++
			}
		})
	}, 900, 300)
	s.FitColumns("Name", "Kind", "Size", "Nope", "A long title, wider than its cells", "Kind")
	tt.Frame()
	tt.Frame()
	// Measured once whole, the columns fitted.
	long, _ := tt.Find("a much longer name.txt")
	doc, _ := tt.Find("Document")
	title, _ := tt.Find("A long title, wider than its cells")
	if long.W < 100 || title.W < 100 {
		t.Fatalf("the texts measured %v and %v: still cut off", long.W, title.W)
	}
	if w := s.Columns.Widths["Name"]; w < long.W+20 || w > long.W+21 {
		t.Errorf("Name fitted to %v for a text of %v", w, long.W)
	}
	if w := s.Columns.Widths["Kind"]; w < doc.W+20 || w > doc.W+21 {
		t.Errorf("Kind fitted to %v for a text of %v", w, doc.W)
	}
	if w := s.Columns.Widths["A long title, wider than its cells"]; w < title.W+19.9 {
		t.Errorf("the column of a long title fitted to %v, not to its header of %v", w, title.W)
	}
	if _, ok := s.Columns.Widths["Size"]; ok {
		t.Errorf("a Fixed column fitted: %v", s.Columns.Widths)
	}
	if _, ok := s.Columns.Widths["Nope"]; ok {
		t.Errorf("an ID of no column fitted: %v", s.Columns.Widths)
	}
	if changed != 1 {
		t.Errorf("Changed reported %d times, want once", changed)
	}
	// Nothing more asked: the next frames fit nothing.
	s.Columns.Widths = map[string]float32{"Name": 40}
	tt.Frame()
	tt.Frame()
	if w := s.Columns.Widths["Name"]; w != 40 {
		t.Errorf("Name fitted again to %v without FitColumns", w)
	}
}

// A double click on an edge of a table without rows fits the column to its
// header.
func TestTableFitWithoutRows(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name", Width: 100}, {Title: "Kind", Width: 100}}
	tt := coreNewTester(tableView(&s, cols, nil), 500, 300)
	name := header(t, tt, "Name")
	edge := float32(name.X+name.W) - 2
	tt.ClickAt(edge, float32(name.Y+name.H/2))
	tt.ClickAt(edge, float32(name.Y+name.H/2))
	tt.Frame()
	title, _ := tt.Find("Name")
	if w := s.Columns.Widths["Name"]; w < title.W+19.9 || w > title.W+21 || w > 90 {
		t.Fatalf("Name fitted to %v for a header of %v", w, title.W)
	}
	if _, ok := s.Columns.Widths["Kind"]; ok {
		t.Fatalf("Kind fitted too: %v", s.Columns.Widths)
	}
}

// FixedColumnOrder: a header dragged past its neighbor stays where it is;
// one released on itself is a click, which sorts, as a Fixed column's is;
// the edge still resizes.
func TestTableFixedColumnOrder(t *testing.T) {
	sort := SortOrder{}
	s := ListState{FixedColumnOrder: true, Sort: &sort}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100, Sortable: true}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	kind := header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)
	x := float32(kind.X + 20)
	drag := func(to float32) {
		tt.Press(x, y)
		for _, d := range []float32{5, 20, to} {
			tt.Move(x+d, y)
		}
		tt.Release(x+to, y)
	}
	drag(90) // past the middle of Size
	if got := s.Columns.Order; got != nil {
		t.Fatalf("the order: %q", got)
	}
	if sort.Column != "" {
		t.Fatalf("a drag released on Size sorted by %q", sort.Column)
	}
	drag(30) // released on Kind
	if s.Columns.Order != nil || sort.Column != "Kind" {
		t.Fatalf("released on its header: the order %q, sorted by %q", s.Columns.Order, sort.Column)
	}
	edge := float32(kind.X+kind.W) - 2
	tt.Press(edge, y)
	tt.Move(edge+30, y)
	tt.Release(edge+30, y)
	if w := s.Columns.Widths["Kind"]; w != 130 || s.Columns.Order != nil {
		t.Fatalf("the edge dragged 30 DIPs: width %v, order %q", w, s.Columns.Order)
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
	// The release's own move moves the column too, and is no click.
	x = float32(header(t, tt, "Kind").X + 20)
	tt.Press(x, y)
	tt.Move(x-5, y)
	tt.Move(x-20, y)
	tt.Release(x-60, y)
	if got := s.Columns.Order; got[1] != "Kind" || got[2] != "Size" {
		t.Errorf("Kind let go past the middle of Size: %q", got)
	}
	if sort.Column != "" {
		t.Errorf("dragging the header sorted by %q", sort.Column)
	}
}

// Changed reports a column moved or resized once the user lets go, when
// the layout is new, and a column fitted to its cells, not as the pointer
// moves: the app saves ListState.Columns once per gesture.
func TestTableColumnsChanged(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80}}
	names := []string{"a.txt"}
	changes := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			if coreTable(c, &s, cols, len(names), func(row, col int) {
				coreText(c, []string{names[row], "Document", "1 KB"}[col]).SingleLine()
			}).Grow(1).Changed() {
				changes++
			}
		})
	}, 500, 300)
	expect := func(n int, what string) {
		t.Helper()
		if changes != n {
			t.Fatalf("%s: Changed reported %d times, want %d (%+v)", what, changes, n, s.Columns)
		}
	}
	kind := header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)

	// Resizing Kind by its edge: reported on release only.
	edge := float32(kind.X+kind.W) - 2
	tt.Press(edge, y)
	tt.Move(edge+10, y)
	tt.Move(edge+30, y)
	expect(0, "resizing, held")
	tt.Release(edge+30, y)
	expect(1, "resized")
	if w := s.Columns.Widths["Kind"]; w != 130 {
		t.Fatalf("Kind is %v wide", w)
	}
	// A press on the edge that comes back where it started changes nothing.
	edge += 30
	tt.Press(edge, y)
	tt.Move(edge+20, y)
	tt.Move(edge, y)
	tt.Release(edge, y)
	expect(1, "resized and back")

	// A double click on the edge fits the column to its cells: one change,
	// and none for a column already fitted.
	tt.ClickAt(edge, y)
	tt.ClickAt(edge, y)
	expect(2, "fitted")
	kind = header(t, tt, "Kind")
	edge = float32(kind.X+kind.W) - 2
	tt.ClickAt(edge, y)
	tt.ClickAt(edge, y)
	expect(2, "fitted again")

	// Moving Kind past Size: reported on release only.
	x := float32(kind.X + 20)
	tt.Press(x, y)
	for _, d := range []float32{5, 20, 50, 60} {
		tt.Move(x+d, y)
	}
	if got := s.Columns.Order; len(got) != 3 || got[2] != "Kind" {
		t.Fatalf("the order while held: %q", got)
	}
	expect(2, "moving, held")
	tt.Release(x+60, y)
	expect(3, "moved")
	// A drag that leaves the column where it was changes nothing.
	x = float32(header(t, tt, "Kind").X + 20)
	tt.Press(x, y)
	tt.Move(x-5, y)
	tt.Move(x-10, y)
	tt.Release(x-10, y)
	if got := s.Columns.Order; got[2] != "Kind" {
		t.Fatalf("Kind moved: %q", got)
	}
	expect(3, "dragged in place")
	// Nor does a click on a header that does not sort.
	tt.Click("Size")
	expect(3, "clicked")
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

// The last column sharing the room left ends at the table's end: its
// header has no edge, as AppKit's. A last column with a width keeps its.
func TestTableLastColumnEdge(t *testing.T) {
	// edge reports whether the end of a column's header resizes it, its
	// pointer moving there and dragging it 30 DIPs out.
	edge := func(tt *Tester, s *ListState, title string, rtl bool) bool {
		h := header(t, tt, title)
		x, y, dx := float32(h.X+h.W)-2, float32(h.Y+h.H/2), float32(-30)
		if rtl {
			x, dx = float32(h.X)+2, 30
		}
		tt.Move(x, y)
		cursor := tt.Cursor() == CursorResizeEW
		before := header(t, tt, title).W
		tt.Press(x, y)
		tt.Move(x+dx, y)
		tt.Release(x+dx, y)
		after := header(t, tt, title).W
		if cursor != (after != before) {
			t.Errorf("%s: the resize pointer %v, but dragged from %v to %v", title, cursor, before, after)
		}
		return cursor
	}
	for _, rtl := range []bool{false, true} {
		s := ListState{}
		cols := []TableColumn{{Title: "Name", Width: 150}, {Title: "Kind", Width: 100}, {Title: "Size"}}
		tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
		if rtl {
			tt.SetDirection(RightToLeft)
		}
		if edge(tt, &s, "Size", rtl) {
			t.Errorf("rtl %v: the last column, sharing the room left, has an edge", rtl)
		}
		if _, ok := s.Columns.Widths["Size"]; ok {
			t.Errorf("rtl %v: the last column took a width: %v", rtl, s.Columns.Widths)
		}
		if !edge(tt, &s, "Kind", rtl) {
			t.Errorf("rtl %v: a column with a width before it has no edge", rtl)
		}
		// Moved before the others, it has its edge back, and the last one,
		// with a width, keeps its own.
		s.Columns.Order = []string{"Size", "Name", "Kind"}
		tt.Frame()
		if !edge(tt, &s, "Size", rtl) || !edge(tt, &s, "Kind", rtl) {
			t.Errorf("rtl %v: moved first, Size or the last column, Kind, has no edge", rtl)
		}
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
	tt.Advance(renameDelay + 50*time.Millisecond)
	tt.Type("sea")
	tt.Click("notes.txt")
	if names[1] != "sea.jpeg" || sel != 0 {
		t.Errorf("after a click elsewhere: %q, chose %d", names[1], sel)
	}
	// A double click opens, and does not edit.
	tt.Click("notes.txt")
	tt.Click("notes.txt")
	tt.Advance(renameDelay + 50*time.Millisecond)
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
	second := 0
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
			// Callbacks add up: another one runs along with the first.
			wrapElement(e).OnSelectionChange(func() { second++ })
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
	if second != 1 {
		t.Errorf("the second OnSelectionChange ran %d times for one click, want 1", second)
	}
	tt.Click("Name")
	check("a click on a header", []string{"changed", "sort"}, []string{"sort"})
	tt.Frame()
	check("a frame after the sort", nil, nil)
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
	if !slices.Contains(seen, "columns") || !slices.Contains(calls, "columns") {
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
	// Moving a column: Kind dragged past Size reports the columns, not a
	// change of the choice nor of the sort.
	kind, size := header(t, tt, "Kind"), header(t, tt, "Size")
	from, to := float32(kind.X+20), float32(size.X+size.W/2+10)
	tt.Press(from, y)
	for _, f := range []float32{0.1, 0.4, 0.8, 1} {
		tt.Move(from+(to-from)*f, y)
	}
	tt.Release(to, y)
	if !slices.Contains(seen, "columns") || !slices.Equal(calls, []string{"columns"}) {
		t.Errorf("moving a column: saw %q, called %q", seen, calls)
	}
	if k, z := header(t, tt, "Kind"), header(t, tt, "Size"); z.X >= k.X {
		t.Errorf("Size at %v, Kind at %v: the column did not move", z.X, k.X)
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
	// The row and its checkbox share one call of Label.
	calls := 0
	s.Label = func(i int) string { calls++; return names[i] }
	tt.Frame()
	if calls != len(names) {
		t.Errorf("Label called %d times for %d rows", calls, len(names))
	}
}

func TestTableHeaderCheckboxWithoutRowsToCheck(t *testing.T) {
	for _, tc := range []struct {
		name  string
		n     int
		isHdr func(int) bool
	}{
		{"no rows", 0, nil},
		{"only section headers", 3, func(int) bool { return true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var checked Selection[int]
			sel := -1
			s := ListState{Checked: &checked, Selected: &sel, Header: tc.isHdr}
			events := 0
			tt := coreNewTester(func(c *context) {
				e := coreTable(c, &s, []TableColumn{{Title: "Name"}}, tc.n, func(row, col int) { coreText(c, "Row") }).Grow(1)
				if e.CheckedChanged() {
					events++
				}
			}, 400, 300)
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			head := accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all")
			tt.ClickAt(float32(head.Bounds.X+head.Bounds.W/2), float32(head.Bounds.Y+head.Bounds.H/2))
			if events != 0 || checked.Len() != 0 {
				t.Errorf("%d CheckedChanged events, checked %v", events, checked.keys)
			}
		})
	}
}

func TestTableSpaceInTypedPrefix(t *testing.T) {
	names := []string{"my doc", "my docs", "other"}
	var checked Selection[int]
	sel := -1
	s := ListState{Checked: &checked, Selected: &sel, Label: func(i int) string { return names[i] }}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, []TableColumn{{Title: "Name"}}, len(names), func(row, col int) { coreText(c, names[row]).SingleLine() }).Grow(1)
	}, 400, 300)
	tt.Click("other")
	tt.Key(0, KeyM)
	tt.Key(0, KeyY)
	tt.Key(0, KeySpace)
	tt.Key(0, KeyD)
	if sel != 0 || checked.Len() != 0 {
		t.Fatalf("my d: the choice is %d, checked %v; want 0, none", sel, checked.keys)
	}
	for _, k := range []Key{KeyO, KeyC, KeyS} {
		tt.Key(0, k)
	}
	if sel != 1 || checked.Len() != 0 {
		t.Fatalf("my docs: the choice is %d, checked %v; want 1, none", sel, checked.keys)
	}
	// Once the pause is over, Space checks again.
	tt.rt.states[s.frame.owner.id].typedAt = time.Time{}
	tt.Key(0, KeySpace)
	if !checked.Has(1) || checked.Len() != 1 {
		t.Errorf("Space with no prefix pending checked %v, want row 1", checked.keys)
	}
}

func TestTableCopyEdges(t *testing.T) {
	names := []string{"a.txt", "b.txt"}
	var chosen Selection[int]
	sel := -1
	text, calls := "", 0
	s := ListState{Selected: &sel, Selection: &chosen, Copy: func(rows []int) string { calls++; return text }}
	tt := coreNewTester(tableView(&s, []TableColumn{{Title: "Name"}}, names), 400, 300)
	tt.Click("a.txt")
	tt.SetClipboard("before")
	tt.Key(Cmd, KeyC)
	if calls != 1 || tt.Clipboard() != "before" {
		t.Errorf("Copy returning nothing: %d calls, clipboard %q; want 1, \"before\"", calls, tt.Clipboard())
	}
	tt.Command("copy")
	if tt.Clipboard() != "before" {
		t.Errorf("the Edit menu's Copy returning nothing: clipboard %q", tt.Clipboard())
	}
	// A key held down repeats: one copy.
	text, calls = "a.txt\n", 0
	for _, repeat := range []bool{false, true, true} {
		tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyC), Mods: platform.Modifiers(Cmd), Repeat: repeat})
	}
	if calls != 1 || tt.Clipboard() != "a.txt\n" {
		t.Errorf("a repeated Cmd+C: %d calls, clipboard %q; want 1, \"a.txt\\n\"", calls, tt.Clipboard())
	}
}

func TestTableVersionNearReSort(t *testing.T) {
	const n = 100
	ids := make([]int, n)
	for i := range ids {
		ids[i] = i
	}
	sel := 50
	s := ListState{Selected: &sel, Key: func(i int) any { return ids[i] }}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, []TableColumn{{Title: "ID"}}, n, func(row, col int) { coreTextf(c, "Item %d", ids[row]) }).Grow(1)
	}, 400, 300)
	s.ScrollTo(50, Start)
	tt.Frame()
	if first, _ := s.Visible(); first != 50 {
		t.Fatalf("scrolled to row %d, want 50", first)
	}
	// The item at the top moves 40 rows, within reach of a search.
	ids[50], ids[10] = ids[10], ids[50]
	s.Version++
	tt.Frame()
	if first, _ := s.Visible(); first != 50 {
		t.Errorf("the first row shown is %d, want 50: the place stays by index", first)
	}
	if sel != 10 {
		t.Errorf("the choice is row %d, want 10: it follows its key", sel)
	}
}

// checkAllNode returns a table's header checkbox.
func checkAllNode(t *testing.T, tt *Tester) platform.AccessNode {
	t.Helper()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	return accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all")
}

func TestTableCheckedHeaderCount(t *testing.T) {
	// (a) Section headers are not counted.
	rows := []string{"Section", "a", "b", "Section", "c"}
	var checked Selection[int]
	sel := -1
	s := ListState{Checked: &checked, Selected: &sel, Header: func(i int) bool { return rows[i] == "Section" }}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, []TableColumn{{Title: "Name"}}, len(rows), func(row, col int) { coreText(c, rows[row]).SingleLine() }).Grow(1)
	}, 400, 300)
	checked.Add(1)
	checked.Add(2)
	tt.Frame()
	if st := checkAllNode(t, tt).States; st&platform.AccessMixed == 0 || st&platform.AccessChecked != 0 {
		t.Errorf("two of three rows checked: states %b, want mixed", st)
	}
	checked.Add(4)
	tt.Frame()
	if st := checkAllNode(t, tt).States; st&platform.AccessChecked == 0 {
		t.Errorf("all rows but the section headers checked: states %b, want checked", st)
	}

	// (b) Other rows shown, as many, with a new Version.
	sets := [][]string{{"a", "b", "c"}, {"x", "y", "z"}}
	shown := 0
	var ck Selection[string]
	s2 := ListState{Checked: &ck, Selected: &sel, Key: func(i int) any { return sets[shown][i] }}
	tt = coreNewTester(func(c *context) {
		coreTable(c, &s2, []TableColumn{{Title: "Name"}}, 3, func(row, col int) { coreText(c, sets[shown][row]).SingleLine() }).Grow(1)
	}, 400, 300)
	for _, k := range sets[0] {
		ck.Add(k)
	}
	tt.Frame()
	if st := checkAllNode(t, tt).States; st&platform.AccessChecked == 0 {
		t.Fatalf("the first set, all checked: states %b", st)
	}
	shown = 1
	s2.Version++
	tt.Frame()
	if st := checkAllNode(t, tt).States; st&(platform.AccessChecked|platform.AccessMixed) != 0 {
		t.Errorf("the second set, none checked: states %b", st)
	}
	shown = 0
	s2.Version++
	tt.Frame()
	if st := checkAllNode(t, tt).States; st&platform.AccessChecked == 0 {
		t.Errorf("the first set again: states %b", st)
	}
}

func TestTableCheckedColumnStaysFirst(t *testing.T) {
	var checked Selection[int]
	s := ListState{Checked: &checked}
	cols := []TableColumn{{Title: "Name", Width: 120}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	name, kind := header(t, tt, "Name"), header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)
	from, to := float32(kind.X+20), float32(name.X+10)
	tt.Press(from, y)
	for _, f := range []float32{0.1, 0.4, 0.8, 1} {
		tt.Move(from+(to-from)*f, y)
	}
	tt.Release(to, y)
	if got := s.Columns.Order; len(got) == 0 || got[0] != "Kind" {
		t.Fatalf("the order: %q, want Kind first and no checkbox column", got)
	}
	box := checkAllNode(t, tt).Bounds.X
	for _, title := range []string{"Name", "Kind", "Size"} {
		if h := header(t, tt, title); box >= h.X {
			t.Errorf("the checkbox column at %v, %s at %v", box, title, h.X)
		}
	}
}

func TestTableRowMenuEdges(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}}
	var got [][]int
	menu := func(m *Menu, rows []int) {
		if m.Item("Open").Chosen() {
			got = append(got, slices.Clone(rows))
		}
	}
	sel := -1
	s := ListState{Selected: &sel, RowMenu: menu}
	var seen []string
	tt := coreNewTester(func(c *context) {
		e := coreTable(c, &s, cols, len(names), func(row, col int) { coreText(c, names[row]).SingleLine() }).Grow(1)
		if e.SelectionChanged() {
			seen = append(seen, "selection")
		}
		if e.Changed() {
			seen = append(seen, "changed")
		}
	}, 500, 300)
	// (a) A right click on a row not chosen chooses it, and says so.
	tt.RightClick("b.txt")
	if sel != 1 || !slices.Equal(seen, []string{"selection", "changed"}) {
		t.Errorf("a right click on a row not chosen: the choice %d, events %q", sel, seen)
	}
	if err := tt.ChooseMenuItem("Open"); err != nil {
		t.Fatal(err)
	}
	// (b) Below the last row, inside the table: the menu acts on the rows chosen.
	b, _ := tt.Find("b.txt")
	tt.RightClickAt(b.X+b.W/2, 280)
	if m := tt.Menu(); !slices.Equal(m, []string{"Open"}) {
		t.Fatalf("a right click below the rows: the menu %q", m)
	}
	if err := tt.ChooseMenuItem("Open"); err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(got, [][]int{{1}, {1}}, slices.Equal) {
		t.Errorf("right clicks on b.txt and below the rows acted on %v, want [[1] [1]]", got)
	}
	// (c) Shift+F10 with nothing chosen opens none.
	tt.Click("b.txt") // for the focus
	tt.Key(Shift, KeyF10)
	if tt.Menu() == nil {
		t.Fatal("Shift+F10 with a row chosen opened no menu")
	}
	if err := tt.ChooseMenuItem("Open"); err != nil {
		t.Fatal(err)
	}
	sel = -1
	tt.Frame()
	tt.Key(Shift, KeyF10)
	if m := tt.Menu(); m != nil {
		t.Errorf("Shift+F10 with nothing chosen opened %q", m)
	}
}

// With SortBy, a Cmd-click sorts by several columns: it adds a column,
// ascending, after those sorted by, makes it descending, then takes it
// out. A click sorts by one column again.
func TestTableSortBy(t *testing.T) {
	var by []SortOrder
	s := ListState{SortBy: &by}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100, Sortable: true}, {Title: "Size", Width: 80, Sortable: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "b.txt"}), 500, 300)
	check := func(step string, want ...SortOrder) {
		t.Helper()
		if !slices.Equal(by, want) {
			t.Fatalf("%s: %+v, want %+v", step, by, want)
		}
	}
	tt.Click("Name")
	check("a click on Name", SortOrder{Column: "Name"})
	tt.ClickWith(Cmd, "Kind")
	check("a Cmd-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Kind"})
	kept := by
	tt.ClickWith(Cmd, "Size")
	check("a Cmd-click on Size", SortOrder{Column: "Name"}, SortOrder{Column: "Kind"}, SortOrder{Column: "Size"})
	if len(kept) != 2 {
		t.Fatalf("a click changed the slice the app kept: %+v", kept)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleColumnHeader, "Size"); n.States&platform.AccessSortAscending == 0 || n.Label != "Size" {
		t.Errorf("Size, sorted by after Name and Kind: %+v", n)
	}
	tt.ClickWith(Cmd, "Kind")
	check("a second Cmd-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Kind", Descending: true}, SortOrder{Column: "Size"})
	tt.ClickWith(Cmd, "Kind")
	check("a third Cmd-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Size"})
	// The first column goes the same way, the next taking its place.
	tt.ClickWith(Cmd, "Name")
	check("a Cmd-click on Name", SortOrder{Column: "Name", Descending: true}, SortOrder{Column: "Size"})
	tt.ClickWith(Cmd, "Name")
	check("a second Cmd-click on Name", SortOrder{Column: "Size"})
	// A click sorts by one column again, and on the first reverses it.
	tt.ClickWith(Cmd, "Kind")
	tt.Click("Kind")
	check("a click on Kind", SortOrder{Column: "Kind"})
	tt.ClickWith(Cmd, "Name")
	tt.Click("Kind")
	check("a click on Kind, first", SortOrder{Column: "Kind", Descending: true})
}

func TestTableHeaderMenu(t *testing.T) {
	built, chosen := -1, -1
	s := ListState{Columns: TableLayout{Order: []string{"Size", "Name", "Kind"}}}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	// Without HeaderMenu, a header has no menu.
	tt.RightClick("Kind")
	if m := tt.Menu(); m != nil {
		t.Fatalf("a menu without HeaderMenu: %q", m)
	}
	s.HeaderMenu = func(m *Menu, col int) {
		built = col
		if m.Item("Hide Column").Chosen() {
			chosen = col
		}
	}
	tt.Frame()
	// Kind shows third, and is the second of the columns.
	tt.RightClick("Kind")
	if m := tt.Menu(); len(m) != 1 || m[0] != "Hide Column" || built != 1 {
		t.Fatalf("the menu %q, built for column %d", m, built)
	}
	if err := tt.ChooseMenuItem("Hide Column"); err != nil {
		t.Fatal(err)
	}
	if chosen != 1 {
		t.Errorf("chose the item for column %d", chosen)
	}
}

// A right-click choosing a row before its menu opens, between frames,
// reports the new choice in the next frame, as a click does.
func TestTableRightClickReportsSelection(t *testing.T) {
	names := []string{"a.txt", "b.txt"}
	var chosen Selection[string]
	s := ListState{Selection: &chosen, Key: func(i int) any { return names[i] },
		RowMenu: func(m *Menu, rows []int) { m.Item("Open") }}
	seen, calls := 0, 0
	tt := coreNewTester(func(c *context) {
		e := coreTable(c, &s, []TableColumn{{Title: "Name"}}, len(names), func(row, col int) { coreText(c, names[row]) }).Grow(1)
		if e.SelectionChanged() {
			seen++
		}
		wrapElement(e).OnSelectionChange(func() { calls++ })
	}, 400, 300)
	tt.RightClick("b.txt")
	if !chosen.Has("b.txt") || seen != 1 || calls != 1 {
		t.Fatalf("right-click on b.txt: chosen %v, SelectionChanged %d, OnSelectionChange %d; want b.txt, 1, 1", slices.Collect(chosen.All()), seen, calls)
	}
}

// TestTableRowSelected tells a cell whether its row is chosen, so it can
// leave the selection's text color alone, and nothing outside a row.
func TestTableRowSelected(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	sel := -1
	s := ListState{Selected: &sel}
	seen := map[int]bool{}
	outside := true
	tt := coreNewTester(func(c *context) {
		outside = c.RowSelected()
		coreTable(c, &s, []TableColumn{{Title: "Name"}}, len(names), func(row, col int) {
			seen[row] = c.RowSelected()
			coreText(c, names[row])
		}).Grow(1)
	}, 400, 300)
	tt.Click("b.txt")
	tt.Frame()
	if outside || seen[0] || !seen[1] || seen[2] {
		t.Fatalf("RowSelected: outside %v, rows %v", outside, seen)
	}
}

// CheckedCount, when set, is what the header's checkbox shows, and the
// table counts no rows itself.
func TestTableCheckedCount(t *testing.T) {
	names := []string{"a", "b", "c"}
	var checked Selection[string]
	counted := 0
	s := ListState{Checked: &checked, Key: func(i int) any { return names[i] },
		CheckedCount: func() (int, int) { counted++; return 3, 3 }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, &s, []TableColumn{{Title: "Name"}}, len(names), func(row, col int) { coreText(c, names[row]) }).Grow(1)
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	if counted == 0 {
		t.Fatal("the table never asked CheckedCount")
	}
	if st := accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all").States; st&platform.AccessChecked == 0 {
		t.Fatalf("header checkbox states %b with CheckedCount all of 3, want checked", st)
	}
}

// Set with SortBy, Sort follows its first column, for the app to read
// the column that sorts first; a Shift-click adds a column as a Cmd-click
// does, as web grids do.
func TestTableSortFollowsSortBy(t *testing.T) {
	var by []SortOrder
	var sort SortOrder
	s := ListState{Sort: &sort, SortBy: &by}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100, Sortable: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "b.txt"}), 500, 300)
	check := func(step string, first SortOrder, want ...SortOrder) {
		t.Helper()
		if !slices.Equal(by, want) || sort != first {
			t.Fatalf("%s: SortBy %+v, Sort %+v; want %+v, %+v", step, by, sort, want, first)
		}
	}
	tt.Click("Name")
	check("a click on Name", SortOrder{Column: "Name"}, SortOrder{Column: "Name"})
	tt.ClickWith(Shift, "Kind")
	check("a Shift-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Name"}, SortOrder{Column: "Kind"})
	tt.ClickWith(Cmd, "Name")
	tt.ClickWith(Shift, "Name")
	check("Name taken out", SortOrder{Column: "Kind"}, SortOrder{Column: "Kind"})
	tt.ClickWith(Shift, "Kind")
	tt.ClickWith(Shift, "Kind")
	check("sorted by none", SortOrder{})
	// The app setting SortBy itself.
	by = []SortOrder{{Column: "Kind", Descending: true}}
	tt.Frame()
	check("SortBy set by the app", SortOrder{Column: "Kind", Descending: true}, SortOrder{Column: "Kind", Descending: true})
}

// The header past the columns has HeaderMenu's menu for column -1, which
// does not open when it has no items.
func TestTableHeaderMenuPastTheColumns(t *testing.T) {
	built := 0
	items := true
	s := ListState{HeaderMenu: func(m *Menu, col int) {
		built = col
		if items {
			m.Item("Reset Columns")
		}
	}}
	cols := []TableColumn{{Title: "Name", Width: 120}, {Title: "Kind", Width: 100}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	kind, _ := tt.Find("Kind")
	tt.RightClickAt(450, kind.Y+kind.H/2)
	if m := tt.Menu(); len(m) != 1 || m[0] != "Reset Columns" || built != -1 {
		t.Fatalf("past the columns: the menu %q, built for column %d", m, built)
	}
	tt.CloseMenu()
	items = false
	tt.RightClickAt(450, kind.Y+kind.H/2)
	if m := tt.Menu(); m != nil {
		t.Fatalf("a menu without items opened: %q", m)
	}
	items = true
	tt.RightClick("Kind")
	if built != 1 {
		t.Fatalf("a column's header built the menu for column %d", built)
	}
}

// CellMenu gives the cells a context menu over the whole cell, its padding
// too, told the row and the column as the cell builder receives them.
func TestTableCellMenu(t *testing.T) {
	var s ListState
	cols := []TableColumn{{Title: "A", Width: 100, Fixed: true}, {Title: "B", Width: 100, Fixed: true}}
	gotRow, gotCol := -1, -1
	s.CellMenu = func(m *Menu, row, col int) {
		gotRow, gotCol = row, col
		if col == 1 {
			m.Item("Open")
		}
	}
	tt := NewTester(func(c *Context) {
		Table(c, &s, cols, 4, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	r, _ := tt.Find("r2c1")
	tt.RightClickAt(r.X+r.W+20, r.Y+r.H/2) // past the text, in the cell
	if m := tt.Menu(); len(m) != 1 || m[0] != "Open" || gotRow != 2 || gotCol != 1 {
		t.Fatalf("right-click in r2c1 past its text: menu %q, row %d, col %d", m, gotRow, gotCol)
	}
	tt.CloseMenu()
	r, _ = tt.Find("r1c0")
	tt.RightClickAt(r.X+2, r.Y+r.H/2)
	if m := tt.Menu(); m != nil || gotRow != 1 || gotCol != 0 {
		t.Fatalf("a menu built without items opened: %q (row %d, col %d)", m, gotRow, gotCol)
	}
}

// HeaderCell gets each column's header cell as the table builds it: what
// it builds goes in the cell after the title, and col is the column's
// index in the columns given to Table.
func TestTableHeaderCell(t *testing.T) {
	hovered := -1
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80}}
	tt := coreNewTester(func(c *context) {
		s.HeaderCell = func(cell Element, col int) {
			coreTextf(c, "mark %d", col)
			if cell.Hovered() {
				hovered = col
			}
		}
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, &s, cols, 1, func(row, col int) { coreText(c, "a.txt") }).Grow(1)
		})
	}, 500, 300)
	if !tt.HasText("mark 0") || !tt.HasText("mark 2") {
		t.Fatalf("marks: %q", tt.Texts())
	}
	name, _ := tt.Find("Name")
	mark, _ := tt.Find("mark 0")
	if mark.X < name.X+name.W || mark.Y > name.Y+name.H {
		t.Errorf("the mark at %+v, not after the title at %+v", mark, name)
	}
	size, _ := tt.Find("Size")
	tt.Move(size.X+2, size.Y+size.H/2)
	tt.Frame()
	if hovered != 2 {
		t.Fatalf("hovered %d", hovered)
	}
}

// With RowHeight every row is that high, a short one as a tall one would
// be; the header keeps the theme's height.
func TestTableRowHeight(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80, Align: End}}
	s := ListState{}
	tt := coreNewTester(tableView(&s, cols, names), 500, 300)
	r0, _ := rowBox(tt, &s, 0)
	if want := LightTheme().Space(8); r0.H != want {
		t.Fatalf("a row by default is %v high, want %v", r0.H, want)
	}
	s = ListState{RowHeight: 22}
	tt = coreNewTester(tableView(&s, cols, names), 500, 300)
	for i := range names {
		r, _ := rowBox(tt, &s, i)
		if r.H != 22 {
			t.Errorf("row %d is %v high, want 22", i, r.H)
		}
		if i > 0 {
			prev, _ := rowBox(tt, &s, i-1)
			if r.Y != prev.Y+22 {
				t.Errorf("row %d starts at %v, after one at %v", i, r.Y, prev.Y)
			}
		}
	}
	// The header above the first row keeps the theme's height.
	if r, _ := rowBox(tt, &s, 0); r.Y != r0.Y {
		t.Errorf("the first row starts at %v, not %v below the header", r.Y, r0.Y)
	}
	if !tt.HasText("b.txt") {
		t.Errorf("texts %q", tt.Texts())
	}
}

// wideTable shows a table of n rows and count columns of width wide, the
// first of them frozen of those that frozen says, whose cell r2c5 is row 2's
// in column 5, and counts the cells built, by column.
func wideTable(s *ListState, count int, width float32, frozen int, built map[int]int) func(c *context) {
	cols := make([]TableColumn, count)
	for j := range cols {
		cols[j] = TableColumn{Title: "H" + strconv.Itoa(j), Width: width, Frozen: j < frozen}
	}
	return func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, s, cols, 20, func(row, col int) {
				if built != nil {
					built[col]++
				}
				coreText(c, "r"+strconv.Itoa(row)+"c"+strconv.Itoa(col)).SingleLine()
			}).Grow(1)
		})
	}
}

// builtColumns returns the columns whose cells were built, in order.
func builtColumns(built map[int]int) []int {
	var cols []int
	for j := range built {
		cols = append(cols, j)
	}
	slices.Sort(cols)
	return cols
}

func TestTableVirtualColumnsBuildsThoseInView(t *testing.T) {
	for _, virtual := range []bool{false, true} {
		built := map[int]int{}
		s := ListState{VirtualColumns: virtual}
		tt := coreNewTester(wideTable(&s, 100, 100, 0, built), 400, 300)
		tt.Frame()
		clear(built)
		tt.Frame()
		cols := builtColumns(built)
		switch {
		case !virtual && len(cols) != 100:
			t.Errorf("off: built %d columns, want all", len(cols))
		case virtual && (cols[0] != 0 || cols[len(cols)-1] > 8 || len(cols) < 4):
			t.Errorf("on, at the start: built columns %v", cols)
		}
	}
}

// The columns in view are those built as the table scrolls, and the
// headers and the cells stay as aligned as with every column built.
func TestTableVirtualColumnsScrolls(t *testing.T) {
	built := map[int]int{}
	s := ListState{VirtualColumns: true}
	tt := coreNewTester(wideTable(&s, 100, 100, 0, built), 400, 300)
	tt.Frame()
	refS := ListState{}
	ref := coreNewTester(wideTable(&refS, 100, 100, 0, nil), 400, 300)
	for _, dx := range []float32{3000, 300, -1000, 6000, 20000, -20000} {
		tt.Scroll(200, 150, dx, 0)
		ref.Scroll(200, 150, dx, 0)
		tt.Frame()
		ref.Frame()
		if scrollXOf(&s) != scrollXOf(&refS) {
			t.Fatalf("scrolled by %v: at %v, want %v", dx, scrollXOf(&s), scrollXOf(&refS))
		}
		x := int(scrollXOf(&s) / 100)
		for _, j := range []int{x, x + 1, x + 2} {
			name := "r2c" + strconv.Itoa(j)
			got, ok := tt.Find(name)
			want, wok := ref.Find(name)
			if !ok || !wok || got != want {
				t.Fatalf("scrolled by %v: %s at %v (%v), want %v (%v)", dx, name, got, ok, want, wok)
			}
			head := header(t, tt, "H"+strconv.Itoa(j))
			if float64(got.X) != head.X+10 {
				t.Errorf("scrolled by %v: %s at %v, its header at %v", dx, name, got.X, head.X)
			}
		}
	}
	// Not all of them: the columns far from view are not built.
	clear(built)
	tt.Scroll(200, 150, -1e6, 0)
	tt.Frame()
	if cols := builtColumns(built); len(cols) > 9 {
		t.Errorf("at the start again: built %v", cols)
	}
}

// A column sharing the room left cannot be placed before the table is
// laid out: every column is built then, and with right to left text.
func TestTableVirtualColumnsNeedsWidths(t *testing.T) {
	built := map[int]int{}
	s := ListState{VirtualColumns: true}
	cols := []TableColumn{{Title: "A"}, {Title: "B", Width: 300}, {Title: "C", Width: 300}, {Title: "D", Width: 300}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, cols, 3, func(row, col int) {
			built[col]++
			coreText(c, "x").SingleLine()
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Frame()
	if cols := builtColumns(built); len(cols) != 4 {
		t.Errorf("with a flexible column: built %v", cols)
	}
}

// The header cannot be dragged away from the columns not built: it is
// built whole, so that moving a column and the accessibility tree see them
// all.
func TestTableVirtualColumnsBuildsAllHeaders(t *testing.T) {
	s := ListState{VirtualColumns: true}
	tt := coreNewTester(wideTable(&s, 100, 100, 0, nil), 400, 300)
	tt.Frame()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, j := range []int{0, 50, 99} {
		accessNode(t, tt.h.access, platform.RoleColumnHeader, "H"+strconv.Itoa(j))
	}
}

// Frozen columns stay at the table's start as the others scroll under
// them: their headers and their cells keep their places.
func TestTableFrozenColumns(t *testing.T) {
	for _, virtual := range []bool{false, true} {
		s := ListState{VirtualColumns: virtual}
		tt := coreNewTester(wideTable(&s, 40, 100, 2, nil), 500, 300)
		tt.Frame()
		h0, h1 := header(t, tt, "H0"), header(t, tt, "H1")
		c0, _ := tt.Find("r3c0")
		c1, _ := tt.Find("r3c1")
		tt.Scroll(250, 150, 1050, 0)
		tt.Frame()
		if got := header(t, tt, "H0"); got != h0 {
			t.Errorf("virtual %v: H0 moved from %v to %v", virtual, h0, got)
		}
		if got := header(t, tt, "H1"); got != h1 {
			t.Errorf("virtual %v: H1 moved from %v to %v", virtual, h1, got)
		}
		if got, _ := tt.Find("r3c0"); got != c0 {
			t.Errorf("virtual %v: r3c0 moved from %v to %v", virtual, c0, got)
		}
		if got, _ := tt.Find("r3c1"); got != c1 {
			t.Errorf("virtual %v: r3c1 moved from %v to %v", virtual, c1, got)
		}
		if scrollXOf(&s) != 1050 {
			t.Fatalf("virtual %v: scrolled to %v", virtual, scrollXOf(&s))
		}
		// The others went by as far as the table scrolled, header and cell.
		head := header(t, tt, "H12")
		if cell, ok := tt.Find("r3c12"); !ok || float64(cell.X) != head.X+10 || head.X != h0.X-1050+1200 {
			t.Errorf("virtual %v: H12 at %v, r3c12 at %v (%v)", virtual, head.X, cell.X, ok)
		}
	}
}

// What the pinned columns cover cannot be clicked: the click lands in the
// pinned column, where it shows, not in the one scrolled under it.
func TestTableFrozenHitTest(t *testing.T) {
	s := &ListState{CellSelect: true, VirtualColumns: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Frozen: true}, {Title: "A", Width: 100, Frozen: true}}
	for k := range 30 {
		cols = append(cols, TableColumn{Title: string(rune('a'+k%26)) + strconv.Itoa(k), Width: 100})
	}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 10, func(row, col int) {
			Textf(c, "r%dc%d", row, col)
		}).Grow(1)
	}, 500, 300)
	tt.Frame()
	tt.Scroll(250, 150, 430, 0)
	tt.Frame()
	// Column 5 (x 540 of the row) is at 540-430 = 110: under A (40-140).
	x, y := rowCenter(t, tt, "r2c1")
	tt.ClickAt(x+60, y) // by the right of A's text, where column 3 is under it
	wantBlock(t, s, sel, "click over the pinned A", []int{2}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
	// A column in view at the right (column 8, at 740-430) takes its click.
	x, y = rowCenter(t, tt, "r2c8")
	tt.ClickAt(x, y)
	wantBlock(t, s, sel, "click on r2c8", []int{2}, ColumnRange{Cursor: "g6", Anchor: "g6"}, []string{"g6"})
}

// Keys move the cursor to a column not built, which scrolls into view past
// the pinned ones.
func TestTableFrozenRevealsCursor(t *testing.T) {
	s := &ListState{CellSelect: true, VirtualColumns: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "A", Width: 100, Frozen: true}}
	for k := range 40 {
		cols = append(cols, TableColumn{Title: "C" + strconv.Itoa(k), Width: 100})
	}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 5, func(row, col int) {
			Textf(c, "r%dc%d", row, col)
		}).Grow(1)
	}, 500, 300)
	tt.Frame()
	tt.Click("r0c0")
	for range 20 {
		tt.Key(0, KeyRight)
		tt.Frame()
	}
	if s.Cells.Cursor != "C19" {
		t.Fatalf("the cursor is in %q", s.Cells.Cursor)
	}
	cell, ok := tt.Find("r0c20")
	pinned, _ := tt.Find("r0c0")
	if !ok || cell.X < pinned.X+100 || cell.X > 500 {
		t.Errorf("the cursor's cell at %v (%v), under the pinned one at %v or out of the view", cell.X, ok, pinned.X)
	}
}

// Frozen columns move like any other: the pinned run is the leading
// Frozen columns of the order the layout gives, and only a run from the
// first is pinned. Fixed ones stay where they are.
func TestTableFrozenArrangeAndGeometry(t *testing.T) {
	cols := []TableColumn{{Title: "A", Width: 50, Frozen: true}, {Title: "B", Width: 60, Frozen: true}, {Title: "C", Width: 70}, {Title: "D", Width: 80, Frozen: true}, {Title: "E", Width: 90}}
	layout := TableLayout{}
	order := layout.arrange(cols)
	if !slices.Equal(order, []int{0, 1, 2, 3, 4}) {
		t.Errorf("arrange: %v", order)
	}
	starts, n, w := tableGeometry(&layout, cols, order)
	if n != 2 || w != 110 || len(starts) != 6 || starts[2] != 110 || starts[5] != 350 {
		t.Errorf("geometry: starts %v, %d frozen, %v wide", starts, n, w)
	}
	// Moved: D takes the first place, then B, A; C and E follow.
	layout = TableLayout{Order: []string{"D", "B", "A", "E", "C"}}
	order = layout.arrange(cols)
	if !slices.Equal(order, []int{3, 1, 0, 4, 2}) {
		t.Errorf("arrange after the move: %v", order)
	}
	starts, n, w = tableGeometry(&layout, cols, order)
	if n != 3 || w != 190 || starts[3] != 190 {
		t.Errorf("pinned run after the move: starts %v, %d frozen, %v wide", starts, n, w)
	}
	// An unfrozen column first pins nothing, and a Frozen one after it
	// scrolls.
	layout = TableLayout{Order: []string{"C", "A", "B", "D", "E"}}
	order = layout.arrange(cols)
	if _, n, w = tableGeometry(&layout, cols, order); n != 0 || w != 0 {
		t.Errorf("an unfrozen first column: %d frozen, %v wide", n, w)
	}
	// A Fixed column stays where it is, Frozen or not.
	fixed := []TableColumn{{Title: "A", Width: 50, Fixed: true, Frozen: true}, {Title: "B", Width: 60, Frozen: true}, {Title: "C", Width: 70}}
	layout = TableLayout{Order: []string{"C", "B", "A"}}
	if order = layout.arrange(fixed); !slices.Equal(order, []int{0, 2, 1}) {
		t.Errorf("a fixed column moved: %v", order)
	}
	// One sharing the room left cannot pin, nor can a column after it.
	flex := []TableColumn{{Title: "A", Frozen: true}, {Title: "B", Width: 60, Frozen: true}}
	if starts, n, _ := tableGeometry(&TableLayout{}, flex, []int{0, 1}); n != 0 || starts != nil {
		t.Errorf("a frozen column sharing the room left: starts %v, %d frozen", starts, n)
	}
}

// A Frozen column is dragged to another place and resized like any other,
// and the pinned run follows the order after the move.
func TestTableFrozenMovesAndResizes(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name", Width: 100, Frozen: true}, {Title: "Kind", Width: 100, Frozen: true}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	name := header(t, tt, "Name")
	y := float32(name.Y + name.H/2)
	// Name dragged past the middle of Kind.
	x := float32(name.X + 20)
	tt.Press(x, y)
	for _, d := range []float32{5, 20, 50, 60} {
		tt.Move(x+d, y)
	}
	tt.Release(x+60, y)
	if got := s.Columns.Order; len(got) != 3 || got[0] != "Kind" || got[1] != "Name" || got[2] != "Size" {
		t.Fatalf("the order after moving a frozen column: %q", got)
	}
	order := s.Columns.arrange(cols)
	if _, n, w := tableGeometry(&s.Columns, cols, order); n != 2 || w != 200 {
		t.Errorf("pinned run after the move: %d frozen, %v wide", n, w)
	}
	// Moved past the unfrozen Size, it leaves the pinned run.
	name = header(t, tt, "Name")
	x = float32(name.X + 20)
	tt.Press(x, y)
	for _, d := range []float32{5, 20, 50, 60} {
		tt.Move(x+d, y)
	}
	tt.Release(x+60, y)
	if got := s.Columns.Order; got[2] != "Name" {
		t.Fatalf("the order after the second move: %q", got)
	}
	order = s.Columns.arrange(cols)
	if _, n, w := tableGeometry(&s.Columns, cols, order); n != 1 || w != 100 {
		t.Errorf("pinned run once Name left it: %d frozen, %v wide", n, w)
	}
	// Kind, still pinned, resizes from the edge of its header.
	kind := header(t, tt, "Kind")
	tt.Press(float32(kind.X+kind.W-1), y)
	tt.Move(float32(kind.X+kind.W+20), y)
	tt.Release(float32(kind.X+kind.W+40), y)
	if w := header(t, tt, "Kind").W; w <= 100 {
		t.Errorf("a frozen column stayed %v wide", w)
	}
}

// Tables that ask for neither are as they were: every column built, none
// pinned.
func TestTableWithoutVirtualOrFrozenIsUnchanged(t *testing.T) {
	built := map[int]int{}
	s := ListState{}
	tt := coreNewTester(wideTable(&s, 30, 100, 0, built), 400, 300)
	tt.Frame()
	if n := len(builtColumns(built)); n != 30 {
		t.Errorf("built %d of 30 columns", n)
	}
	tt.Scroll(200, 150, 500, 0)
	tt.Frame()
	// The fork's accessBounds is what of the box shows, empty once it is
	// scrolled out of view.
	if h := header(t, tt, "H0"); h.W > 0 {
		t.Errorf("H0 at %v, %v wide after scrolling 500: it should have scrolled away", h.X, h.W)
	}
}

// scrollXOf returns how far the table s shows is scrolled sideways.
func scrollXOf(s *ListState) float64 { return s.frame.e.st.scrollX }

// A pinned cell takes the background of the nearest element around it
// that has one, and one that is translucent, as a chosen or hovered row's,
// is composited over the ones behind it, down to an opaque one: taken as
// it is, it would let the cells scrolling under the pinned one show through.
func TestFrozenStyleIsOpaque(t *testing.T) {
	th := LightTheme()
	page := RGB(250, 250, 250)
	tint := RGBA(0, 0, 255, 0.2)
	chain := func(cell Color, around ...Color) *node {
		var parent *node
		for i := len(around) - 1; i >= 0; i-- {
			parent = &node{parent: parent, bg: around[i]}
		}
		return &node{parent: parent, bg: cell}
	}
	for _, tc := range []struct {
		name string
		e    *node
		want Color
	}{
		{"an opaque row", chain(Transparent, page), page},
		{"a translucent row over the page", chain(Transparent, tint, page), tint.Over(page)},
		{"a translucent row, the page's none", chain(Transparent, tint), tint.Over(th.Background)},
		{"nothing around it", chain(Transparent), th.Background},
		{"a translucent cell in a translucent row", chain(tint, tint, page), tint.Over(tint.Over(page))},
		{"a cell of its own", chain(RGB(1, 2, 3), page), RGB(1, 2, 3)},
	} {
		frozenStyle(th, false)(tc.e)
		if tc.e.bg != tc.want || tc.e.bg.A != 255 {
			t.Errorf("%s: pinned cell is %+v, want the opaque %+v", tc.name, tc.e.bg, tc.want)
		}
	}
}

// Right to left, the window follows the scroll as it does left to right:
// the columns from the start, at the right, are built, and the others as it
// scrolls, counted from the start.
func TestTableVirtualColumnsRightToLeft(t *testing.T) {
	built := map[int]int{}
	s := ListState{VirtualColumns: true}
	tt := coreNewTester(wideTable(&s, 100, 100, 0, built), 400, 300)
	tt.SetDirection(RightToLeft)
	tt.Frame()
	clear(built)
	tt.Frame()
	if cols := builtColumns(built); cols[0] != 0 || cols[len(cols)-1] > 8 || len(cols) < 4 {
		t.Errorf("at the start: built columns %v", cols)
	}
	tt.Scroll(200, 150, -3000, 0) // right to left, the start is on the right

	tt.Frame()
	clear(built)
	tt.Frame()
	cols := builtColumns(built)
	if cols[0] < 20 || cols[len(cols)-1] > 40 || len(cols) > 12 {
		t.Errorf("scrolled to 3000: built columns %v", cols)
	}
	// One at that place shows.
	if _, ok := tt.Find("r1c" + strconv.Itoa(cols[len(cols)/2])); !ok {
		t.Errorf("a column in the window is not shown: texts %q", tt.Texts())
	}
}

// Right to left, the pinned columns are at the right, and revealing a
// column scrolls the table as little as shows it in what they leave.
func TestCellSelectRevealsColumnsRightToLeftFrozen(t *testing.T) {
	s := ListState{CellSelect: true, Selection: &Selection[int]{}}
	cols := make([]TableColumn, 20)
	for j := range cols {
		cols[j] = TableColumn{Title: string(rune('A' + j)), Width: 100, Frozen: j < 2}
	}
	tt := NewTester(func(c *Context) {
		Table(c, &s, cols, 5, func(row, col int) {
			Textf(c, "r%dc%d", row, col)
		}).Grow(1)
	}, 500, 300)
	tt.SetDirection(RightToLeft)
	tt.Frame()
	tt.Click("r1c0")
	for range 12 {
		tt.Key(0, KeyLeft)
	}
	h, list := tt.rt.states[s.tableCols[s.colIndex(s.Cells.Cursor)].head], s.frame.e.st
	// What the pinned two leave, at the right, is 300 of the 500.
	if s.Cells.Cursor != "M" || h.x < list.x || h.x+h.w > list.x+list.w-s.frozenW {
		t.Fatalf("Left 12 times from A right to left: cursor %q, its header at %v..%v, the view %v..%v less %v pinned",
			s.Cells.Cursor, h.x, h.x+h.w, list.x, list.x+list.w, s.frozenW)
	}
	// Back at the first column that scrolls, next to the pinned ones: it is
	// not left under them.
	for range 10 {
		tt.Key(0, KeyRight)
	}
	h = tt.rt.states[s.tableCols[s.colIndex(s.Cells.Cursor)].head]
	if s.Cells.Cursor != "C" || h.x < list.x || h.x+h.w > list.x+list.w-s.frozenW {
		t.Fatalf("Right 10 times: cursor %q, its header at %v..%v, the view %v..%v less %v pinned",
			s.Cells.Cursor, h.x, h.x+h.w, list.x, list.x+list.w, s.frozenW)
	}
}

// A pinned header follows the pointer as the others do, scrolled or not:
// Left would replace the inset that holds it in place.
func TestTableFrozenHeaderFollowsThePointer(t *testing.T) {
	for _, scrolled := range []bool{false, true} {
		s := ListState{}
		cols := []TableColumn{{Title: "A", Width: 100, Frozen: true}, {Title: "B", Width: 100, Frozen: true}, {Title: "C", Width: 100}, {Title: "D", Width: 300}, {Title: "E", Width: 300}}
		tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
		if scrolled {
			tt.Scroll(300, 150, 150, 0)
			tt.Frame()
		}
		b := header(t, tt, "B")
		y, x := float32(b.Y+b.H/2), float32(b.X+20)
		tt.Press(x, y)
		for _, d := range []float32{5, 20, 40} {
			tt.Move(x+d, y)
		}
		if got := header(t, tt, "B"); got.X != b.X+40 {
			t.Errorf("scrolled %v: B at %v after a 40 DIP drag from %v, want %v", scrolled, got.X, b.X, b.X+40)
		}
		tt.Release(x+40, y)
	}
}

// A List keeps the row holding the focus built as it scrolls out of view,
// and VirtualColumns the column: a cell holding the keyboard keeps it. It
// lets the column go once the keyboard is elsewhere.
func TestTableVirtualColumnsKeepTheFocusedColumn(t *testing.T) {
	built := map[int]int{}
	s := ListState{VirtualColumns: true}
	cols := make([]TableColumn, 60)
	for j := range cols {
		cols[j] = TableColumn{Title: "H" + strconv.Itoa(j), Width: 100}
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, &s, cols, 5, func(row, col int) {
				built[col]++
				if row == 1 && col == 1 {
					coreBox(c).Key("editor").Focusable().Children(func() { coreText(c, "editor") })
					return
				}
				coreText(c, "r"+strconv.Itoa(row)+"c"+strconv.Itoa(col)).SingleLine()
			}).Grow(1)
		})
	}, 400, 300)
	tt.Frame()
	tt.Click("editor")
	tt.Frame()
	tt.Scroll(200, 150, 3000, 0)
	tt.Frame()
	clear(built)
	tt.Frame()
	if _, ok := built[1]; !ok || !tt.HasText("editor") {
		t.Fatalf("the column holding the focus, scrolled out of view, is gone: built %v", builtColumns(built))
	}
	// The keyboard goes to a cell in view, and the column goes with it.
	tt.Click("r1c30")
	tt.Frame()
	clear(built)
	tt.Frame()
	if _, ok := built[1]; ok {
		t.Errorf("the column is kept built once the keyboard left it: built %v", builtColumns(built))
	}
}

// The Cursor's column is kept too, for the keys to move from its cell.
func TestTableVirtualColumnsKeepTheCursorColumn(t *testing.T) {
	tt, s, _ := cellTable(t, 8, 30, 100)
	s.VirtualColumns = true
	tt.Frame()
	tt.Click("r1c2")
	tt.Frame()
	tt.Scroll(200, 150, 1500, 0)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("r1c18"); !ok {
		t.Fatalf("the table did not scroll: texts %q", tt.Texts())
	}
	if _, ok := tt.Find("r1c2"); !ok {
		t.Fatalf("the cell of the Cursor %q, scrolled out of view, is not built", s.Cells.Cursor)
	}
}

// A pinned header under the pointer is opaque with a theme whose hover
// color is translucent, as it is without the pointer: the headers scrolling
// under it would show through.
func TestTableFrozenHeaderHoveredIsOpaque(t *testing.T) {
	sort := SortOrder{}
	s := ListState{Sort: &sort}
	cols := []TableColumn{{Title: "A", Width: 100, Frozen: true, Sortable: true}, {Title: "B", Width: 400}}
	// The node of the frame built last, whose look the paint settled.
	var head *node
	s.HeaderCell = func(e Element, j int) {
		if j == 0 {
			head = e.node()
		}
	}
	th := *LightTheme()
	th.SurfaceHover = RGBA(0, 0, 0, 0.1)
	tt := coreNewTester(func(c *context) {
		c.SetTheme(&th)
		coreColumn(c).Fill().Children(func() {
			coreTable(c, &s, cols, 3, func(row, col int) { coreText(c, "x") }).Grow(1)
		})
	}, 300, 200)
	tt.Frame()
	a := header(t, tt, "A")
	tt.Move(float32(a.X+a.W/2), float32(a.Y+a.H/2))
	tt.Frame()
	tt.Frame()
	if head == nil || head.styleFn == nil {
		t.Fatal("no pinned header built")
	}
	// Painting restores what its style sets: run it as the paint does.
	head.styleFn(head)
	if !head.Hovered() || head.bg.A != 255 {
		t.Errorf("the hovered pinned header's background %+v (hovered %v), want an opaque one", head.bg, head.Hovered())
	}
}

// The first pinned cell of a row with rounded corners (RoundRows) takes the
// corners at the row's start, which its square background would cover; the
// others, and the cells of a header row, are as they were.
func TestFrozenStyleTakesTheRowsStartCorners(t *testing.T) {
	th := LightTheme()
	for _, rtl := range []bool{false, true} {
		row := &node{radius: [4]float32{6, 7, 8, 9}}
		if rtl {
			row.dir = dirRTL
		}
		first, second := &node{parent: row, dir: row.dir}, &node{parent: row, dir: row.dir}
		frozenStyle(th, true)(first)
		frozenStyle(th, false)(second)
		want := [4]float32{6, 0, 0, 9}
		if rtl {
			want = [4]float32{0, 7, 8, 0}
		}
		if first.radius != want {
			t.Errorf("rtl %v: the first pinned cell's corners %v, want %v", rtl, first.radius, want)
		}
		if second.radius != [4]float32{} {
			t.Errorf("rtl %v: a later pinned cell's corners %v, want none", rtl, second.radius)
		}
	}
}

// A table without VirtualColumns does not read which cell holds the
// keyboard: each read asks for another frame whenever the focus moves.
func TestTableReadsTheFocusOnlyForVirtualColumns(t *testing.T) {
	for _, virtual := range []bool{false, true} {
		s := ListState{VirtualColumns: virtual}
		cols := []TableColumn{{Title: "A", Width: 100}, {Title: "B", Width: 100}}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				coreTable(c, &s, cols, 3, func(row, col int) {
					coreText(c, "r"+strconv.Itoa(row)+"c"+strconv.Itoa(col))
				}).Grow(1)
			})
		}, 400, 300)
		tt.Frame()
		tt.Frame()
		if got := tt.rt.withinRead; got != virtual {
			t.Errorf("VirtualColumns %v: the frame read FocusWithin %v, want %v", virtual, got, virtual)
		}
	}
}

// gapIDs returns the IDs of the blocks standing for the columns not built,
// in the order of the tree.
func gapIDs(root *node) []uint64 {
	var ids []uint64
	var walk func(n *node)
	walk = func(n *node) {
		if k, ok := n.key.(string); ok && strings.HasPrefix(k, "\x00gap") {
			ids = append(ids, n.id)
		}
		for ch := n.first; ch != nil; ch = ch.next {
			walk(ch)
		}
	}
	walk(root)
	return ids
}

// The block after the columns built stays the same element as the table
// scrolls, as the one before: keyed by where the run it stands for ends it
// would be another as the window moves, and lose its state.
func TestTableVirtualColumnsGapsKeepTheirKeys(t *testing.T) {
	s := ListState{VirtualColumns: true}
	cols := make([]TableColumn, 60)
	for j := range cols {
		cols[j] = TableColumn{Title: "H" + strconv.Itoa(j), Width: 100}
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, &s, cols, 1, func(row, col int) { coreText(c, "r"+strconv.Itoa(col)).SingleLine() }).Grow(1)
		})
	}, 400, 300)
	tt.Frame()
	tt.Scroll(200, 150, 1500, 0)
	tt.Frame()
	tt.Frame()
	before := gapIDs(tt.rt.c.root)
	tt.Scroll(200, 150, 200, 0)
	tt.Frame()
	tt.Frame()
	after := gapIDs(tt.rt.c.root)
	if len(before) != 2 || !slices.Equal(before, after) {
		t.Errorf("the blocks' IDs changed as the table scrolled: %v, then %v", before, after)
	}
}

// A column with no title or ID, a checkbox column say, is kept built while
// it holds the keyboard as the others are.
func TestTableVirtualColumnsKeepTheFocusedUntitledColumn(t *testing.T) {
	s := ListState{VirtualColumns: true}
	cols := make([]TableColumn, 60)
	for j := range cols {
		cols[j] = TableColumn{Title: "H" + strconv.Itoa(j), Width: 100}
	}
	cols[1] = TableColumn{Width: 100}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTable(c, &s, cols, 5, func(row, col int) {
				if row == 1 && col == 1 {
					coreBox(c).Key("check").Focusable().Children(func() { coreText(c, "check") })
					return
				}
				coreText(c, "r"+strconv.Itoa(row)+"c"+strconv.Itoa(col)).SingleLine()
			}).Grow(1)
		})
	}, 400, 300)
	tt.Frame()
	tt.Click("check")
	tt.Frame()
	tt.Scroll(200, 150, 3000, 0)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("check") {
		t.Fatal("the untitled column holding the focus, scrolled out of view, is gone")
	}
}

// The column a drag over the cells started in is kept built, for the drag
// to go on from it as the table scrolls away.
func TestTableVirtualColumnsKeepTheDragColumn(t *testing.T) {
	tt, s, _ := cellTable(t, 8, 30, 100)
	s.VirtualColumns = true
	tt.Frame()
	x, y := rowCenter(t, tt, "r1c2")
	x2, _ := rowCenter(t, tt, "r1c3")
	tt.Press(x, y)
	tt.Move(x+3, y)
	// The pointer moves the Cursor on, to the column under it.
	tt.Move(x2+60, y)
	tt.Frame()
	if !s.dragSel.on || s.dragSel.fromCol == s.Cells.Cursor {
		t.Fatalf("the drag is not on, or the Cursor did not leave its column: on %v, from %q, Cursor %q", s.dragSel.on, s.dragSel.fromCol, s.Cells.Cursor)
	}
	tt.Scroll(200, 150, 1500, 0)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("r1c18"); !ok {
		t.Fatalf("the table did not scroll: texts %q", tt.Texts())
	}
	if _, ok := tt.Find("r1c2"); !ok {
		t.Fatalf("the column the drag started in (%q), scrolled out of view, is not built", s.dragSel.fromCol)
	}
}

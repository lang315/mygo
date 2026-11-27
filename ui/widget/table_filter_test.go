package widget

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

type fdoc struct {
	id         int
	name, kind string
	size       int
}

func fdocCols() []Column[fdoc] {
	return []Column[fdoc]{
		{Title: "Name", Text: func(d *fdoc) string { return d.name }, Width: 160, Sortable: true, Filterable: true},
		{Title: "Kind", Text: func(d *fdoc) string { return d.kind }, Width: 120, Sortable: true, Filterable: true},
		{Title: "Size", Text: func(d *fdoc) string { return fmt.Sprint(d.size) }, Width: 80},
	}
}

func fdocs() []fdoc {
	return []fdoc{{1, "ann", "Doc", 10}, {2, "Hannah", "Sheet", 20}, {3, "bob", "Doc", 30}, {4, "ANDY", "Image", 40}}
}

func fdocGrid(rows ...fdoc) *DataGridViewWidget[fdoc, int] {
	if rows == nil {
		rows = fdocs()
	}
	return DataGridView(func(d *fdoc) int { return d.id }, fdocCols()...).SetRows(rows).Expand()
}

// fshows fails unless the names shown, in any order, are want.
func fshows(t *testing.T, tt *ui.Tester, want ...string) {
	t.Helper()
	tt.Frame()
	texts := tt.Texts()
	for _, n := range []string{"ann", "Hannah", "bob", "ANDY", "cid", "dan"} {
		if slices.Contains(texts, n) != slices.Contains(want, n) {
			t.Fatalf("want %q shown, got %q", want, tt.Texts())
		}
	}
}

func TestColumnFilterContains(t *testing.T) {
	g := fdocGrid()
	tt := tester(t, g)
	g.SetColumnFilter("Name", ColumnFilter{Contains: "an"})
	fshows(t, tt, "ann", "Hannah", "ANDY")
	if f := g.ColumnFilter("Name"); f.Contains != "an" || f.Values != nil {
		t.Fatalf("ColumnFilter(Name) = %+v", f)
	}
	// A column of its own Filter uses it for Contains, as the search does.
	cols := fdocCols()
	cols[0].Filter = func(d *fdoc, kw string) bool { return d.name == kw }
	g2 := DataGridView(func(d *fdoc) int { return d.id }, cols...).SetRows(fdocs()).Expand()
	tt2 := tester(t, g2)
	g2.SetColumnFilter("Name", ColumnFilter{Contains: "ann"})
	fshows(t, tt2, "ann")
}

func TestColumnFilterValues(t *testing.T) {
	g := fdocGrid()
	tt := tester(t, g)
	vals := []string{"Doc", "Sheet"}
	g.SetColumnFilter("Kind", ColumnFilter{Values: vals})
	fshows(t, tt, "ann", "Hannah", "bob")
	vals[0] = "Image" // the grid keeps a copy
	got := g.ColumnFilter("Kind")
	if !slices.Equal(got.Values, []string{"Doc", "Sheet"}) {
		t.Fatalf("ColumnFilter(Kind) = %+v after the caller changed its slice", got)
	}
	got.Values[0] = "Image" // and returns one
	fshows(t, tt, "ann", "Hannah", "bob")
	if f := g.ColumnFilter("Kind"); f.Values[0] != "Doc" {
		t.Fatalf("ColumnFilter returned the grid's slice: %+v", f)
	}
	// An empty, non-nil Values keeps none, and stays non-nil.
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{}})
	fshows(t, tt)
	if f := g.ColumnFilter("Kind"); f.Values == nil {
		t.Fatal("an empty Values came back nil")
	}
	g.SetColumnFilter("Kind", ColumnFilter{})
	fshows(t, tt, "ann", "Hannah", "bob", "ANDY")
}

func TestColumnFilterAndSearch(t *testing.T) {
	g := fdocGrid().BottomPanel()
	tt := tester(t, g)
	g.Search("Name", "a")
	fshows(t, tt, "ann", "Hannah", "ANDY")
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc", "Image"}})
	fshows(t, tt, "ann", "ANDY")
	g.SetColumnFilter("Name", ColumnFilter{Contains: "N"})
	fshows(t, tt, "ann", "ANDY")
	g.Search("Name", "")
	fshows(t, tt, "ann", "ANDY")
	g.SetColumnFilter("Name", ColumnFilter{})
	fshows(t, tt, "ann", "bob", "ANDY")
}

// A title no column has is ignored; a column that is not Filterable can be
// filtered by the program (the spec ignores unknown titles alone).
func TestColumnFilterUnknownTitleIgnored(t *testing.T) {
	g := fdocGrid()
	tt := tester(t, g)
	g.SetColumnFilter("Nope", ColumnFilter{Contains: "zz"})
	fshows(t, tt, "ann", "Hannah", "bob", "ANDY")
	if f := g.ColumnFilter("Nope"); !f.isZero() {
		t.Fatalf("ColumnFilter(Nope) = %+v", f)
	}
	mu.Lock()
	filter := g.filter
	mu.Unlock()
	if filter != nil {
		t.Fatal("an ignored filter left the view a filter")
	}
	g.SetColumnFilter("Size", ColumnFilter{Values: []string{"30"}})
	fshows(t, tt, "bob")
}

func TestClearFiltersKeepsSearch(t *testing.T) {
	g := fdocGrid().BottomPanel()
	tt := tester(t, g)
	g.Search("Name", "an")
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	fshows(t, tt, "ann")
	g.ClearFilters()
	fshows(t, tt, "ann", "Hannah", "ANDY")
	if f := g.ColumnFilter("Kind"); !f.isZero() {
		t.Fatalf("ColumnFilter(Kind) = %+v after ClearFilters", f)
	}
	if v := g.keyword.Text(); v != "an" {
		t.Fatalf("the panel's keyword is %q after ClearFilters", v)
	}
}

// boxesY returns the check boxes shown, from the top down.
func boxesY(tt *Tester) []string {
	var ns []ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleCheckBox {
			ns = append(ns, n)
		}
	}
	slices.SortStableFunc(ns, func(a, b ui.TestNode) int { return int(a.Bounds.Y - b.Bounds.Y) })
	var out []string
	for _, n := range ns {
		out = append(out, n.Label)
	}
	return out
}

func TestFilterDropDownLists(t *testing.T) {
	var fired atomic.Int32
	g := fdocGrid().SetFilterDropDowns(true).OnFilterChange(func() { fired.Add(1) })
	tt := NewTester(t, g, 800, 600)
	if tt.Has(ByRole(ui.RoleButton).Label("Filter Size")) {
		t.Fatal("a column that is not Filterable has a funnel")
	}
	tt.Find(ByRole(ui.RoleButton).Label("Filter Kind")).Click()
	if got := boxesY(tt); !slices.Equal(got, []string{"Doc", "Image", "Sheet"}) {
		t.Fatalf("the drop-down lists %q", got)
	}
	tt.Find(ByRole(ui.RoleCheckBox).Label("Sheet")).Click()
	fshows(t, tt.Tester, "ann", "bob", "ANDY")
	if n := fired.Load(); n != 1 {
		t.Fatalf("OnFilterChange ran %d times for one uncheck", n)
	}
	if f := g.ColumnFilter("Kind"); !slices.Equal(f.Values, []string{"Doc", "Image"}) || f.Contains != "" {
		t.Fatalf("ColumnFilter(Kind) = %+v", f)
	}
	// Contains, then Clear and Select all.
	field := tt.Find(ByRole(ui.RoleTextField).Label("Contains"))
	b := field.Bounds()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Type("o")
	tt.Settle()
	fshows(t, tt.Tester, "ann", "bob")
	tt.Find(ByRole(ui.RoleButton).Label("Clear")).Click()
	fshows(t, tt.Tester, "ann", "Hannah", "bob", "ANDY")
	if f := g.ColumnFilter("Kind"); !f.isZero() {
		t.Fatalf("ColumnFilter(Kind) = %+v after Clear", f)
	}
	tt.Find(ByRole(ui.RoleCheckBox).Label("Doc")).Click()
	fshows(t, tt.Tester, "Hannah", "ANDY")
	tt.Find(ByRole(ui.RoleButton).Label("Select all")).Click()
	fshows(t, tt.Tester, "ann", "Hannah", "bob", "ANDY")
	if f := g.ColumnFilter("Kind"); f.Values != nil {
		t.Fatalf("ColumnFilter(Kind) = %+v after Select all", f)
	}
	if n := fired.Load(); n != 5 {
		t.Fatalf("OnFilterChange ran %d times for five changes", n)
	}
	// Select all again changes nothing, and runs nothing.
	tt.Find(ByRole(ui.RoleButton).Label("Select all")).Click()
	if n := fired.Load(); n != 5 {
		t.Fatalf("a Select all that changed nothing ran OnFilterChange: %d", n)
	}
	// Unchecking a value, then checking it again, leaves every value checked:
	// Values nil, so rows loaded later show.
	tt.Find(ByRole(ui.RoleCheckBox).Label("Image")).Click()
	tt.Find(ByRole(ui.RoleCheckBox).Label("Image")).Click()
	if f := g.ColumnFilter("Kind"); f.Values != nil {
		t.Fatalf("ColumnFilter(Kind) = %+v with every value checked again", f)
	}
}

// With the # column first, each funnel is in its own column's header.
func TestFilterDropDownsWithIndexColumn(t *testing.T) {
	g := fdocGrid().IndexColumn().SetFilterDropDowns(true)
	tt := NewTester(t, g, 800, 600)
	for _, title := range []string{"Name", "Kind"} {
		f := tt.Find(ByRole(ui.RoleButton).Label("Filter " + title)).Bounds()
		h := tt.Find(ByRole(ui.RoleColumnHeader).Label(title)).Bounds()
		if f.X < h.X || f.X+f.W > h.X+h.W {
			t.Fatalf("the funnel of %s at %+v, not in its header at %+v", title, f, h)
		}
	}
}

// The panel's toggle and Clear run OnFilterChange only when they change
// what is filtered.
func TestFilterPanelNoChangeNoEvent(t *testing.T) {
	var fired atomic.Int32
	g := fdocGrid().SetFilterPanel(true).OnFilterChange(func() { fired.Add(1) })
	g.userFilters(true, true) // Clear with no filter
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	g.userFilters(true, false) // on while on
	Update(func() {})
	if n := fired.Load(); n != 0 {
		t.Fatalf("panel actions that changed nothing ran OnFilterChange %d times", n)
	}
	g.userFilters(false, false)
	g.userFilters(true, true)
	if n := fired.Load(); n != 2 {
		t.Fatalf("an off and a Clear ran OnFilterChange %d times", n)
	}
}

func TestFilterDropDownCapsAt1000(t *testing.T) {
	var rows []fdoc
	for i := range 1001 {
		rows = append(rows, fdoc{id: i, name: fmt.Sprintf("n%04d", 1000-i), kind: "Doc"})
	}
	g := fdocGrid(rows...).SetFilterDropDowns(true)
	tt := NewTester(t, g, 800, 600)
	if tt.Has(ByLabel("Only the first 1000 values are listed")) {
		t.Fatal("the cap's note shows before the drop-down opens")
	}
	tt.Find(ByRole(ui.RoleButton).Label("Filter Name")).Click()
	got := boxesY(tt)
	if len(got) != 1000 || got[0] != "n0000" || got[999] != "n0999" || slices.Contains(got, "n1000") {
		t.Fatalf("%d values listed, from %q", len(got), got[:3])
	}
	if !tt.Has(ByLabel("Only the first 1000 values are listed")) {
		t.Fatal("the drop-down does not say it cut the list")
	}
	// 1000 distinct values exactly are not cut.
	g2 := fdocGrid(rows[:1000]...).SetFilterDropDowns(true)
	tt2 := NewTester(t, g2, 800, 600)
	tt2.Find(ByRole(ui.RoleButton).Label("Filter Name")).Click()
	if got := boxesY(tt2); len(got) != 1000 || tt2.Has(ByLabel("Only the first 1000 values are listed")) {
		t.Fatalf("1000 values: %d listed, note %v", len(got), tt2.Has(ByLabel("Only the first 1000 values are listed")))
	}
}

func TestFilterFunnelFilledWhenFiltered(t *testing.T) {
	g := fdocGrid().SetFilterDropDowns(true)
	tt := NewTester(t, g, 800, 600)
	kind := tt.Find(ByRole(ui.RoleButton).Label("Filter Kind"))
	if kind.Checked() {
		t.Fatal("the funnel is filled with no filter")
	}
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	if !kind.Checked() || tt.Find(ByRole(ui.RoleButton).Label("Filter Name")).Checked() {
		t.Fatal("the funnel of the filtered column is not filled, or another's is")
	}
	g.SetColumnFilter("Kind", ColumnFilter{Contains: "o"})
	if !kind.Checked() {
		t.Fatal("a Contains filter leaves the funnel empty")
	}
	g.ClearFilters()
	if kind.Checked() {
		t.Fatal("the funnel stays filled after ClearFilters")
	}
}

// fclickNode clicks the middle of n with the pointer.
func fclickNode(tt *Tester, n *Node) {
	b := n.Bounds()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
}

func TestAutoFilterRowTypes(t *testing.T) {
	var fired atomic.Int32
	g := fdocGrid().SetAutoFilterRow(true).OnFilterChange(func() { fired.Add(1) })
	tt := NewTester(t, g, 800, 600)
	if tt.Has(ByRole(ui.RoleTextField).Label("Filter Size")) {
		t.Fatal("a column that is not Filterable has a field")
	}
	name := tt.Find(ByRole(ui.RoleTextField).Label("Filter Name"))
	title := tt.Find(ByRole(ui.RoleColumnHeader).Label("Name")).Bounds()
	if b := name.Bounds(); b.Y < title.Y+title.H/2-1 || b.Y+b.H > title.Y+title.H {
		t.Fatalf("the field at %+v is not under the title in its header %+v", b, title)
	}
	fclickNode(tt, name)
	tt.Type("a")
	tt.Settle()
	tt.Type("n")
	tt.Settle()
	fshows(t, tt.Tester, "ann", "Hannah", "ANDY")
	if f := g.ColumnFilter("Name"); f.Contains != "an" {
		t.Fatalf("ColumnFilter(Name) = %+v", f)
	}
	if n := fired.Load(); n != 2 {
		t.Fatalf("OnFilterChange ran %d times for two keys", n)
	}
	if v := name.Value(); v != "an" {
		t.Fatalf("the field holds %q", v)
	}
	// The program's filter shows in the field.
	g.SetColumnFilter("Name", ColumnFilter{Contains: "bo"})
	tt.Frame()
	if v := name.Value(); v != "bo" {
		t.Fatalf("after SetColumnFilter the field holds %q", v)
	}
	fshows(t, tt.Tester, "bob")
	g.ClearFilters()
	tt.Frame()
	if v := name.Value(); v != "" {
		t.Fatalf("after ClearFilters the field holds %q", v)
	}
	if n := fired.Load(); n != 2 {
		t.Fatalf("the program's filters ran OnFilterChange: %d", n)
	}
}

func TestAutoFilterRowScrollsSideways(t *testing.T) {
	cols := fdocCols()
	cols[0].Frozen, cols[0].Width = true, 150
	cols[1].Width = 400
	cols[2].Width, cols[2].Filterable = 400, true
	g := DataGridView(func(d *fdoc) int { return d.id }, cols...).SetRows(fdocs()).Expand().SetAutoFilterRow(true)
	tt := NewTester(t, g, 500, 300)
	field := func(title string) ui.Rect { return tt.Find(ByRole(ui.RoleTextField).Label("Filter " + title)).Bounds() }
	head := func(title string) ui.Rect { return tt.Find(ByRole(ui.RoleColumnHeader).Label(title)).Bounds() }
	over := func(title string) {
		t.Helper()
		f, h := field(title), head(title)
		if f.X < h.X || f.X > h.X+4 || f.Y < h.Y+h.H/2-1 || f.Y+f.H > h.Y+h.H {
			t.Fatalf("%s's field at %+v, not over its header at %+v", title, f, h)
		}
	}
	over("Name")
	over("Kind")
	name, kind := field("Name"), field("Kind")
	tt.Scroll(300, 200, 300, 0)
	tt.Frame()
	if f := field("Name"); f.X != name.X {
		t.Fatalf("the Frozen column's field moved from %v to %v", name.X, f.X)
	}
	if f := field("Kind"); f.X >= kind.X {
		t.Fatalf("Kind's field did not scroll: %v, then %v", kind.X, f.X)
	}
	over("Size")
	sz := field("Size")
	if h := head("Size"); sz.W < h.W-6 {
		t.Fatalf("Size's field is %v wide, its header %v", sz.W, h.W)
	}
}

func TestFilterPanelText(t *testing.T) {
	g := fdocGrid().SetFilterPanel(true)
	tt := tester(t, g)
	g.SetColumnFilter("Name", ColumnFilter{Contains: "an"})
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc", "Sheet"}})
	tt.Frame()
	if want := `Name contains "an" and Kind is Doc, Sheet`; !tt.HasText(want) {
		t.Fatalf("no %q in %q", want, tt.Texts())
	}
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{}})
	tt.Frame()
	if want := `Name contains "an" and Kind is (none)`; !tt.HasText(want) {
		t.Fatalf("no %q in %q", want, tt.Texts())
	}
}

func TestFilterPanelToggleKeepsFilters(t *testing.T) {
	var fired atomic.Int32
	g := fdocGrid().SetFilterPanel(true).OnFilterChange(func() { fired.Add(1) })
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	tt := NewTester(t, g, 800, 600)
	fshows(t, tt.Tester, "ann", "bob")
	box := tt.Find(ByRole(ui.RoleCheckBox).Label("Kind is Doc"))
	if !box.Checked() {
		t.Fatal("the panel's box is not checked while the filters apply")
	}
	box.Click()
	fshows(t, tt.Tester, "ann", "Hannah", "bob", "ANDY")
	if f := g.ColumnFilter("Kind"); !slices.Equal(f.Values, []string{"Doc"}) || box.Checked() {
		t.Fatalf("turned off: ColumnFilter(Kind) = %+v, box checked %v", f, box.Checked())
	}
	box.Click()
	fshows(t, tt.Tester, "ann", "bob")
	if n := fired.Load(); n != 2 {
		t.Fatalf("OnFilterChange ran %d times for two toggles", n)
	}
	// Off, a new filter turns them on again.
	box.Click()
	fshows(t, tt.Tester, "ann", "Hannah", "bob", "ANDY")
	g.SetColumnFilter("Name", ColumnFilter{Contains: "b"})
	fshows(t, tt.Tester, "bob")
	if !tt.Find(ByRole(ui.RoleCheckBox).Label(`Name contains "b" and Kind is Doc`)).Checked() {
		t.Fatal("the box is not checked after a filter turned them on")
	}
}

func TestFilterPanelHiddenWithoutFilters(t *testing.T) {
	var fired atomic.Int32
	g := fdocGrid().SetFilterPanel(true).OnFilterChange(func() { fired.Add(1) })
	tt := NewTester(t, g, 800, 600)
	if tt.Has(ByRole(ui.RoleCheckBox)) || tt.Has(ByRole(ui.RoleButton).Label("Clear filters")) {
		t.Fatal("the filter panel shows with no filter")
	}
	g.SetColumnFilter("Kind", ColumnFilter{Contains: "o"})
	if !tt.Has(ByRole(ui.RoleButton).Label("Clear filters")) {
		t.Fatal("the filter panel does not show with a filter")
	}
	g.ClearFilters()
	if tt.Has(ByRole(ui.RoleButton).Label("Clear filters")) {
		t.Fatal("the filter panel shows after ClearFilters")
	}
	g.SetColumnFilter("Kind", ColumnFilter{Contains: "o"})
	tt.Find(ByRole(ui.RoleButton).Label("Clear filters")).Click()
	fshows(t, tt.Tester, "ann", "Hannah", "bob", "ANDY")
	if tt.Has(ByRole(ui.RoleButton).Label("Clear filters")) || fired.Load() != 1 {
		t.Fatalf("after the panel's Clear: shown %v, OnFilterChange %d", tt.Has(ByRole(ui.RoleButton).Label("Clear filters")), fired.Load())
	}
	// The bottom panel's search alone shows no strip.
	g2 := fdocGrid().SetFilterPanel(true).BottomPanel()
	g2.Search("Name", "an")
	tt2 := NewTester(t, g2, 800, 600)
	if tt2.Has(ByRole(ui.RoleButton).Label("Clear filters")) {
		t.Fatal("the filter panel shows for the search alone")
	}
}

// The filter panel coming and going keeps the bottom panel's keyword, being
// typed in, its focus and its text.
func TestFilterPanelKeepsTheKeywordsFocus(t *testing.T) {
	g := fdocGrid().BottomPanel().SetFilterPanel(true)
	tt := NewTester(t, g, 800, 600)
	b := tt.Find(ByRole(ui.RoleButton).Label("Search")).Bounds()
	tt.ClickAt(b.X-12-120, b.Y+b.H/2) // the keyword, 240 wide, left of the button
	tt.Type("a")
	tt.Settle()
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	tt.Settle()
	if !tt.Has(ByRole(ui.RoleButton).Label("Clear filters")) {
		t.Fatal("the filter panel does not show")
	}
	tt.Type("b")
	tt.Settle()
	g.ClearFilters()
	tt.Settle()
	tt.Type("c")
	tt.Settle()
	if v := g.keyword.Text(); v != "abc" {
		t.Fatalf("the keyword holds %q after the filter panel came and went", v)
	}
}

func TestFilterUpdateRowMovesInAndOut(t *testing.T) {
	g := fdocGrid().SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}}).Sort("Name", false)
	tt := tester(t, g)
	fshows(t, tt, "ann", "bob")
	g.SetSelected(3) // bob
	g.UpdateRow(3, func(d *fdoc) { d.kind = "Sheet" })
	fshows(t, tt, "ann")
	if got := g.Selected(); len(got) != 0 {
		t.Fatalf("Selected %v with the selected row filtered out", got)
	}
	g.UpdateRow(4, func(d *fdoc) { d.kind = "Doc"; d.name = "Aaron" })
	tt.Frame()
	if !tt.HasText("Aaron") {
		t.Fatalf("the row changed into the filter does not show: %q", tt.Texts())
	}
	inOrder(t, tt, "Aaron", "ann")
	g.UpdateRow(3, func(d *fdoc) { d.kind = "Doc" })
	tt.Frame()
	inOrder(t, tt, "Aaron", "ann", "bob")
	if got := g.Selected(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Selected %v once the selected row shows again", got)
	}
	// AddRows and SetRows filter too.
	g.AddRows([]fdoc{{5, "cid", "Doc", 1}, {6, "dan", "Image", 1}})
	tt.Frame()
	if !tt.HasText("cid") || tt.HasText("dan") {
		t.Fatalf("AddRows: %q", tt.Texts())
	}
	g.SetRows([]fdoc{{7, "dan", "Doc", 1}, {8, "bob", "Sheet", 1}})
	fshows(t, tt, "dan")
}

// A column filter hides selected rows as the search does: they stay
// selected, and Selected and CountSelected leave them out while hidden.
func TestFilterSelectionAsSearch(t *testing.T) {
	type state struct {
		sel   []int
		count int
	}
	run := func(hide func(g *DataGridViewWidget[fdoc, int]), show func(g *DataGridViewWidget[fdoc, int])) (hidden, shown state, events int) {
		var n atomic.Int32
		g := fdocGrid().BottomPanel().OnSelectionChanged(func() { n.Add(1) })
		tt := NewTester(t, g, 800, 600)
		g.SetSelected(1, 3) // ann, bob
		tt.Settle()
		n.Store(0)
		hide(g)
		tt.Settle()
		hidden = state{g.Selected(), g.CountSelected()}
		show(g)
		tt.Settle()
		return hidden, state{g.Selected(), g.CountSelected()}, int(n.Load())
	}
	sh, ss, se := run(func(g *DataGridViewWidget[fdoc, int]) { g.Search("Name", "ann") },
		func(g *DataGridViewWidget[fdoc, int]) { g.Search("Name", "") })
	fh, fs, fe := run(func(g *DataGridViewWidget[fdoc, int]) { g.SetColumnFilter("Name", ColumnFilter{Contains: "ann"}) },
		func(g *DataGridViewWidget[fdoc, int]) { g.ClearFilters() })
	if fe != 2 {
		t.Fatalf("hiding then showing a selected row by a column filter ran OnSelectionChanged %d times, want 2", fe)
	}
	if !slices.Equal(sh.sel, []int{1}) || sh.count != 1 || !slices.Equal(ss.sel, []int{1, 3}) {
		t.Fatalf("the search: hidden %+v, shown %+v", sh, ss)
	}
	if !slices.Equal(fh.sel, sh.sel) || fh.count != sh.count || !slices.Equal(fs.sel, ss.sel) || fs.count != ss.count || fe != se {
		t.Fatalf("a column filter: hidden %+v, shown %+v, %d events; the search: %+v, %+v, %d events", fh, fs, fe, sh, ss, se)
	}
}

func TestFilterSetterDoesNotFireOnFilterChange(t *testing.T) {
	var fired atomic.Int32
	g := fdocGrid().SetFilterDropDowns(true).SetAutoFilterRow(true).SetFilterPanel(true).
		OnFilterChange(func() { fired.Add(1) })
	tt := NewTester(t, g, 800, 600)
	g.SetColumnFilter("Name", ColumnFilter{Contains: "an"})
	tt.Settle()
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	tt.Settle()
	tt.Find(ByRole(ui.RoleButton).Label("Filter Kind")).Click() // opening changes nothing
	g.SetColumnFilter("Name", ColumnFilter{Contains: "a"})
	tt.Settle()
	g.ClearFilters()
	tt.Settle()
	g.Search("Name", "b")
	tt.Settle()
	if n := fired.Load(); n != 0 {
		t.Fatalf("setters ran OnFilterChange %d times", n)
	}
}

func TestFilterRowSourceLoadedRowsOnly(t *testing.T) {
	src := NewRowSource(func(d *fdoc) int { return d.id })
	src.SetRows(fdocs()[:2]) // ann Doc, Hannah Sheet
	g := DataGridViewOf(src, fdocCols()...).Expand().SetFilterDropDowns(true).
		SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc", "Image"}})
	tt := NewTester(t, g, 800, 600)
	fshows(t, tt.Tester, "ann")
	tt.Find(ByRole(ui.RoleButton).Label("Filter Kind")).Click()
	if got := boxesY(tt); !slices.Equal(got, []string{"Doc", "Sheet"}) {
		t.Fatalf("the drop-down lists %q of the rows loaded", got)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	src.AddRows(fdocs()[2:]) // bob Doc, ANDY Image load
	fshows(t, tt.Tester, "ann", "bob", "ANDY")
	tt.Find(ByRole(ui.RoleButton).Label("Filter Kind")).Click()
	if got := boxesY(tt); !slices.Equal(got, []string{"Doc", "Image", "Sheet"}) {
		t.Fatalf("after more rows loaded the drop-down lists %q", got)
	}
}

func TestFilterFromGoroutine(t *testing.T) {
	g := fdocGrid().SetFilterDropDowns(true).SetAutoFilterRow(true).SetFilterPanel(true)
	tt := NewTester(t, g, 800, 600)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 50 {
			g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
			g.SetColumnFilter("Name", ColumnFilter{Contains: "n"})
			_ = g.ColumnFilter("Name")
			if i%2 == 0 {
				g.ClearFilters()
			}
			g.SetFilterPanel(i%3 != 0)
		}
		g.SetFilterPanel(true)
		g.ClearFilters()
		g.SetColumnFilter("Name", ColumnFilter{Contains: "bo"})
	}()
	for range 50 {
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	fshows(t, tt.Tester, "bob")
	if !tt.HasText(`Name contains "bo"`) || tt.Find(ByRole(ui.RoleTextField).Label("Filter Name")).Value() != "bo" {
		t.Fatalf("the last filter set from the goroutine does not show: %q", tt.Texts())
	}
}

// A grid whose filter setters were called to turn everything off builds the
// tree a grid without them does, and has no hook.
func TestFilterOffIsUnchanged(t *testing.T) {
	plain := fdocGrid().BottomPanel()
	off := fdocGrid().BottomPanel().SetFilterDropDowns(true).SetAutoFilterRow(true).SetFilterPanel(true).
		SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}}).
		SetFilterDropDowns(false).SetAutoFilterRow(false).SetFilterPanel(false).
		OnFilterChange(func() {}).ClearFilters().SetColumnFilter("Kind", ColumnFilter{})
	a, b := NewTester(t, plain, 800, 600), NewTester(t, off, 800, 600)
	na, nb := a.Nodes(), b.Nodes()
	if len(na) != len(nb) {
		t.Fatalf("%d nodes off, %d without filters", len(nb), len(na))
	}
	for i := range na {
		x, y := na[i], nb[i]
		x.ID, y.ID = 0, 0
		if x != y {
			t.Fatalf("node %d: %+v off, %+v without filters", i, y, x)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if off.filterHead != nil || off.filterWrap != nil || off.filterStrip != nil || off.filter != nil {
		t.Fatal("a grid with its filters off keeps a hook or a filter")
	}
	if plain.fl != nil {
		t.Fatal("a grid never filtered made its filters")
	}
}

func TestFilterAllocNothingAtRest(t *testing.T) {
	plain := tester(t, fdocGrid())
	with := tester(t, fdocGrid().SetFilterDropDowns(true).SetAutoFilterRow(true).SetFilterPanel(true).
		SetColumnFilter("Name", ColumnFilter{Contains: "a"}).SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}}))
	for range 10 {
		plain.Frame()
		with.Frame()
	}
	a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	t.Logf("a frame at rest allocates %v times with the filters, %v without", b, a)
	if b > a {
		t.Errorf("a frame at rest allocates %v times with the filters, %v without", b, a)
	}
}

// With the auto filter row, the empty state starts under the fields, not over
// them.
func TestFilterAutoRowEmptyStateUnderFields(t *testing.T) {
	g := fdocGrid().SetAutoFilterRow(true).SetEmptyState(EmptyState("Nothing here", "")).
		SetColumnFilter("Name", ColumnFilter{Contains: "zzz"})
	tt := NewTester(t, g, 800, 600)
	f := tt.Find(ByRole(ui.RoleTextField).Label("Filter Name")).Bounds()
	e := tt.Find(ByLabel("Nothing here")).Bounds()
	if e.Y < f.Y+f.H {
		t.Fatalf("the empty state at %+v covers the field at %+v", e, f)
	}
	// It is centered in the body, which starts a row lower than without the
	// auto filter row: the state moves down by half that row.
	plain := fdocGrid().SetEmptyState(EmptyState("Nothing here", "")).
		SetColumnFilter("Name", ColumnFilter{Contains: "zzz"})
	pe := NewTester(t, plain, 800, 600).Find(ByLabel("Nothing here")).Bounds()
	if d := e.Y - pe.Y; d < f.H/2-2 {
		t.Fatalf("the empty state is %v lower with the auto filter row than without, want about half the row (%v)", d, f.H/2)
	}
}

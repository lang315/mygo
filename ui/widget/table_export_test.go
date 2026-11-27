package widget

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// shownIs fails unless ShownCells gives header and rows.
func shownIs[T any, K comparable](t *testing.T, g *DataGridViewWidget[T, K], header []string, rows ...[]string) {
	t.Helper()
	h, rs := g.ShownCells()
	if !slices.Equal(h, header) {
		t.Fatalf("header %q, want %q", h, header)
	}
	if len(rs) != len(rows) {
		t.Fatalf("rows %q, want %q", rs, rows)
	}
	for i := range rows {
		if !slices.Equal(rs[i], rows[i]) {
			t.Fatalf("rows %q, want %q", rs, rows)
		}
	}
}

var (
	acct1 = []string{"name001", "0", "note1"}
	acct2 = []string{"name002", "37", "note2"}
)

// Hidden columns are left out, and the index column, which has no Text;
// the others come in the order the header shows them.
func TestShownCellsOrderAndColumns(t *testing.T) {
	g := threeColumns(2).IndexColumn()
	shownIs(t, g, []string{"Name", "Score", "Note"}, acct1, acct2)

	// A layout set before the first frame, not handed to ui yet.
	g.SetColumnsLayout(ui.TableLayout{Order: []string{"Note", "Name", "Score"}})
	shownIs(t, g, []string{"Note", "Name", "Score"}, []string{"note1", "name001", "0"}, []string{"note2", "name002", "37"})
	tt := tester(t, g)
	settleTable(tt)
	shownIs(t, g, []string{"Note", "Name", "Score"}, []string{"note1", "name001", "0"}, []string{"note2", "name002", "37"})

	// A layout naming some columns: they come first, the others after them
	// in theirs, as ui arranges them. Titles of no column shown, the index
	// column's and repeats are passed over.
	g.SetColumnsLayout(ui.TableLayout{Order: []string{"nope", "#", "Score", "Score"}})
	settleTable(tt)
	if h := headers(tt); !sameTitles(h, []string{"#", "Score", "Name", "Note"}) {
		t.Fatalf("ui shows %q", h)
	}
	shownIs(t, g, []string{"Score", "Name", "Note"}, []string{"0", "name001", "note1"}, []string{"37", "name002", "note2"})

	g.SetColumnVisible("Name", false)
	shownIs(t, g, []string{"Score", "Note"}, []string{"0", "note1"}, []string{"37", "note2"})
	g.SetColumnVisible("Name", true)
	shownIs(t, g, []string{"Score", "Name", "Note"}, []string{"0", "name001", "note1"}, []string{"37", "name002", "note2"})
}

// The user's move of a header, with no OnColumnsChange, is the order.
func TestShownCellsAfterTheUserMovesAColumn(t *testing.T) {
	g := threeColumns(1)
	tt := tester(t, g)
	note := find(t, tt, "Note")
	x, y := note.X+20, note.Y+note.H/2
	tt.Press(x, y)
	for _, d := range []float32{-5, -20, -60, -120, -180, -260} {
		tt.Move(x+d, y)
	}
	tt.Release(x-260, y)
	settleTable(tt)
	if h := headers(tt); !sameTitles(h, []string{"Note", "Name", "Score"}) {
		t.Fatalf("the move did not happen: headers %q", h)
	}
	shownIs(t, g, []string{"Note", "Name", "Score"}, []string{"note1", "name001", "0"})
	g.SetColumnVisible("Name", false)
	settleTable(tt)
	shownIs(t, g, []string{"Note", "Score"}, []string{"note1", "0"})
}

func TestShownCellsNoRowsAndNoColumns(t *testing.T) {
	shownIs(t, threeColumns(0), []string{"Name", "Score", "Note"})
	shownIs(t, threeColumns(1), []string{"Name", "Score", "Note"}, acct1)
	none := threeColumns(2, "Name", "Score", "Note")
	shownIs(t, none, []string{}, []string{}, []string{})
}

// Rows come as shown: sorted, filtered by the columns' filters and the
// search, grouped, with no line for a group's header.
func TestShownCellsSortedFilteredGrouped(t *testing.T) {
	g := fdocGrid()
	ann, hannah, bob, andy := []string{"ann", "Doc", "10"}, []string{"Hannah", "Sheet", "20"}, []string{"bob", "Doc", "30"}, []string{"ANDY", "Image", "40"}
	header := []string{"Name", "Kind", "Size"}
	shownIs(t, g, header, ann, hannah, bob, andy)
	g.Sort("Name", true)
	shownIs(t, g, header, bob, ann, hannah, andy)
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc", "Image"}})
	shownIs(t, g, header, bob, ann, andy)
	g.Search("Name", "an")
	shownIs(t, g, header, ann, andy)
	g.ClearFilters()
	shownIs(t, g, header, ann, hannah, andy)
	g.Search("Name", "")
	g.SetGroupBy(func(d *fdoc) string { return d.kind })
	shownIs(t, g, header, bob, ann, andy, hannah) // Doc, Image, Sheet
	g.SetGroupOrder("Sheet")
	shownIs(t, g, header, hannah, bob, ann, andy)

	// Shown, the grid's lines hold the groups' headers too.
	tt := tester(t, g)
	settleTable(tt)
	shownIs(t, g, header, hannah, bob, ann, andy)

	// A row an update moves into a filter's rows shows at once, in its group.
	g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc"}})
	shownIs(t, g, header, bob, ann)
	g.UpdateRow(2, func(d *fdoc) { d.kind = "Doc" })
	shownIs(t, g, header, bob, ann, []string{"Hannah", "Doc", "20"}) // "Hannah" < "ann"
	g.UpdateRow(1, func(d *fdoc) { d.kind = "Sheet" })
	shownIs(t, g, header, bob, []string{"Hannah", "Doc", "20"})
}

// A grid of a RowSource shows the rows loaded so far, by its own sort and
// search.
func TestShownCellsRowSourceLoaded(t *testing.T) {
	src := NewRowSource(func(d *fdoc) int { return d.id })
	src.SetRows(fdocs()[:2]) // ann Doc, Hannah Sheet
	header := []string{"Name", "Kind", "Size"}
	g := DataGridViewOf(src, fdocCols()...).Sort("Name", true)
	other := DataGridViewOf(src, fdocCols()...).Search("Kind", "doc")
	shownIs(t, g, header, []string{"ann", "Doc", "10"}, []string{"Hannah", "Sheet", "20"})
	shownIs(t, other, header, []string{"ann", "Doc", "10"})
	src.AddRows(fdocs()[2:]) // bob Doc, ANDY Image load
	shownIs(t, g, header, []string{"bob", "Doc", "30"}, []string{"ann", "Doc", "10"}, []string{"Hannah", "Sheet", "20"}, []string{"ANDY", "Image", "40"})
	shownIs(t, other, header, []string{"ann", "Doc", "10"}, []string{"bob", "Doc", "30"})
}

// ShownCells from another goroutine, while the window shows and the rows,
// filters, columns and layout change, races nothing (-race) and gives a
// row of cells per column.
func TestShownCellsFromGoroutine(t *testing.T) {
	g := fdocGrid().SetGroupBy(func(d *fdoc) string { return d.kind })
	tt := tester(t, g)
	tick := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	var bad string
	go func() {
		defer wg.Done()
		for i := range 60 {
			<-tick
			g.UpdateRow(1+i%4, func(d *fdoc) { d.size = i })
			g.SetColumnFilter("Kind", ColumnFilter{Values: []string{"Doc", "Sheet"}[:1+i%2]})
			g.SetColumnVisible("Kind", i%3 != 0)
			g.SetColumnsLayout(ui.TableLayout{Order: []string{"Size", "Name", "Kind"}[i%3:]})
			h, rs := g.ShownCells()
			for _, r := range rs {
				if len(r) != len(h) && bad == "" {
					bad = fmt.Sprintf("header %q, row %q", h, r)
				}
			}
		}
	}()
	for range 60 {
		tick <- struct{}{}
		tt.Frame()
	}
	wg.Wait()
	if bad != "" {
		t.Fatal(bad)
	}
	g.SetColumnFilter("Kind", ColumnFilter{}).SetColumnVisible("Kind", true).
		SetColumnsLayout(ui.TableLayout{Order: []string{"Size", "Name", "Kind"}})
	settleTable(tt)
	shownIs(t, g, []string{"Size", "Name", "Kind"},
		[]string{"56", "ann", "Doc"}, []string{"58", "bob", "Doc"}, []string{"59", "ANDY", "Image"}, []string{"57", "Hannah", "Sheet"})
}

// ShownCells changes nothing the grid shows: a grid it read draws what one
// never read does, and a frame at rest allocates nothing more.
func TestShownCellsOffIsUnchanged(t *testing.T) {
	grid := func() *DataGridViewWidget[acct, int] {
		return testTable(manyAccts(200)...).Sort("Score", false)
	}
	read := grid()
	plain, with := tester(t, grid()), tester(t, read)
	read.ShownCells()
	settleTable(plain)
	settleTable(with)
	if !samePixels(plain, with) {
		t.Fatal("the frames differ after ShownCells")
	}
	na, nb := plain.Nodes(), with.Nodes()
	if len(na) != len(nb) {
		t.Fatalf("%d nodes, %d after ShownCells", len(na), len(nb))
	}
	for i := range na {
		na[i].ID, nb[i].ID = 0, 0
		if na[i] != nb[i] {
			t.Fatalf("node %d: %+v, after ShownCells %+v", i, na[i], nb[i])
		}
	}
	read.ShownCells()
	plain.Frame()
	with.Frame()
	// The grid allocates at rest on main too: compare with one never read.
	if a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame); b > a {
		t.Errorf("a frame at rest after ShownCells allocates %v times, %v without", b, a)
	}
}

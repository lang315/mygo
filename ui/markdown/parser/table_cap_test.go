package parser

import (
	"strings"
	"testing"
)

// paddedCells counts the empty cells the parser added to rows shorter
// than their table, and the cells of every kind.
func paddedCells(d *Document) (padded, cells int) {
	for i := range d.blocks {
		b := &d.blocks[i]
		if b.Kind != TableCell {
			continue
		}
		cells++
		if b.segB == b.segA {
			padded++
		}
	}
	return padded, cells
}

// wideTable is a table of columns columns and rows rows of one cell.
func wideTable(columns, rows int) string {
	return strings.Repeat("|a", columns) + "|\n" + strings.Repeat("|-", columns) + "|\n" + strings.Repeat("|x\n", rows)
}

func TestTablePaddingIsBounded(t *testing.T) {
	const most = 500_000
	d := New(Default)
	d.Set(wideTable(3000, 5000))
	padded, _ := paddedCells(d)
	if padded > most || padded < most-3000 {
		t.Fatalf("%d padded cells, want at most %d and within a row of it", padded, most)
	}
	// The table ends where the cap is reached: the rows after it are a
	// paragraph.
	var kinds []Kind
	for i := range d.Children(0) {
		kinds = append(kinds, d.blocks[i].Kind)
	}
	if len(kinds) != 2 || kinds[0] != Table || kinds[1] != Paragraph {
		t.Errorf("top-level blocks %v, want a table and a paragraph", kinds)
	}
}

// TestTablePaddingCapIncremental checks that text appended a line at a
// time, and edits before and between tables, give the blocks a fresh
// parse gives, the count of empty cells included.
func TestTablePaddingCapIncremental(t *testing.T) {
	src := wideTable(3000, 100) + "\n" + wideTable(3000, 100) + "\nend\n"
	check := func(name string, d *Document) {
		t.Helper()
		fresh := Parse(d.Source())
		p, c := paddedCells(d)
		fp, fc := paddedCells(fresh)
		if p != fp || c != fc || d.padded != p || fresh.padded != fp || p > maxPaddedCells {
			t.Errorf("%s: %d padded of %d cells (counted %d); fresh parse %d of %d (counted %d)", name, p, c, d.padded, fp, fc, fresh.padded)
		}
		if d.Len() != fresh.Len() {
			t.Errorf("%s: %d blocks, fresh parse %d", name, d.Len(), fresh.Len())
		}
	}

	streamed := New(Default)
	for line := range strings.Lines(src) {
		streamed.Append(line)
	}
	check("appended", streamed)

	d := Parse(src)
	check("set", d)
	// A wider table before both: the second table is cut.
	d.Edit(0, 0, wideTable(3000, 100)+"\n")
	check("table inserted", d)
	// The first table removed again: the second is whole.
	d.Edit(0, len(wideTable(3000, 100))+1, "")
	check("table removed", d)
	d.Edit(0, 0, "a\n\n")
	check("paragraph inserted", d)

	// Far under the cap, an edit keeps the tables after it.
	small := Parse("a\n\n" + wideTable(10, 10) + "\nb\n\n" + wideTable(10, 10))
	small.Edit(0, 1, "c")
	check("small edit", small)
}

func TestFullTableKeepsAllCells(t *testing.T) {
	var b strings.Builder
	b.WriteString(strings.Repeat("|h", 1000) + "|\n" + strings.Repeat("|-", 1000) + "|\n")
	for range 100 {
		b.WriteString(strings.Repeat("|c", 1000) + "|\n")
	}
	d := Parse(b.String())
	padded, cells := paddedCells(d)
	if cells != 1000*101 || padded != 0 {
		t.Errorf("%d cells, %d padded; want %d, 0", cells, padded, 1000*101)
	}
}

package ui

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"
)

type tvRow struct{ a, b int }

var tvCompare = map[string]func(x, y tvRow) int{
	"a": func(x, y tvRow) int { return cmp.Compare(x.a, y.a) },
	"b": func(x, y tvRow) int { return cmp.Compare(x.b, y.b) },
}

// tvWant returns the rows a view shows, by brute force: those keep keeps,
// stably sorted by the column's comparator.
func tvWant(rows []tvRow, sort SortOrder, keep func(tvRow) bool) []int {
	var idx []int
	for i := range rows {
		if keep == nil || keep(rows[i]) {
			idx = append(idx, i)
		}
	}
	if f := tvCompare[sort.Column]; f != nil {
		slices.SortStableFunc(idx, func(i, j int) int {
			d := f(rows[i], rows[j])
			if sort.Descending {
				d = -d
			}
			return d
		})
	}
	return idx
}

func tvGot[T any](v *TableView[T]) []int {
	got := make([]int, v.Len())
	for i := range got {
		got[i] = v.Row(i)
	}
	return got
}

func TestTableViewMatchesSortAndFilter(t *testing.T) {
	for seed := range 1000 {
		r := rand.New(rand.NewPCG(uint64(seed), 0))
		row := func() tvRow { return tvRow{r.IntN(10), r.IntN(10)} }
		rows := make([]tvRow, r.IntN(60))
		for i := range rows {
			rows[i] = row()
		}
		v := NewTableView(tvCompare)
		var sort SortOrder
		var keep func(tvRow) bool
		for step := range 30 {
			switch r.IntN(7) {
			case 0, 1: // rows changed in place
				for range r.IntN(4) + 1 {
					if len(rows) == 0 {
						break
					}
					i := r.IntN(len(rows))
					rows[i] = row()
					v.Changed(i)
				}
			case 2: // rows appended, the slice reallocated or not
				for range r.IntN(3) + 1 {
					rows = append(rows, row())
				}
			case 3:
				sort = SortOrder{Column: []string{"a", "b", "none"}[r.IntN(3)], Descending: r.IntN(2) == 1}
			case 4:
				if r.IntN(3) == 0 {
					keep = nil
				} else {
					lim := r.IntN(10)
					keep = func(x tvRow) bool { return x.b < lim }
				}
				v.SetFilter(keep)
			case 5: // rows removed from the end
				rows = rows[:r.IntN(len(rows)+1)]
			case 6: // another slice of the same rows, one changed without Changed
				rows = slices.Clone(rows)
				if len(rows) > 0 {
					rows[r.IntN(len(rows))] = row()
				}
			}
			v.Sync(rows, sort)
			if got, want := tvGot(v), tvWant(rows, sort, keep); !slices.Equal(got, want) {
				t.Fatalf("seed %d, step %d: got %v, want %v", seed, step, got, want)
			}
		}
	}
}

func TestTableViewSliceReallocated(t *testing.T) {
	rows := make([]tvRow, 3, 3)
	rows[0], rows[1], rows[2] = tvRow{a: 3}, tvRow{a: 1}, tvRow{a: 2}
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	rows = append(rows, tvRow{a: 0}) // beyond the capacity: a new array
	rows[1].a = 9                    // and a change the view was not told of
	v.Sync(rows, sort)
	if got, want := tvGot(v), []int{3, 2, 0, 1}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTableViewChangedEdges(t *testing.T) {
	rows := make([]tvRow, 40)
	for i := range rows {
		rows[i] = tvRow{a: i}
	}
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	rows[5].a = 100
	v.Changed(5, 5, 5, -1, 40, 1000) // duplicates and rows past the end
	v.Sync(rows, sort)
	if got, want := tvGot(v), tvWant(rows, sort, nil); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTableViewVersion(t *testing.T) {
	rows := []tvRow{{a: 1, b: 5}, {a: 2, b: 5}, {a: 3, b: 5}}
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	ver := v.Version()
	v.Sync(rows, sort)
	if v.Version() != ver {
		t.Error("Sync without changes changed the version")
	}
	rows[1].b = 0 // not the column sorted by: same order
	v.Changed(1)
	v.Sync(rows, sort)
	if v.Version() != ver {
		t.Error("a change keeping the order changed the version")
	}
	rows[0].a = 9
	v.Changed(0)
	v.Sync(rows, sort)
	if v.Version() == ver {
		t.Error("a change of order kept the version")
	}
}

func TestTableViewColumnWithoutComparator(t *testing.T) {
	rows := []tvRow{{a: 3}, {a: 1}, {a: 2}}
	v := NewTableView(tvCompare)
	v.Sync(rows, SortOrder{Column: "size", Descending: true})
	if got := tvGot(v); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("got %v, want the slice's order", got)
	}
}

func tvBenchRows(n int) []tvRow {
	rows := make([]tvRow, n)
	for i := range rows {
		rows[i] = tvRow{a: i * 7919 % n, b: i % 7}
	}
	return rows
}

func BenchmarkTableViewSyncIdle(b *testing.B) {
	rows := tvBenchRows(100_000)
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	b.ReportAllocs()
	for b.Loop() {
		v.Sync(rows, sort)
	}
}

func BenchmarkTableViewChanged(b *testing.B) {
	for _, k := range []int{1, 10, 100} {
		b.Run("k="+strconv.Itoa(k), func(b *testing.B) {
			rows := tvBenchRows(100_000)
			v := NewTableView(tvCompare)
			sort := SortOrder{Column: "a"}
			v.Sync(rows, sort)
			seq := 0
			b.ReportAllocs()
			for b.Loop() {
				for j := range k {
					i := (seq*31 + j*977) % len(rows)
					rows[i].a = (rows[i].a*13 + 7) % len(rows)
					v.Changed(i)
				}
				seq++
				v.Sync(rows, sort)
			}
		})
	}
}

func BenchmarkTableViewResort(b *testing.B) {
	rows := tvBenchRows(100_000)
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	b.ReportAllocs()
	for b.Loop() {
		v.Reset()
		v.Sync(rows, sort)
	}
}

package ui

import (
	"cmp"
	"math"
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
		for step := range 60 {
			switch r.IntN(8) {
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
			case 7: // rows changed and a new filter, before one Sync
				for range r.IntN(4) + 1 {
					if len(rows) == 0 {
						break
					}
					i := r.IntN(len(rows))
					rows[i] = row()
					v.Changed(i)
				}
				if r.IntN(3) == 0 {
					keep = nil
				} else {
					lim := r.IntN(10)
					keep = func(x tvRow) bool { return x.b < lim }
				}
				v.SetFilter(keep)
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

func TestTableViewDescendingWithExtremeComparator(t *testing.T) {
	// -math.MinInt is math.MinInt: negating this comparator's result fails.
	extreme := map[string]func(x, y tvRow) int{"a": func(x, y tvRow) int {
		if x.a < y.a {
			return math.MinInt
		}
		if x.a > y.a {
			return math.MaxInt
		}
		return 0
	}}
	rows := []tvRow{{a: 2}, {a: 5}, {a: 1}, {a: 5}, {a: 3}}
	v := NewTableView(extreme)
	v.Sync(rows, SortOrder{Column: "a", Descending: true})
	// Rows equal keep the slice's order.
	if got, want := tvGot(v), []int{1, 3, 4, 0, 2}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	rows[2].a = 9
	v.Changed(2)
	v.Sync(rows, SortOrder{Column: "a", Descending: true})
	if got, want := tvGot(v), []int{2, 1, 3, 4, 0}; !slices.Equal(got, want) {
		t.Fatalf("after a change: got %v, want %v", got, want)
	}
}

func TestTableViewChangedPlacesIncrementally(t *testing.T) {
	var calls int
	counting := map[string]func(x, y tvRow) int{"a": func(x, y tvRow) int {
		calls++
		return cmp.Compare(x.a, y.a)
	}}
	rows := tvBenchRows(10_000)
	v := NewTableView(counting)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	if calls < 100_000 {
		t.Fatalf("a full sort made %d comparisons, expected over 100,000", calls)
	}
	calls = 0
	rows[1].a = 5000
	v.Changed(1)
	v.Sync(rows, sort)
	if calls >= 200 {
		t.Errorf("placing one changed row made %d comparisons, want fewer than 200", calls)
	}
	if got, want := tvGot(v), tvWant(rows, sort, nil); !slices.Equal(got, want) {
		t.Error("the incremental placement is not the sorted order")
	}
}

func TestTableViewVersionOfFilters(t *testing.T) {
	rows := []tvRow{{a: 1, b: 1}, {a: 2, b: 2}, {a: 3, b: 3}}
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	ver := v.Version()
	v.SetFilter(func(x tvRow) bool { return x.b < 3 })
	v.Sync(rows, sort)
	if v.Version() == ver {
		t.Error("a filter hiding a row kept the version")
	}
	ver = v.Version()
	rows[2].b = 1 // enters the filter
	v.Changed(2)
	v.Sync(rows, sort)
	if v.Version() == ver || v.Len() != 3 {
		t.Errorf("a row entering the filter: version kept or Len %d, want 3", v.Len())
	}
	ver = v.Version()
	rows[0].b = 9 // leaves it
	v.Changed(0)
	v.Sync(rows, sort)
	if v.Version() == ver || v.Len() != 2 {
		t.Errorf("a row leaving the filter: version kept or Len %d, want 2", v.Len())
	}
	ver = v.Version()
	v.SetFilter(func(x tvRow) bool { return x.b < 3 }) // the same rows
	v.Sync(rows, sort)
	if v.Version() != ver {
		t.Error("a filter keeping the same rows changed the version")
	}
}

// countingCompare is tvCompare's "a" counting its calls in *calls.
func countingCompare(calls *int) map[string]func(x, y tvRow) int {
	return map[string]func(x, y tvRow) int{"a": func(x, y tvRow) int {
		*calls++
		return cmp.Compare(x.a, y.a)
	}}
}

func TestTableViewFilterAfterNoneSortsNothing(t *testing.T) {
	var calls int
	rows := tvBenchRows(10_000)
	v := NewTableView(countingCompare(&calls))
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	calls = 0
	keep := func(x tvRow) bool { return x.b < 3 }
	v.SetFilter(keep)
	v.Sync(rows, sort)
	if calls != 0 {
		t.Errorf("a filter on sorted rows made %d comparisons, want none", calls)
	}
	if got, want := tvGot(v), tvWant(rows, sort, keep); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTableViewFilterThenSortInOneSync(t *testing.T) {
	rows := tvBenchRows(500)
	v := NewTableView(tvCompare)
	v.Sync(rows, SortOrder{Column: "a"})
	keep := func(x tvRow) bool { return x.b == 2 }
	v.SetFilter(keep)
	sort := SortOrder{Column: "b", Descending: true}
	v.Sync(rows, sort)
	if got, want := tvGot(v), tvWant(rows, sort, keep); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTableViewFilterDoesNotSortAgain(t *testing.T) {
	var calls int
	rows := tvBenchRows(10_000)
	v := NewTableView(countingCompare(&calls))
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	small := func(x tvRow) bool { return x.b < 3 }
	for _, step := range []struct {
		name string
		keep func(tvRow) bool
	}{
		{"a filter", small},
		{"another filter", func(x tvRow) bool { return x.b > 4 }},
		{"no filter", nil},
		{"the filter again", small},
	} {
		calls = 0
		v.SetFilter(step.keep)
		v.Sync(rows, sort)
		if calls != 0 {
			t.Errorf("%s: %d comparisons, want none", step.name, calls)
		}
		if got, want := tvGot(v), tvWant(rows, sort, step.keep); !slices.Equal(got, want) {
			t.Fatalf("%s: got %v, want %v", step.name, got, want)
		}
	}
}

func TestTableViewClearFilterAfterChanges(t *testing.T) {
	var calls int
	rows := slices.Grow(tvBenchRows(10_000), 16) // appends keep the array
	v := NewTableView(countingCompare(&calls))
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	keep := func(x tvRow) bool { return x.b < 3 }
	v.SetFilter(keep)
	v.Sync(rows, sort)
	for _, i := range []int{1, 2, 3, 500, 9999} { // rows shown and hidden
		rows[i].a = (rows[i].a + 4999) % len(rows)
		v.Changed(i)
	}
	rows = append(rows, tvRow{a: 7, b: 6}, tvRow{a: 8, b: 1})
	v.Sync(rows, sort)
	if got, want := tvGot(v), tvWant(rows, sort, keep); !slices.Equal(got, want) {
		t.Fatalf("filtered: got %v, want %v", got, want)
	}
	calls = 0
	v.SetFilter(nil)
	v.Sync(rows, sort)
	if calls != 0 {
		t.Errorf("clearing the filter made %d comparisons, want none", calls)
	}
	if got, want := tvGot(v), tvWant(rows, sort, nil); !slices.Equal(got, want) {
		t.Fatalf("cleared: got %v, want %v", got, want)
	}
}

func TestTableViewClearFilterAfterRealloc(t *testing.T) {
	rows := tvBenchRows(1000) // len == cap: the append below reallocates
	v := NewTableView(tvCompare)
	sort := SortOrder{Column: "a"}
	v.Sync(rows, sort)
	keep := func(x tvRow) bool { return x.b < 3 }
	v.SetFilter(keep)
	v.Sync(rows, sort)
	rows = append(rows, tvRow{a: 3, b: 0})
	v.Sync(rows, sort)
	v.SetFilter(nil)
	v.Sync(rows, sort)
	if got, want := tvGot(v), tvWant(rows, sort, nil); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
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
